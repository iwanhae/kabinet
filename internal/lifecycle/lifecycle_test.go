package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/iwanhae/kabinet/internal/catalog"
	"github.com/iwanhae/kabinet/internal/compact"
	"github.com/iwanhae/kabinet/internal/wal"
)

func TestSealedBacklogAgeUsesSealTimeNotEventTime(t *testing.T) {
	now := time.Now()
	segments := []wal.Segment{
		{Start: now.Add(-24 * time.Hour), SealedAt: now.Add(-2 * time.Minute)},
		{Start: now.Add(-48 * time.Hour), SealedAt: now.Add(-5 * time.Minute)},
	}

	if age := sealedBacklogAge(now, segments); age != 5*time.Minute {
		t.Fatalf("expected 5 minute sealed backlog age, got %s", age)
	}
}

func TestSelectConvertBatchBoundsInput(t *testing.T) {
	segments := []wal.Segment{{Size: 10}, {Size: 15}, {Size: 20}}
	batch, size := selectConvertBatch(segments, 24)
	if len(batch) != 2 || size != 25 {
		t.Fatalf("expected two segments totaling 25 bytes, got %d totaling %d", len(batch), size)
	}

	many := make([]wal.Segment, maxSegmentsPerConvert+1)
	for i := range many {
		many[i].Size = 1
	}
	batch, size = selectConvertBatch(many, int64(len(many)+1))
	if len(batch) != maxSegmentsPerConvert || size != maxSegmentsPerConvert {
		t.Fatalf("expected segment cap %d, got %d totaling %d", maxSegmentsPerConvert, len(batch), size)
	}
}

func TestSelectMergeBatchBoundsInput(t *testing.T) {
	files := []catalog.File{{Size: 40}, {Size: 40}, {Size: 40}, {Size: 40}}
	batch, size := selectMergeBatch(files, 100, 96)
	if len(batch) != 3 || size != 120 {
		t.Fatalf("expected three files totaling 120 bytes, got %d totaling %d", len(batch), size)
	}

	batch, size = selectMergeBatch(files, 1_000, 2)
	if len(batch) != 2 || size != 80 {
		t.Fatalf("expected file cap 2, got %d totaling %d", len(batch), size)
	}
}

func TestSelectStartupMergeBatch(t *testing.T) {
	files := func(sizes ...int64) []catalog.File {
		out := make([]catalog.File, len(sizes))
		for i, size := range sizes {
			out[i] = catalog.File{Path: fmt.Sprintf("file-%d", i), Size: size}
		}
		return out
	}

	tests := []struct {
		name      string
		files     []catalog.File
		target    int64
		maxFiles  int
		wantPaths []string
		wantSize  int64
	}{
		{
			name:      "accumulates three files to target",
			files:     files(20, 20, 30, 10),
			target:    64,
			maxFiles:  96,
			wantPaths: []string{"file-0", "file-1", "file-2"},
			wantSize:  70,
		},
		{
			name:      "large file separates small runs",
			files:     files(20, 100, 20, 20),
			target:    100,
			maxFiles:  96,
			wantPaths: []string{"file-2", "file-3"},
			wantSize:  40,
		},
		{
			name:     "isolated small files are retained",
			files:    files(20, 100, 20),
			target:   100,
			maxFiles: 96,
		},
		{
			name:      "input count bounds a batch",
			files:     files(10, 10, 10),
			target:    100,
			maxFiles:  2,
			wantPaths: []string{"file-0", "file-1"},
			wantSize:  20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batch, size := selectStartupMergeBatch(tt.files, tt.target, tt.maxFiles)
			var paths []string
			for _, file := range batch {
				paths = append(paths, file.Path)
			}
			if !reflect.DeepEqual(paths, tt.wantPaths) || size != tt.wantSize {
				t.Fatalf("got paths=%v size=%d, want paths=%v size=%d", paths, size, tt.wantPaths, tt.wantSize)
			}
		})
	}
}

func TestCompactStartupMergesInPlaceAndRemovesInputs(t *testing.T) {
	archiveDir := t.TempDir()
	tmpDir := t.TempDir()
	base := time.UnixMilli(1_700_000_000_000)
	l1Inputs := writeCatalogFiles(t, archiveDir, catalog.L1, base, []int{40, 40, 40, 40, 40})
	writeCatalogFiles(t, archiveDir, catalog.L2, base.Add(time.Hour), []int{60, 60, 60, 60, 60})

	cat, err := catalog.Open(archiveDir)
	if err != nil {
		t.Fatal(err)
	}
	m := New(cat, Config{
		TempDir:               tmpDir,
		ConvertThresholdBytes: 100,
		MergeTargetBytes:      200,
		MaxL1Files:            96,
	})

	var inputCounts []int
	jobNumber := 0
	m.runJob = func(_ context.Context, job compact.Job) (*compact.Result, error) {
		inputCounts = append(inputCounts, len(job.Inputs))
		var size int64
		for _, input := range job.Inputs {
			info, err := os.Stat(input)
			if err != nil {
				return nil, err
			}
			size += info.Size()
		}
		if err := os.WriteFile(job.Output, make([]byte, size), 0o644); err != nil {
			return nil, err
		}
		jobNumber++
		min := base.Add(time.Duration(jobNumber) * 24 * time.Hour)
		return &compact.Result{Rows: 1, MinMs: min.UnixMilli(), MaxMs: min.Add(time.Minute).UnixMilli()}, nil
	}

	if err := m.CompactStartup(context.Background()); err != nil {
		t.Fatalf("startup compaction failed: %v", err)
	}
	if !reflect.DeepEqual(inputCounts, []int{3, 2, 4}) {
		t.Fatalf("unexpected startup batches: %v", inputCounts)
	}
	if got := len(cat.Level(catalog.L1)); got != 2 {
		t.Fatalf("expected 2 compacted l1 files, got %d", got)
	}
	if got := len(cat.Level(catalog.L2)); got != 2 {
		t.Fatalf("expected one compacted and one original l2 file, got %d total", got)
	}
	for _, path := range l1Inputs {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("expected startup input %s to be deleted, stat err=%v", path, err)
		}
	}

	reopened, err := catalog.Open(archiveDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(reopened.Level(catalog.L1)) != 2 || len(reopened.Level(catalog.L2)) != 2 {
		t.Fatalf("reopened catalog disagrees: l1=%d l2=%d", len(reopened.Level(catalog.L1)), len(reopened.Level(catalog.L2)))
	}
}

