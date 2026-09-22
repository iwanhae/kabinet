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
	"github.com/iwanhae/kabinet/internal/utils"
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
	// StartupL2TargetBytes is the input size target for the one-time streaming
	// L2 repack. It is separate because legacy L2 files expand when rewritten.
	StartupL2TargetBytes int64
	// MaxL1Files merges early once this many L1 files have accumulated and
	// caps the input count of each startup merge job.
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
		c.CompactInterval = 6 * time.Hour
	}
	if c.ConvertThresholdBytes <= 0 {
		c.ConvertThresholdBytes = 64 << 20
	}
	if c.MergeTargetBytes <= 0 {
		c.MergeTargetBytes = 512 << 20
	}
	if c.StartupL2TargetBytes <= 0 {
		c.StartupL2TargetBytes = 340 << 20
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
	cfg    Config
	cat    *catalog.Catalog
	runJob func(context.Context, compact.Job) (*compact.Result, error)
}

// New creates a lifecycle manager.
func New(cat *catalog.Catalog, cfg Config) *Manager {
	cfg.withDefaults()
	return &Manager{cfg: cfg, cat: cat}
}

// CompactStartup consolidates legacy small files in place before the query
// and ingest layers start. L1 is deduplicated; L2 is streamed without another
// global dedup pass. A failure in one level does not prevent the other level
// from being attempted.
func (m *Manager) CompactStartup(ctx context.Context) error {
	levels := []struct {
		level       int
		targetBytes int64
		mode        string
	}{
		{level: catalog.L1, targetBytes: m.cfg.ConvertThresholdBytes, mode: compact.ModeMerge},
		{level: catalog.L2, targetBytes: m.cfg.StartupL2TargetBytes, mode: compact.ModeRepack},
	}

	var errs utils.MultiError
	for _, item := range levels {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := m.compactSmallLevel(ctx, item.level, item.targetBytes, item.mode); err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			log.Printf("lifecycle: startup compaction failed level=l%d: %v", item.level, err)
			errs.Add(fmt.Errorf("l%d startup compaction: %w", item.level, err))
		}
	}

	if len(errs.Errors) > 0 {
		return &errs
	}
	return nil
}

func (m *Manager) compactSmallLevel(ctx context.Context, level int, targetBytes int64, mode string) error {
	before := m.cat.Level(level)
	beforeBytes := catalogFilesSize(before)
	log.Printf("lifecycle: startup compaction starting level=l%d mode=%s files=%d bytes=%d target_bytes=%d",
		level, mode, len(before), beforeBytes, targetBytes)

	jobs := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		batch, batchSize := selectStartupMergeBatch(m.cat.Level(level), targetBytes, m.cfg.MaxL1Files)
		if len(batch) == 0 {
			break
		}

		log.Printf("lifecycle: startup merging level=l%d mode=%s inputs=%d input_bytes=%d target_bytes=%d",
			level, mode, len(batch), batchSize, targetBytes)
		if err := m.mergeParquetBatch(ctx, batch, level, mode, fmt.Sprintf("startup_%s_l%d", mode, level), true); err != nil {
			return fmt.Errorf("merge batch %d (%d files, %d bytes): %w", jobs+1, len(batch), batchSize, err)
		}
		jobs++
	}

	after := m.cat.Level(level)
	log.Printf("lifecycle: startup compaction complete level=l%d jobs=%d files_before=%d files_after=%d bytes_before=%d bytes_after=%d",
		level, jobs, len(before), len(after), beforeBytes, catalogFilesSize(after))
	return nil
}

