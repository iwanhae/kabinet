// Package lifecycle orchestrates the data-management layer: it decides when
// to convert sealed WAL segments to Parquet, when to merge small Parquet
// files, and which files to drop for retention. The heavy work itself always
// runs in the compactor subprocess (see internal/compact); this package only
// schedules it and updates the catalog. It communicates with the ingest and
// query layers exclusively through the filesystem and the catalog.
package lifecycle

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/iwanhae/kabinet/internal/catalog"
	"github.com/iwanhae/kabinet/internal/compact"
	"github.com/iwanhae/kabinet/internal/wal"
)

// Config tunes the lifecycle manager.
type Config struct {
	WalDir  string
	TempDir string
	// StorageLimitBytes bounds wal + archive disk usage; oldest archive files
	// are deleted first when exceeded.
	StorageLimitBytes int64
	// CompactInterval is how long sealed segments may wait before conversion.
	CompactInterval time.Duration
	// ConvertThresholdBytes converts early once the sealed backlog reaches this size.
	ConvertThresholdBytes int64
	// MergeTargetBytes merges L1 files into one L2 file once their combined
	// size reaches this target.
	MergeTargetBytes int64
	// MaxL1Files merges early once this many L1 files have accumulated,
	// keeping per-query file counts bounded on low-traffic clusters.
	MaxL1Files int
	// MemoryLimitMB is passed to the compactor subprocess (DuckDB memory_limit).
	MemoryLimitMB int
	// DeleteGrace delays physical deletion after a file leaves the catalog,
	// so in-flight queries holding the file open can finish.
	DeleteGrace time.Duration
	// TickInterval is the scheduler cadence.
	TickInterval time.Duration
}

func (c *Config) withDefaults() {
	if c.CompactInterval <= 0 {
		c.CompactInterval = 10 * time.Minute
	}
	if c.ConvertThresholdBytes <= 0 {
		c.ConvertThresholdBytes = 32 << 20
	}
	if c.MergeTargetBytes <= 0 {
		c.MergeTargetBytes = 128 << 20
	}
	if c.MaxL1Files <= 0 {
		c.MaxL1Files = 96
	}
	if c.MemoryLimitMB <= 0 {
		c.MemoryLimitMB = 512
	}
	if c.DeleteGrace <= 0 {
		c.DeleteGrace = time.Minute
	}
	if c.TickInterval <= 0 {
		c.TickInterval = time.Minute
	}
}

// maxSegmentsPerConvert caps one conversion batch to bound subprocess input.
const maxSegmentsPerConvert = 256

// Manager runs the data-management loop.
type Manager struct {
	cfg Config
	cat *catalog.Catalog
}

// New creates a lifecycle manager.
func New(cat *catalog.Catalog, cfg Config) *Manager {
	cfg.withDefaults()
	return &Manager{cfg: cfg, cat: cat}
}

// Run blocks until ctx is cancelled, executing one maintenance pass per tick.
func (m *Manager) Run(ctx context.Context) {
	log.Printf("lifecycle: starting. compact_interval=%s merge_target_bytes=%d storage_limit_bytes=%d",
		m.cfg.CompactInterval, m.cfg.MergeTargetBytes, m.cfg.StorageLimitBytes)

	ticker := time.NewTicker(m.cfg.TickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := m.convert(ctx); err != nil {
				log.Printf("lifecycle: convert failed (will retry): %v", err)
			}
			if err := m.merge(ctx); err != nil {
				log.Printf("lifecycle: merge failed (will retry): %v", err)
			}
			if err := m.retain(); err != nil {
				log.Printf("lifecycle: retention failed (will retry): %v", err)
			}
		case <-ctx.Done():
			log.Println("lifecycle: stopping.")
			return
		}
	}
}