func TestCompactStartupContinuesWithL2AfterL1Failure(t *testing.T) {
	archiveDir := t.TempDir()
	tmpDir := t.TempDir()
	base := time.UnixMilli(1_700_000_000_000)
	l1Inputs := writeCatalogFiles(t, archiveDir, catalog.L1, base, []int{40, 40})
	writeCatalogFiles(t, archiveDir, catalog.L2, base.Add(time.Hour), []int{60, 60})

	cat, err := catalog.Open(archiveDir)
	if err != nil {
		t.Fatal(err)
	}
	m := New(cat, Config{TempDir: tmpDir, ConvertThresholdBytes: 100, MergeTargetBytes: 200, MaxL1Files: 96})
	l2Ran := false
	m.runJob = func(_ context.Context, job compact.Job) (*compact.Result, error) {
		if strings.Contains(job.Inputs[0], string(filepath.Separator)+"l1"+string(filepath.Separator)) {
			return nil, errors.New("broken l1 input")
		}
		l2Ran = true
		if err := os.WriteFile(job.Output, make([]byte, 120), 0o644); err != nil {
			return nil, err
		}
		return &compact.Result{Rows: 1, MinMs: base.UnixMilli(), MaxMs: base.Add(time.Minute).UnixMilli()}, nil
	}

	if err := m.CompactStartup(context.Background()); err == nil {
		t.Fatal("expected startup compaction error")
	}
	if !l2Ran {
		t.Fatal("expected l2 compaction to continue after l1 failure")
	}
	if got := len(cat.Level(catalog.L1)); got != 2 {
		t.Fatalf("expected failed l1 inputs to remain, got %d", got)
	}
	if got := len(cat.Level(catalog.L2)); got != 1 {
		t.Fatalf("expected l2 inputs to merge, got %d files", got)
	}
	for _, path := range l1Inputs {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected failed input %s to remain: %v", path, err)
		}
	}
}

func TestCompactStartupReprocessesOutputsWhenBatchIsCapped(t *testing.T) {
	archiveDir := t.TempDir()
	tmpDir := t.TempDir()
	base := time.UnixMilli(1_700_000_000_000)
	writeCatalogFiles(t, archiveDir, catalog.L1, base, []int{10, 10, 10, 10, 10, 10, 10})

	cat, err := catalog.Open(archiveDir)
	if err != nil {
		t.Fatal(err)
	}
	m := New(cat, Config{TempDir: tmpDir, ConvertThresholdBytes: 100, MergeTargetBytes: 200, MaxL1Files: 3})
	var inputCounts []int
	jobNumber := 0
	m.runJob = func(_ context.Context, job compact.Job) (*compact.Result, error) {
		inputCounts = append(inputCounts, len(job.Inputs))
		var size int64
		for _, input := range job.Inputs {
			info, err := os.Stat(input)
			if err != nil {
				return nil, err
			}
			size += info.Size()
		}
		if err := os.WriteFile(job.Output, make([]byte, size), 0o644); err != nil {
			return nil, err
		}
		jobNumber++
		min := base.Add(time.Duration(jobNumber) * 24 * time.Hour)
		return &compact.Result{Rows: 1, MinMs: min.UnixMilli(), MaxMs: min.Add(time.Minute).UnixMilli()}, nil
	}

	if err := m.CompactStartup(context.Background()); err != nil {
		t.Fatalf("startup compaction failed: %v", err)
	}
	if !reflect.DeepEqual(inputCounts, []int{3, 3, 3}) {
		t.Fatalf("expected capped outputs to be merged again, got batches %v", inputCounts)
	}
	if got := len(cat.Level(catalog.L1)); got != 1 {
		t.Fatalf("expected all seven inputs to consolidate to one file, got %d", got)
	}
}

func writeCatalogFiles(t *testing.T, archiveDir string, level int, start time.Time, sizes []int) []string {
	t.Helper()
	dir := filepath.Join(archiveDir, fmt.Sprintf("l%d", level))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	paths := make([]string, len(sizes))
	for i, size := range sizes {
		min := start.Add(time.Duration(i) * time.Minute)
		path := filepath.Join(dir, catalog.FileName(min, min.Add(time.Minute), int64(i+1)))
		if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
			t.Fatal(err)
		}
		paths[i] = path
	}
	return paths
}
