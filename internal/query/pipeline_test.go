package query

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/iwanhae/kabinet/internal/catalog"
	"github.com/iwanhae/kabinet/internal/compact"
	"github.com/iwanhae/kabinet/internal/wal"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func testEvent(uid, rv string, ts time.Time) *corev1.Event {
	return &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-" + uid + "-" + rv,
			Namespace:         "default",
			UID:               types.UID(uid),
			ResourceVersion:   rv,
			CreationTimestamp: metav1.NewTime(ts),
		},
		InvolvedObject: corev1.ObjectReference{Kind: "Pod", Namespace: "default", Name: "pod-" + uid},
		Reason:         "Testing",
		Message:        "message " + uid + "/" + rv,
		Type:           "Normal",
		FirstTimestamp: metav1.NewTime(ts),
		LastTimestamp:  metav1.NewTime(ts),
		Count:          2,
	}
}

// writeSealedSegment writes events into a fresh WAL writer and seals by
// shutting it down.
func writeSealedSegment(t *testing.T, walDir, tmpDir string, events []*corev1.Event) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	w, err := wal.NewWriter(ctx, wal.Options{Dir: walDir, TempDir: tmpDir, FlushInterval: 10 * time.Millisecond})
	if err != nil {
		t.Fatalf("failed to create wal writer: %v", err)
	}
	for _, e := range events {
		if err := w.Append(ctx, e); err != nil {
			t.Fatalf("failed to append: %v", err)
		}
	}
	time.Sleep(50 * time.Millisecond)
	cancel()
	w.Wait()
	time.Sleep(2 * time.Millisecond) // segment names are ms-resolution; keep them distinct
}

func TestPipelineConvertAndQuery(t *testing.T) {
	base := t.TempDir()
	walDir := filepath.Join(base, "wal")
	tmpDir := filepath.Join(base, "tmp")
	archiveDir := filepath.Join(base, "archive")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatal(err)
	}

	ts := time.Now().Add(-time.Hour).Truncate(time.Second)

	// Segment 1: two events, one duplicated (same uid+resourceVersion, as an
	// informer relist would produce), and one event with missing
	// firstTimestamp/count that must be normalized at read time.
	missing := testEvent("uid-b", "2", ts.Add(time.Minute))
	missing.FirstTimestamp = metav1.Time{}
	missing.LastTimestamp = metav1.Time{}
	missing.Count = 0
	writeSealedSegment(t, walDir, tmpDir, []*corev1.Event{
		testEvent("uid-a", "1", ts),
		testEvent("uid-a", "1", ts), // duplicate
		missing,
	})

	segments, err := wal.ListSealed(walDir)
	if err != nil || len(segments) != 1 {
		t.Fatalf("expected 1 sealed segment, got %d (err=%v)", len(segments), err)
	}

	// Convert to Parquet through the compactor logic.
	cat, err := catalog.Open(archiveDir)
	if err != nil {
		t.Fatalf("failed to open catalog: %v", err)
	}
	out := filepath.Join(tmpDir, "convert.parquet")
	result, err := compact.Run(context.Background(), compact.Job{
		Mode:          compact.ModeConvert,
		Inputs:        []string{segments[0].Path},
		Output:        out,
		TempDir:       tmpDir,
		MemoryLimitMB: 256,
	})
	if err != nil {
		t.Fatalf("convert failed: %v", err)
	}
	if result.Rows != 2 {
		t.Fatalf("expected 2 rows after dedup, got %d", result.Rows)
	}

	min, max := time.UnixMilli(result.MinMs), time.UnixMilli(result.MaxMs)
	final := filepath.Join(cat.LevelDir(catalog.L1), catalog.FileName(min, max, cat.NextSeq()))
	if err := os.Rename(out, final); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(final)
	cat.Add(catalog.File{Path: final, Level: catalog.L1, Min: min, Max: max, Size: info.Size(), Rows: result.Rows})
	if err := os.Remove(segments[0].Path); err != nil {
		t.Fatal(err)
	}

	// Segment 2 stays raw in the WAL: the query must union Parquet + JSONL.
	writeSealedSegment(t, walDir, tmpDir, []*corev1.Event{
		testEvent("uid-c", "3", ts.Add(2*time.Minute)),
	})

	executor, err := New(cat, nil, walDir)
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}
	defer executor.Close()

	start, end := ts.Add(-time.Minute), ts.Add(10*time.Minute)

	rows, meta, err := executor.RangeQuery(context.Background(),
		"SELECT count(*)::BIGINT AS c FROM $events", start, end)
	if err != nil {
		t.Fatalf("range query failed: %v", err)
	}
	if len(rows) != 1 || rows[0]["c"] != int64(3) {
		t.Fatalf("expected count 3 (2 parquet + 1 jsonl), got %v", rows)
	}
	if len(meta.Files) != 2 {
		t.Fatalf("expected 2 source files (1 parquet, 1 segment), got %v", meta.Files)
	}

	// Read-time normalization of the event with missing fields.
	rows, _, err = executor.RangeQuery(context.Background(),
		`SELECT "count", firstTimestamp, lastTimestamp FROM $events WHERE metadata.uid = 'uid-b'`, start, end)
	if err != nil {
		t.Fatalf("normalization query failed: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row for uid-b, got %d", len(rows))
	}
	if rows[0]["count"] != int32(1) {
		t.Fatalf("expected normalized count 1, got %v (%T)", rows[0]["count"], rows[0]["count"])
	}
	if rows[0]["firstTimestamp"] == nil || rows[0]["lastTimestamp"] == nil {
		t.Fatalf("expected timestamps backfilled from creationTimestamp, got %v", rows[0])
	}

	// A range with no data must still resolve with the canonical schema.
	rows, _, err = executor.RangeQuery(context.Background(),
		"SELECT count(*)::BIGINT AS c FROM $events", ts.Add(-48*time.Hour), ts.Add(-47*time.Hour))
	if err != nil {
		t.Fatalf("empty range query failed: %v", err)
	}
	if len(rows) != 1 || rows[0]["c"] != int64(0) {
		t.Fatalf("expected count 0 for empty range, got %v", rows)
	}

	// $events referenced multiple times (CTE + main query) must all resolve.
	rows, _, err = executor.RangeQuery(context.Background(), `
		WITH top_ns AS (
			SELECT metadata.namespace AS ns FROM $events GROUP BY 1 ORDER BY COUNT(*) DESC LIMIT 1
		)
		SELECT count(*)::BIGINT AS c FROM $events
		WHERE metadata.namespace IN (SELECT ns FROM top_ns)`, start, end)
	if err != nil {
		t.Fatalf("multi-$events query failed: %v", err)
	}
	if len(rows) != 1 || rows[0]["c"] != int64(3) {
		t.Fatalf("expected count 3 for multi-$events query, got %v", rows)
	}
}

