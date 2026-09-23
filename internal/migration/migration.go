// Package migration snapshots and upgrades an on-disk event format while live
// ingestion continues in a separate WAL generation.
package migration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"syscall"
	"time"

	"github.com/iwanhae/kabinet/internal/catalog"
	"github.com/iwanhae/kabinet/internal/compact"
	"github.com/iwanhae/kabinet/internal/wal"
)

const (
	formatVersion = 2
	maxWALInputs  = 256
)

type Config struct {
	DataDir       string
	TempDir       string
	L1TargetBytes int64
	L2TargetBytes int64
	MemoryLimitMB int
}

type fileSnapshot struct {
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"modTime"`
}

type outputRecord struct {
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	ModTime  int64  `json:"modTime"`
	Rows     int64  `json:"rows"`
	MinMs    int64  `json:"minMs"`
	MaxMs    int64  `json:"maxMs"`
	Checksum uint64 `json:"checksum"`
}

type levelPlan struct {
	Rows     int64  `json:"rows"`
	MinUs    int64  `json:"minUs"`
	MaxUs    int64  `json:"maxUs"`
	StartUs  int64  `json:"startUs"`
	EndUs    int64  `json:"endUs"`
	Checksum uint64 `json:"checksum"`
}

type hourRecord struct {
	StartUs int64        `json:"startUs"`
	EndUs   int64        `json:"endUs"`
	Empty   bool         `json:"empty"`
	Output  outputRecord `json:"output,omitempty"`
}

type finalRecord struct {
	Level   int          `json:"level"`
	StartUs int64        `json:"startUs"`
	EndUs   int64        `json:"endUs"`
	Output  outputRecord `json:"output"`
}

type state struct {
	Phase     string                  `json:"phase"`
	BaseSeq   int64                   `json:"baseSeq"`
	L1        []fileSnapshot          `json:"l1"`
	L2        []fileSnapshot          `json:"l2"`
	WAL       []fileSnapshot          `json:"wal"`
	Completed map[string]outputRecord `json:"completed"`
	Plans     map[string]levelPlan    `json:"plans,omitempty"`
	Hours     map[string]hourRecord   `json:"hours,omitempty"`
	Finals    map[string]finalRecord  `json:"finals,omitempty"`
}

type paths struct {
	format    string
	state     string
	workspace string
	staging   string
	backup    string
}

// EnsureV2 performs or resumes the timestamp-column migration. It returns
// only when the live data tree is entirely v2 and safe to open for queries.
func EnsureV2(ctx context.Context, cfg Config) error {
	cfg = withDefaults(cfg)
	p := migrationPaths(cfg.DataDir)
	if version, ok, err := readVersion(p.format); err != nil {
		return err
	} else if ok {
		if version != formatVersion {
			return fmt.Errorf("unsupported data format version %d", version)
		}
		return cleanupCompleted(p)
	}

	s, err := loadOrCreateState(cfg, p)
	if err != nil {
		return err
	}
	if s == nil { // brand-new empty data directory
		return writeVersion(p.format)
	}
	if err := detachWAL(p, s); err != nil {
		return err
	}

	log.Printf("migration: timestamp v2 phase=%s archive_files=%d wal_files=%d", s.Phase, len(s.L1)+len(s.L2), len(s.WAL))
	if s.Phase == "building" {
		allSources := append([]fileSnapshot{}, s.L1...)
		allSources = append(allSources, s.L2...)
		allSources = append(allSources, s.WAL...)
		if err := validateSnapshots(allSources); err != nil {
			return err
		}
		if err := buildStaging(ctx, cfg, p, s); err != nil {
			return err
		}
		s.Phase = "validated"
		if err := saveState(p.state, s); err != nil {
			return err
		}
	}
	return cutover(p, s)
}

// PrepareV2 snapshots the migration inputs and moves existing sealed WAL
// segments out of the live WAL directory. It performs no Parquet rewriting,
// so the server can call it synchronously before starting ingestion and run
// EnsureV2 in the background afterward. The returned bool reports whether a
// migration remains to be completed.
func PrepareV2(cfg Config) (bool, error) {
	cfg = withDefaults(cfg)
	p := migrationPaths(cfg.DataDir)
	if version, ok, err := readVersion(p.format); err != nil {
		return false, err
	} else if ok {
		if version != formatVersion {
			return false, fmt.Errorf("unsupported data format version %d", version)
		}
		return false, cleanupCompleted(p)
	}
	s, err := loadOrCreateState(cfg, p)
	if err != nil {
		return false, err
	}
	if s == nil {
		return false, writeVersion(p.format)
	}
	if err := detachWAL(p, s); err != nil {
		return false, err
	}
	return true, nil
}

func withDefaults(cfg Config) Config {
	if cfg.MemoryLimitMB <= 0 {
		cfg.MemoryLimitMB = 512
	}
	if cfg.L1TargetBytes <= 0 {
		cfg.L1TargetBytes = 64 << 20
	}
	if cfg.L2TargetBytes <= 0 {
		cfg.L2TargetBytes = 340 << 20
	}
	return cfg
}

func migrationPaths(dataDir string) paths {
	workspace := filepath.Join(dataDir, ".migration-timestamp-v2")
	return paths{
		format:    filepath.Join(dataDir, "FORMAT_VERSION"),
		state:     filepath.Join(dataDir, ".migration-timestamp-v2.json"),
		workspace: workspace,
		staging:   filepath.Join(workspace, "archive"),
		backup:    filepath.Join(dataDir, ".archive-timestamp-v1"),
	}
}

func loadOrCreateState(cfg Config, p paths) (*state, error) {
	if raw, err := os.ReadFile(p.state); err == nil {
		var s state
		if err := json.Unmarshal(raw, &s); err != nil {
			return nil, fmt.Errorf("decode migration state: %w", err)
		}
		if s.Completed == nil {
			s.Completed = make(map[string]outputRecord)
		}
		if s.Plans == nil {
			s.Plans = make(map[string]levelPlan)
		}
		if s.Hours == nil {
			s.Hours = make(map[string]hourRecord)
		}
		if s.Finals == nil {
			s.Finals = make(map[string]finalRecord)
		}
		return &s, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	l1, err := snapshotGlob(filepath.Join(cfg.DataDir, "archive", "l1", "*.parquet"))
	if err != nil {
		return nil, err
	}
	l2, err := snapshotGlob(filepath.Join(cfg.DataDir, "archive", "l2", "*.parquet"))
	if err != nil {
		return nil, err
	}
	walDir := filepath.Join(cfg.DataDir, "wal")
	if err := wal.RecoverOpenSegments(walDir); err != nil {
		return nil, fmt.Errorf("recover WAL before migration: %w", err)
	}
	segments, err := wal.ListSealed(walDir)
	if err != nil {
		return nil, err
	}
	walFiles := make([]fileSnapshot, 0, len(segments))
	for _, seg := range segments {
		info, err := os.Stat(seg.Path)
		if err != nil {
			return nil, err
		}
		walFiles = append(walFiles, snapshot(seg.Path, info))
	}
	if len(l1)+len(l2)+len(walFiles) == 0 {
		return nil, nil
	}
	allFiles := append([]fileSnapshot{}, l1...)
	allFiles = append(allFiles, l2...)
	allFiles = append(allFiles, walFiles...)
	if err := preflightDisk(cfg.DataDir, allFiles); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(p.staging, "l1"), 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(p.staging, "l2"), 0o755); err != nil {
		return nil, err
	}
	s := &state{
		Phase: "building", BaseSeq: time.Now().UnixNano(), L1: l1, L2: l2, WAL: walFiles,
		Completed: make(map[string]outputRecord), Plans: make(map[string]levelPlan),
		Hours: make(map[string]hourRecord), Finals: make(map[string]finalRecord),
	}
	if err := saveState(p.state, s); err != nil {
		return nil, err
	}
	return s, nil
}

// detachWAL prevents a live writer from replacing a snapshotted segment with
// the same event-time filename while a long-running migration is reading it.
// Each rename is reconciled and persisted independently so a process crash at
// either side of the rename remains resumable.
func detachWAL(p paths, s *state) error {
	if len(s.WAL) == 0 {
		return nil
	}
	dir := filepath.Join(p.workspace, "wal-source")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for i := range s.WAL {
		source := s.WAL[i].Path
		target := filepath.Join(dir, filepath.Base(source))
		if filepath.Clean(filepath.Dir(source)) == filepath.Clean(dir) {
			continue
		}

		targetInfo, targetErr := os.Stat(target)
		_, sourceErr := os.Stat(source)
		switch {
		case sourceErr == nil && os.IsNotExist(targetErr):
			if err := os.Rename(source, target); err != nil {
				return fmt.Errorf("isolate migration WAL %s: %w", source, err)
			}
			targetInfo, targetErr = os.Stat(target)
		case os.IsNotExist(sourceErr) && targetErr == nil:
			// A prior process completed the rename before saving state.
		case sourceErr == nil && targetErr == nil:
			return fmt.Errorf("migration WAL exists at both %s and %s", source, target)
		default:
			if sourceErr != nil && !os.IsNotExist(sourceErr) {
				return sourceErr
			}
			if targetErr != nil {
				return fmt.Errorf("migration WAL missing from %s and %s", source, target)
			}
		}
		if targetErr != nil {
			return targetErr
		}
		s.WAL[i] = snapshot(target, targetInfo)
		if err := saveState(p.state, s); err != nil {
			return err
		}
	}
	return nil
}

func buildStaging(ctx context.Context, cfg Config, p paths, s *state) error {
	intermediate, err := convertWAL(ctx, cfg, p, s)
	if err != nil {
		return err
	}
	if err := rewriteLevel(ctx, cfg, p, s, catalog.L1, s.L1, intermediate, cfg.L1TargetBytes); err != nil {
		return err
	}
	if err := rewriteLevel(ctx, cfg, p, s, catalog.L2, s.L2, nil, cfg.L2TargetBytes); err != nil {
		return err
	}
	return nil
}

func convertWAL(ctx context.Context, cfg Config, p paths, s *state) ([]string, error) {
	dir := filepath.Join(p.workspace, "wal-parquet")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	var outputs []string
	for start, batchNo := 0, 0; start < len(s.WAL); batchNo++ {
		end, bytesIn := start, int64(0)
		for end < len(s.WAL) && end-start < maxWALInputs {
			bytesIn += s.WAL[end].Size
			end++
			if bytesIn >= cfg.L1TargetBytes {
				break
			}
		}
		key := fmt.Sprintf("wal-%06d", batchNo)
		if record, ok := validCompleted(s.Completed[key]); ok {
			outputs = append(outputs, record.Path)
			start = end
			continue
		}
		inputs := snapshotPaths(s.WAL[start:end])
		out := filepath.Join(dir, key+".parquet")
		if err := os.Remove(out); err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		result, err := runCompactor(ctx, compact.Job{Mode: compact.ModeConvert, Inputs: inputs, Output: out, TempDir: cfg.TempDir, MemoryLimitMB: cfg.MemoryLimitMB})
		if err != nil {
			return nil, fmt.Errorf("convert WAL batch %d: %w", batchNo, err)
		}
		record, err := completedRecord(out, result)
		if err != nil {
			return nil, err
		}
		s.Completed[key] = record
		if err := saveState(p.state, s); err != nil {
			return nil, err
		}
		outputs = append(outputs, out)
		log.Printf("migration: converted WAL batch=%d/%d rows=%d input_bytes=%d output_bytes=%d", batchNo+1, (len(s.WAL)+maxWALInputs-1)/maxWALInputs, result.Rows, bytesIn, record.Size)
		start = end
	}
	return outputs, nil
}

func rewriteLevel(ctx context.Context, cfg Config, p paths, s *state, level int, legacy []fileSnapshot, canonical []string, targetBytes int64) error {
	if len(legacy) == 0 && len(canonical) == 0 {
		return nil
	}
	planKey := fmt.Sprintf("l%d", level)
	plan, planned := s.Plans[planKey]
	if !planned {
		inspection, err := runCompactor(ctx, compact.Job{
			Mode: compact.ModeInspect, LegacyInputs: snapshotPaths(legacy), Inputs: canonical,
			TempDir: cfg.TempDir, MemoryLimitMB: cfg.MemoryLimitMB,
		})
		if err != nil {
			return fmt.Errorf("inspect l%d: %w", level, err)
		}
		plan = levelPlan{Rows: inspection.Rows, MinUs: inspection.MinUs, MaxUs: inspection.MaxUs, Checksum: inspection.Checksum}
		if plan.Rows > 0 {
			plan.StartUs = floorHourUs(plan.MinUs)
			plan.EndUs = floorHourUs(plan.MaxUs) + hourUs
		}
		s.Plans[planKey] = plan
		if err := saveState(p.state, s); err != nil {
			return err
		}
		log.Printf("migration: planned l%d rows=%d hours=%d", level, plan.Rows, (plan.EndUs-plan.StartUs)/hourUs)
	}
	if plan.Rows == 0 {
		return nil
	}

	if err := discardInvalidFinals(p, s, level); err != nil {
		return err
	}
	finals := levelFinals(s, level)
	coverageStart := plan.StartUs
	var pending []hourRecord
	var pendingBytes int64
	for startUs := plan.StartUs; startUs < plan.EndUs; {
		if final, ok := finalCovering(finals, startUs); ok {
			if err := cleanupCoveredHours(s, level, final.StartUs, final.EndUs); err != nil {
				return err
			}
			coverageStart = final.EndUs
			startUs = final.EndUs
			continue
		}

		endUs := startUs + hourUs
		key := hourKey(level, startUs)
		hour, ok := s.Hours[key]
		if !ok || hour.StartUs != startUs || hour.EndUs != endUs || !validHour(hour) {
			out := filepath.Join(p.workspace, "hours", fmt.Sprintf("l%d-%d.parquet", level, startUs))
			if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
				return err
			}
			if err := os.Remove(out); err != nil && !os.IsNotExist(err) {
				return err
			}
			result, err := runCompactor(ctx, compact.Job{
				Mode: compact.ModeMigrate, LegacyInputs: snapshotPaths(legacy), Inputs: canonical, Output: out,
				TempDir: cfg.TempDir, MemoryLimitMB: cfg.MemoryLimitMB, RangeStartUs: &startUs, RangeEndUs: &endUs,
			})
			if err != nil {
				return fmt.Errorf("rewrite l%d hour %s: %w", level, time.UnixMicro(startUs).UTC().Format(time.RFC3339), err)
			}
			hour = hourRecord{StartUs: startUs, EndUs: endUs, Empty: result.Rows == 0}
			if hour.Empty {
				if err := os.Remove(out); err != nil && !os.IsNotExist(err) {
					return err
				}
			} else {
				hour.Output, err = completedRecord(out, result)
				if err != nil {
					return err
				}
			}
			s.Hours[key] = hour
			if err := saveState(p.state, s); err != nil {
				return err
			}
			log.Printf("migration: rewrote l%d hour=%s rows=%d output_bytes=%d", level, time.UnixMicro(startUs).UTC().Format(time.RFC3339), result.Rows, hour.Output.Size)
		}
		if !hour.Empty {
			pending = append(pending, hour)
			pendingBytes += hour.Output.Size
		}
		if pendingBytes >= targetBytes {
			if err := finalizeHours(ctx, cfg, p, s, level, plan, coverageStart, endUs, pending); err != nil {
				return err
			}
			finals = levelFinals(s, level)
			coverageStart, pending, pendingBytes = endUs, nil, 0
		}
		startUs = endUs
	}
	if len(pending) > 0 {
		if err := finalizeHours(ctx, cfg, p, s, level, plan, coverageStart, plan.EndUs, pending); err != nil {
			return err
		}
	}

	var rows int64
	var checksum uint64
	for _, final := range levelFinals(s, level) {
		rows += final.Output.Rows
		checksum ^= final.Output.Checksum
	}
	if rows != plan.Rows {
		return fmt.Errorf("l%d row validation failed: wrote %d, expected %d", level, rows, plan.Rows)
	}
	if checksum != plan.Checksum {
		return fmt.Errorf("l%d checksum validation failed: wrote %d, expected %d", level, checksum, plan.Checksum)
	}
	return nil
}

const hourUs = int64(time.Hour / time.Microsecond)

func floorHourUs(us int64) int64 {
	q := us / hourUs
	if us < 0 && us%hourUs != 0 {
		q--
	}
	return q * hourUs
}

func hourKey(level int, startUs int64) string { return fmt.Sprintf("l%d-hour-%d", level, startUs) }

func finalKey(level int, startUs, endUs int64) string {
	return fmt.Sprintf("l%d-final-%d-%d", level, startUs, endUs)
}

func validHour(hour hourRecord) bool {
	if hour.Empty {
		return true
	}
	_, ok := validCompleted(hour.Output)
	return ok
}

func levelFinals(s *state, level int) []finalRecord {
	out := make([]finalRecord, 0)
	for _, record := range s.Finals {
		if record.Level == level {
			out = append(out, record)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartUs < out[j].StartUs })
	return out
}

func finalCovering(finals []finalRecord, hourStartUs int64) (finalRecord, bool) {
	for _, final := range finals {
		if final.StartUs <= hourStartUs && hourStartUs < final.EndUs {
			return final, true
		}
	}
	return finalRecord{}, false
}

func discardInvalidFinals(p paths, s *state, level int) error {
	changed := false
	for key, final := range s.Finals {
		if final.Level != level {
			continue
		}
		if _, ok := validCompleted(final.Output); ok {
			continue
		}
		if final.Output.Path != "" {
			if err := os.Remove(final.Output.Path); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
		delete(s.Finals, key)
		changed = true
	}
	if changed {
		return saveState(p.state, s)
	}
	return nil
}

func cleanupCoveredHours(s *state, level int, startUs, endUs int64) error {
	for hourStart := startUs; hourStart < endUs; hourStart += hourUs {
		hour, ok := s.Hours[hourKey(level, hourStart)]
		if !ok || hour.Output.Path == "" {
			continue
		}
		if err := os.Remove(hour.Output.Path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func finalizeHours(ctx context.Context, cfg Config, p paths, s *state, level int, plan levelPlan, startUs, endUs int64, hours []hourRecord) error {
	var expectedRows int64
	var expectedChecksum uint64
	inputs := make([]string, len(hours))
	for i, hour := range hours {
		inputs[i] = hour.Output.Path
		expectedRows += hour.Output.Rows
		expectedChecksum ^= hour.Output.Checksum
	}
	if len(inputs) == 0 {
		return nil
	}

	seq := s.BaseSeq + int64(level)*1_000_000 + (startUs-plan.StartUs)/hourUs
	var result *compact.Result
	var source string
	if len(inputs) == 1 {
		h := hours[0].Output
		result = &compact.Result{Rows: h.Rows, MinMs: h.MinMs, MaxMs: h.MaxMs, Checksum: h.Checksum}
		source = inputs[0]
	} else {
		source = filepath.Join(p.workspace, fmt.Sprintf("l%d-final-%d-%d.tmp.parquet", level, startUs, endUs))
		if err := os.Remove(source); err != nil && !os.IsNotExist(err) {
			return err
		}
		var err error
		result, err = runCompactor(ctx, compact.Job{
			Mode: compact.ModeRepack, Inputs: inputs, Output: source,
			TempDir: cfg.TempDir, MemoryLimitMB: cfg.MemoryLimitMB,
		})
		if err != nil {
			return fmt.Errorf("merge l%d hours [%d,%d): %w", level, startUs, endUs, err)
		}
	}
	if result.Rows != expectedRows || result.Checksum != expectedChecksum {
		return fmt.Errorf("l%d merged range validation failed: rows=%d/%d checksum=%d/%d", level, result.Rows, expectedRows, result.Checksum, expectedChecksum)
	}
	finalPath := filepath.Join(p.staging, fmt.Sprintf("l%d", level), catalog.FileName(time.UnixMilli(result.MinMs), time.UnixMilli(result.MaxMs), seq))
	if err := os.Rename(source, finalPath); err != nil {
		return err
	}
	record, err := completedRecord(finalPath, result)
	if err != nil {
		return err
	}
	key := finalKey(level, startUs, endUs)
	s.Finals[key] = finalRecord{Level: level, StartUs: startUs, EndUs: endUs, Output: record}
	if err := saveState(p.state, s); err != nil {
		return err
	}
	// State is durable before inputs are removed. A restart can therefore use
	// the final range as coverage and finish this cleanup safely.
	if err := cleanupCoveredHours(s, level, startUs, endUs); err != nil {
		return err
	}
	log.Printf("migration: finalized l%d hours=[%s,%s) rows=%d output_bytes=%d", level, time.UnixMicro(startUs).UTC().Format(time.RFC3339), time.UnixMicro(endUs).UTC().Format(time.RFC3339), record.Rows, record.Size)
	return nil
}

func cutover(p paths, s *state) error {
	if s.Phase != "validated" && s.Phase != "archive-backed-up" && s.Phase != "archive-installed" {
		return fmt.Errorf("invalid migration phase %q", s.Phase)
	}
	liveArchive := filepath.Join(filepath.Dir(p.format), "archive")
	if s.Phase == "validated" {
		if _, err := os.Stat(p.backup); os.IsNotExist(err) {
			if _, err := os.Stat(liveArchive); err == nil {
				if err := os.Rename(liveArchive, p.backup); err != nil {
					return err
				}
			}
		}
		s.Phase = "archive-backed-up"
		if err := saveState(p.state, s); err != nil {
			return err
		}
	}
	if s.Phase == "archive-backed-up" {
		if _, err := os.Stat(liveArchive); os.IsNotExist(err) {
			if err := os.Rename(p.staging, liveArchive); err != nil {
				return err
			}
		}
		s.Phase = "archive-installed"
		if err := saveState(p.state, s); err != nil {
			return err
		}
	}
	if s.Phase == "archive-installed" {
		for _, f := range s.WAL {
			if err := os.Remove(f.Path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove migrated WAL %s: %w", f.Path, err)
			}
		}
		if err := writeVersion(p.format); err != nil {
			return err
		}
	}
	log.Printf("migration: timestamp v2 cutover complete")
	return cleanupCompleted(p)
}

func cleanupCompleted(p paths) error {
	if err := os.RemoveAll(p.backup); err != nil {
		return err
	}
	if err := os.RemoveAll(p.workspace); err != nil {
		return err
	}
	if err := os.Remove(p.state); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

var runCompactor = runCompactorProcess

func runCompactorProcess(ctx context.Context, job compact.Job) (*compact.Result, error) {
	path, err := compactorPath()
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(job)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, path)
	cmd.Stdin = bytes.NewReader(payload)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	var result compact.Result
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func compactorPath() (string, error) {
	if path := os.Getenv("KABINET_COMPACTOR_PATH"); path != "" {
		return path, nil
	}
	if exe, err := os.Executable(); err == nil {
		path := filepath.Join(filepath.Dir(exe), "compactor")
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	if path, err := exec.LookPath("compactor"); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("compactor binary not found")
}

func readVersion(path string) (int, bool, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	version, err := strconv.Atoi(string(bytes.TrimSpace(raw)))
	return version, true, err
}

func writeVersion(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(strconv.Itoa(formatVersion)+"\n"), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func saveState(path string, s *state) error {
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func snapshotGlob(pattern string) ([]fileSnapshot, error) {
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	out := make([]fileSnapshot, 0, len(paths))
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		out = append(out, snapshot(path, info))
	}
	return out, nil
}

func snapshot(path string, info os.FileInfo) fileSnapshot {
	return fileSnapshot{Path: path, Size: info.Size(), ModTime: info.ModTime().UnixNano()}
}

func validateSnapshots(files []fileSnapshot) error {
	for _, f := range files {
		info, err := os.Stat(f.Path)
		if err != nil {
			return fmt.Errorf("migration source changed: %s: %w", f.Path, err)
		}
		if info.Size() != f.Size || info.ModTime().UnixNano() != f.ModTime {
			return fmt.Errorf("migration source changed: %s", f.Path)
		}
	}
	return nil
}

func validCompleted(record outputRecord) (outputRecord, bool) {
	if record.Path == "" {
		return outputRecord{}, false
	}
	info, err := os.Stat(record.Path)
	return record, err == nil && info.Size() == record.Size && info.ModTime().UnixNano() == record.ModTime
}

func completedRecord(path string, result *compact.Result) (outputRecord, error) {
	info, err := os.Stat(path)
	if err != nil {
		return outputRecord{}, err
	}
	return outputRecord{
		Path: path, Size: info.Size(), ModTime: info.ModTime().UnixNano(),
		Rows: result.Rows, MinMs: result.MinMs, MaxMs: result.MaxMs, Checksum: result.Checksum,
	}, nil
}

func snapshotPaths(files []fileSnapshot) []string {
	out := make([]string, len(files))
	for i, f := range files {
		out[i] = f.Path
	}
	return out
}

func snapshotSize(files []fileSnapshot) int64 {
	var size int64
	for _, f := range files {
		size += f.Size
	}
	return size
}

func preflightDisk(dataDir string, files []fileSnapshot) error {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(dataDir, &stat); err != nil {
		return err
	}
	available := int64(stat.Bavail) * int64(stat.Bsize)
	required := snapshotSize(files) + 2<<30
	if available < required {
		return fmt.Errorf("timestamp migration needs at least %d free bytes, only %d available", required, available)
	}
	return nil
}