// Run blocks until ctx is cancelled, executing one maintenance pass per tick.
func (m *Manager) Run(ctx context.Context) {
	log.Printf("lifecycle: starting. compact_interval=%s compact_target_bytes=%d merge_target_bytes=%d startup_l2_target_bytes=%d max_l1_files=%d storage_limit_bytes=%d",
		m.cfg.CompactInterval, m.cfg.ConvertThresholdBytes, m.cfg.MergeTargetBytes, m.cfg.StartupL2TargetBytes, m.cfg.MaxL1Files, m.cfg.StorageLimitBytes)

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
	backlogAge := sealedBacklogAge(time.Now(), segments)
	if totalSize < m.cfg.ConvertThresholdBytes && backlogAge < m.cfg.CompactInterval {
		return nil
	}

	batch, batchSize := selectConvertBatch(segments, m.cfg.ConvertThresholdBytes)
	inputs := make([]string, len(batch))
	for i, seg := range batch {
		inputs[i] = seg.Path
	}

	seq := m.cat.NextSeq()
	tmpOut := filepath.Join(m.cfg.TempDir, fmt.Sprintf("convert_%d.parquet", seq))
	defer os.Remove(tmpOut)

	log.Printf("lifecycle: converting %d wal segments (%d bytes) to parquet; backlog remaining=%d segments (%d bytes)",
		len(batch), batchSize, len(segments)-len(batch), totalSize-batchSize)
	result, err := m.executeCompactor(ctx, compact.Job{
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

	for _, seg := range batch {
		if err := os.Remove(seg.Path); err != nil {
			log.Printf("lifecycle: failed to remove converted segment %s: %v (duplicates until next merge)", seg.Path, err)
		}
	}
	return nil
}

// sealedBacklogAge returns how long the oldest sealed file has actually been
// waiting. Segment.Start is event time and may be arbitrarily old after a
// Kubernetes relist, so it must not be used as the compaction wait time.
func sealedBacklogAge(now time.Time, segments []wal.Segment) time.Duration {
	oldest := segments[0].SealedAt
	for _, seg := range segments[1:] {
		if seg.SealedAt.Before(oldest) {
			oldest = seg.SealedAt
		}
	}
	return now.Sub(oldest)
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

	batch, batchSize := selectMergeBatch(l1, m.cfg.MergeTargetBytes, m.cfg.MaxL1Files)
	log.Printf("lifecycle: merging %d l1 files (%d bytes) into l2; backlog remaining=%d files (%d bytes)",
		len(batch), batchSize, len(l1)-len(batch), totalSize-batchSize)
	return m.mergeParquetBatch(ctx, batch, catalog.L2, compact.ModeMerge, "merge", false)
}

// mergeParquetBatch merges archive files, publishes the result to outputLevel,
// then retires the inputs. Startup compaction deletes immediately because no
// query can be in flight; regular lifecycle merges retain the deletion grace.
func (m *Manager) mergeParquetBatch(ctx context.Context, batch []catalog.File, outputLevel int, mode string, tempPrefix string, deleteImmediately bool) error {
	inputs := make([]string, len(batch))
	for i, f := range batch {
		inputs[i] = f.Path
	}

	seq := m.cat.NextSeq()
	tmpOut := filepath.Join(m.cfg.TempDir, fmt.Sprintf("%s_%d.parquet", tempPrefix, seq))
	defer os.Remove(tmpOut)

	result, err := m.executeCompactor(ctx, compact.Job{
		Mode:          mode,
		Inputs:        inputs,
		Output:        tmpOut,
		TempDir:       m.cfg.TempDir,
		MemoryLimitMB: m.cfg.MemoryLimitMB,
	})
	if err != nil {
		return err
	}

	if result.Rows > 0 {
		if err := m.publish(tmpOut, outputLevel, result, seq); err != nil {
			return err
		}
	}

	for _, f := range batch {
		m.cat.Remove(f.Path)
		if deleteImmediately {
			if err := os.Remove(f.Path); err != nil && !os.IsNotExist(err) {
				log.Printf("lifecycle: startup compaction failed to delete %s: %v", f.Path, err)
			}
		} else {
			m.scheduleDelete(f.Path)
		}
	}
	return nil
}

func (m *Manager) executeCompactor(ctx context.Context, job compact.Job) (*compact.Result, error) {
	if m.runJob != nil {
		return m.runJob(ctx, job)
	}
	return m.runCompactor(ctx, job)
}

func selectConvertBatch(segments []wal.Segment, targetBytes int64) ([]wal.Segment, int64) {
	var size int64
	count := 0
	for count < len(segments) && count < maxSegmentsPerConvert {
		size += segments[count].Size
		count++
		if size >= targetBytes {
			break
		}
	}
	return segments[:count], size
}

func selectMergeBatch(files []catalog.File, targetBytes int64, maxFiles int) ([]catalog.File, int64) {
	var size int64
	count := 0
	for count < len(files) && count < maxFiles {
		size += files[count].Size
		count++
		if size >= targetBytes {
			break
		}
	}
	return files[:count], size
}

// selectStartupMergeBatch returns the oldest mergeable run of small files.
// Files at or above targetBytes are barriers so the resulting time range does
// not span an already well-sized file. A trailing run is merged when it has at
// least two files even if it does not reach the target.
func selectStartupMergeBatch(files []catalog.File, targetBytes int64, maxFiles int) ([]catalog.File, int64) {
	if targetBytes <= 0 || maxFiles < 2 {
		return nil, 0
	}

	batch := make([]catalog.File, 0, maxFiles)
	var size int64
	for _, file := range files {
		if file.Size >= targetBytes {
			if len(batch) >= 2 {
				return batch, size
			}
			batch = batch[:0]
			size = 0
			continue
		}

		batch = append(batch, file)
		size += file.Size
		if size >= targetBytes || len(batch) >= maxFiles {
			return batch, size
		}
	}

	if len(batch) >= 2 {
		return batch, size
	}
	return nil, 0
}

func catalogFilesSize(files []catalog.File) int64 {
	var size int64
	for _, file := range files {
		size += file.Size
	}
	return size
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