func TestMergeDeduplicatesAcrossFiles(t *testing.T) {
	base := t.TempDir()
	walDir := filepath.Join(base, "wal")
	tmpDir := filepath.Join(base, "tmp")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatal(err)
	}

	ts := time.Now().Add(-time.Hour).Truncate(time.Second)

	// The same event lands in two different Parquet files — exactly what an
	// informer relist after a restart produces.
	convert := func(name string, events []*corev1.Event) string {
		dir := filepath.Join(walDir, name)
		writeSealedSegment(t, dir, tmpDir, events)
		segments, err := wal.ListSealed(dir)
		if err != nil || len(segments) != 1 {
			t.Fatalf("expected 1 segment, got %d (err=%v)", len(segments), err)
		}
		out := filepath.Join(tmpDir, name+".parquet")
		if _, err := compact.Run(context.Background(), compact.Job{
			Mode: compact.ModeConvert, Inputs: []string{segments[0].Path},
			Output: out, TempDir: tmpDir, MemoryLimitMB: 256,
		}); err != nil {
			t.Fatalf("convert failed: %v", err)
		}
		return out
	}

	dup := testEvent("uid-a", "1", ts)
	p1 := convert("one", []*corev1.Event{dup, testEvent("uid-b", "2", ts.Add(time.Minute))})
	p2 := convert("two", []*corev1.Event{dup, testEvent("uid-c", "3", ts.Add(2*time.Minute))})

	merged := filepath.Join(tmpDir, "merged.parquet")
	result, err := compact.Run(context.Background(), compact.Job{
		Mode: compact.ModeMerge, Inputs: []string{p1, p2},
		Output: merged, TempDir: tmpDir, MemoryLimitMB: 256,
	})
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}
	if result.Rows != 3 {
		t.Fatalf("expected 3 rows after cross-file dedup, got %d", result.Rows)
	}
}
