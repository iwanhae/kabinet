package migration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/iwanhae/kabinet/internal/catalog"
	"github.com/iwanhae/kabinet/internal/compact"
)

func TestFloorHourUsUsesUTCHalfOpenBoundaries(t *testing.T) {
	for _, tc := range []struct{ input, want int64 }{
		{0, 0},
		{hourUs - 1, 0},
		{hourUs, hourUs},
		{-1, -hourUs},
		{-hourUs, -hourUs},
	} {
		if got := floorHourUs(tc.input); got != tc.want {
			t.Fatalf("floorHourUs(%d)=%d, want %d", tc.input, got, tc.want)
		}
	}
}

func TestRewriteLevelQueriesEveryHourThenSizeMergesAndResumes(t *testing.T) {
	dataDir := t.TempDir()
	p := migrationPaths(dataDir)
	if err := os.MkdirAll(filepath.Join(p.staging, "l1"), 0o755); err != nil {
		t.Fatal(err)
	}
	s := &state{
		BaseSeq: 1,
		Plans:   make(map[string]levelPlan),
		Hours:   make(map[string]hourRecord),
		Finals:  make(map[string]finalRecord),
	}
	legacy := []fileSnapshot{{Path: "old-a.parquet"}, {Path: "old-b.parquet"}}
	canonical := []string{"wal.parquet"}
	var migratedHours int
	originalRunner := runCompactor
	t.Cleanup(func() { runCompactor = originalRunner })
	runCompactor = func(_ context.Context, job compact.Job) (*compact.Result, error) {
		switch job.Mode {
		case compact.ModeInspect:
			return &compact.Result{Rows: 3, MinUs: 10, MaxUs: 2*hourUs + 10, Checksum: 11 ^ 22}, nil
		case compact.ModeMigrate:
			migratedHours++
			if len(job.LegacyInputs) != len(legacy) || len(job.Inputs) != len(canonical) {
				t.Fatalf("hour %d did not receive the complete level input set: %+v", migratedHours, job)
			}
			start := *job.RangeStartUs
			if *job.RangeEndUs != start+hourUs {
				t.Fatalf("query is not one half-open hour: [%d,%d)", start, *job.RangeEndUs)
			}
			result := &compact.Result{MinMs: start / 1000, MaxMs: (start + 10 + 999) / 1000}
			size := 1
			switch start {
			case 0:
				result.Rows, result.Checksum, size = 1, 11, 6
			case hourUs:
				result.Rows = 0
			case 2 * hourUs:
				result.Rows, result.Checksum, size = 2, 22, 6
			default:
				t.Fatalf("unexpected hour start %d", start)
			}
			if err := os.MkdirAll(filepath.Dir(job.Output), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(job.Output, make([]byte, size), 0o644); err != nil {
				t.Fatal(err)
			}
			return result, nil
		case compact.ModeRepack:
			if len(job.Inputs) != 2 {
				t.Fatalf("expected two chronological hour inputs, got %v", job.Inputs)
			}
			if err := os.WriteFile(job.Output, make([]byte, 9), 0o644); err != nil {
				t.Fatal(err)
			}
			return &compact.Result{Rows: 3, MinMs: 0, MaxMs: (2*hourUs + 1009) / 1000, Checksum: 11 ^ 22}, nil
		default:
			t.Fatalf("unexpected mode %q", job.Mode)
			return nil, nil
		}
	}

	cfg := Config{TempDir: filepath.Join(dataDir, "tmp"), MemoryLimitMB: 64}
	if err := rewriteLevel(t.Context(), cfg, p, s, catalog.L1, legacy, canonical, 10); err != nil {
		t.Fatal(err)
	}
	if migratedHours != 3 {
		t.Fatalf("expected one query for each of three hours, got %d", migratedHours)
	}
	if hour := s.Hours[hourKey(catalog.L1, hourUs)]; !hour.Empty {
		t.Fatalf("empty hour was not durably recorded: %+v", hour)
	}
	finals := levelFinals(s, catalog.L1)
	if len(finals) != 1 || finals[0].StartUs != 0 || finals[0].EndUs != 3*hourUs {
		t.Fatalf("unexpected final coverage: %+v", finals)
	}
	if _, err := os.Stat(finals[0].Output.Path); err != nil {
		t.Fatalf("final output missing: %v", err)
	}

	// The durable final range must make a restart skip all hourly queries,
	// even though its hourly input files were already removed.
	migratedHours = 0
	if err := rewriteLevel(t.Context(), cfg, p, s, catalog.L1, legacy, canonical, 10); err != nil {
		t.Fatal(err)
	}
	if migratedHours != 0 {
		t.Fatalf("resume unnecessarily repeated %d hourly queries", migratedHours)
	}
}

func TestCutoverResumesAndDeletesV1Sources(t *testing.T) {
	dataDir := t.TempDir()
	p := migrationPaths(dataDir)
	oldArchive := filepath.Join(dataDir, "archive", "l1")
	newArchive := filepath.Join(p.staging, "l1")
	if err := os.MkdirAll(oldArchive, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(newArchive, 0o755); err != nil {
		t.Fatal(err)
	}
	oldFile := filepath.Join(oldArchive, "old.parquet")
	newFile := filepath.Join(newArchive, "new.parquet")
	walFile := filepath.Join(dataDir, "wal", "events_1_2.jsonl.zst")
	if err := os.MkdirAll(filepath.Dir(walFile), 0o755); err != nil {
		t.Fatal(err)
	}
	for path, content := range map[string]string{oldFile: "old", newFile: "new", walFile: "wal"} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	s := &state{Phase: "validated", WAL: []fileSnapshot{{Path: walFile}}}
	if err := saveState(p.state, s); err != nil {
		t.Fatal(err)
	}
	if err := cutover(p, s); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "archive", "l1", "new.parquet")); err != nil {
		t.Fatalf("new archive not installed: %v", err)
	}
	if _, err := os.Stat(walFile); !os.IsNotExist(err) {
		t.Fatalf("migrated WAL was not deleted: %v", err)
	}
	if version, ok, err := readVersion(p.format); err != nil || !ok || version != formatVersion {
		t.Fatalf("format marker version=%d ok=%v err=%v", version, ok, err)
	}
	if _, err := os.Stat(p.backup); !os.IsNotExist(err) {
		t.Fatalf("v1 backup was not deleted: %v", err)
	}
}