// convert turns sealed WAL segments into one L1 Parquet file. Segments are
// deleted only after the Parquet file is registered, so a crash anywhere in
// between costs duplicates (removed by the next merge), never data.
func (m *Manager) convert(ctx context.Context) error {
	segments, err := wal.ListSealed(m.cfg.WalDir)
	if err != nil {
		return err
	}
	if len(segments) == 0 {
		return nil
	}

	var totalSize int64
	for _, seg := range segments {
		totalSize += seg.Size
	}
	backlogAge := time.Since(segments[0].Start)
	if totalSize < m.cfg.ConvertThresholdBytes && backlogAge < m.cfg.CompactInterval {
		return nil
	}

	if len(segments) > maxSegmentsPerConvert {
		segments = segments[:maxSegmentsPerConvert]
	}
	inputs := make([]string, len(segments))
	for i, seg := range segments {
		inputs[i] = seg.Path
	}

	seq := m.cat.NextSeq()
	tmpOut := filepath.Join(m.cfg.TempDir, fmt.Sprintf("convert_%d.parquet", seq))
	defer os.Remove(tmpOut)

	log.Printf("lifecycle: converting %d wal segments (%d bytes) to parquet", len(segments), totalSize)
	result, err := m.runCompactor(ctx, compact.Job{
		Mode:          compact.ModeConvert,
		Inputs:        inputs,
		Output:        tmpOut,
		TempDir:       m.cfg.TempDir,
		MemoryLimitMB: m.cfg.MemoryLimitMB,
	})
	if err != nil {
		return err
	}

	if result.Rows > 0 {
		if err := m.publish(tmpOut, catalog.L1, result, seq); err != nil {
			return err
		}
	}

	for _, seg := range segments {
		if err := os.Remove(seg.Path); err != nil {
			log.Printf("lifecycle: failed to remove converted segment %s: %v (duplicates until next merge)", seg.Path, err)
		}
	}
	return nil
}

// merge combines the accumulated L1 files into a single deduplicated L2 file.
func (m *Manager) merge(ctx context.Context) error {
	l1 := m.cat.Level(catalog.L1)
	if len(l1) == 0 {
		return nil
	}

	var totalSize int64
	for _, f := range l1 {
		totalSize += f.Size
	}
	if totalSize < m.cfg.MergeTargetBytes && len(l1) < m.cfg.MaxL1Files {
		return nil
	}

	inputs := make([]string, len(l1))
	for i, f := range l1 {
		inputs[i] = f.Path
	}

	seq := m.cat.NextSeq()
	tmpOut := filepath.Join(m.cfg.TempDir, fmt.Sprintf("merge_%d.parquet", seq))
	defer os.Remove(tmpOut)

	log.Printf("lifecycle: merging %d l1 files (%d bytes) into l2", len(l1), totalSize)
	result, err := m.runCompactor(ctx, compact.Job{
		Mode:          compact.ModeMerge,
		Inputs:        inputs,
		Output:        tmpOut,
		TempDir:       m.cfg.TempDir,
		MemoryLimitMB: m.cfg.MemoryLimitMB,
	})
	if err != nil {
		return err
	}

	if result.Rows > 0 {
		if err := m.publish(tmpOut, catalog.L2, result, seq); err != nil {
			return err
		}
	}

	for _, f := range l1 {
		m.cat.Remove(f.Path)
		m.scheduleDelete(f.Path)
	}
	return nil
}

// publish moves a compactor output into its level directory and registers it.
func (m *Manager) publish(tmpPath string, level int, result *compact.Result, seq int64) error {
	min := time.UnixMilli(result.MinMs)
	max := time.UnixMilli(result.MaxMs)
	finalPath := filepath.Join(m.cat.LevelDir(level), catalog.FileName(min, max, seq))

	info, err := os.Stat(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to stat compactor output: %w", err)
	}
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return fmt.Errorf("failed to publish %s: %w", finalPath, err)
	}

	m.cat.Add(catalog.File{
		Path:  finalPath,
		Level: level,
		Min:   min,
		Max:   max,
		Size:  info.Size(),
		Rows:  result.Rows,
	})
	log.Printf("lifecycle: published %s (%d rows, %d bytes)", finalPath, result.Rows, info.Size())
	return nil
}

// retain deletes the oldest archive files while wal+archive exceed the limit.
func (m *Manager) retain() error {
	if m.cfg.StorageLimitBytes <= 0 {
		return nil
	}

	total := m.cat.TotalSize() + dirSize(m.cfg.WalDir)
	if total <= m.cfg.StorageLimitBytes {
		return nil
	}

	for _, f := range m.cat.All() {
		if total <= m.cfg.StorageLimitBytes {
			break
		}
		m.cat.Remove(f.Path)
		m.scheduleDelete(f.Path)
		total -= f.Size
		log.Printf("lifecycle: retention dropped %s (%d bytes). total now %d bytes", f.Path, f.Size, total)
	}
	return nil
}

// scheduleDelete removes a file after the grace period so in-flight queries
// that already opened it can finish. If the process dies first, the file is
// re-discovered by the catalog scan on restart and deleted again.
func (m *Manager) scheduleDelete(path string) {
	time.AfterFunc(m.cfg.DeleteGrace, func() {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			log.Printf("lifecycle: failed to delete %s: %v", path, err)
		}
	})
}

func dirSize(dir string) int64 {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	var total int64
	for _, entry := range entries {
		if info, err := entry.Info(); err == nil {
			total += info.Size()
		}
	}
	return total
}