func TestEnsureV2RejectsUnknownFormat(t *testing.T) {
	dataDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dataDir, "FORMAT_VERSION"), []byte("99\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := EnsureV2(t.Context(), Config{DataDir: dataDir}); err == nil {
		t.Fatal("expected unknown format version to fail closed")
	}
}

func TestDetachWALIsolatesInputsAndReconcilesRename(t *testing.T) {
	dataDir := t.TempDir()
	p := migrationPaths(dataDir)
	liveDir := filepath.Join(dataDir, "wal")
	if err := os.MkdirAll(liveDir, 0o755); err != nil {
		t.Fatal(err)
	}
	live := filepath.Join(liveDir, "events_1_2.jsonl.zst")
	if err := os.WriteFile(live, []byte("wal"), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(live)
	if err != nil {
		t.Fatal(err)
	}
	s := &state{WAL: []fileSnapshot{snapshot(live, info)}}
	if err := saveState(p.state, s); err != nil {
		t.Fatal(err)
	}
	if err := detachWAL(p, s); err != nil {
		t.Fatal(err)
	}
	isolated := filepath.Join(p.workspace, "wal-source", filepath.Base(live))
	if s.WAL[0].Path != isolated {
		t.Fatalf("WAL path was not updated: %s", s.WAL[0].Path)
	}
	if _, err := os.Stat(live); !os.IsNotExist(err) {
		t.Fatalf("live WAL was not removed: %v", err)
	}
	if _, err := os.Stat(isolated); err != nil {
		t.Fatalf("isolated WAL is missing: %v", err)
	}

	// Simulate a crash after rename but before the updated path was saved.
	s.WAL[0].Path = live
	if err := detachWAL(p, s); err != nil {
		t.Fatalf("failed to reconcile completed rename: %v", err)
	}
	if s.WAL[0].Path != isolated {
		t.Fatalf("reconciled WAL path was not updated: %s", s.WAL[0].Path)
	}
}
