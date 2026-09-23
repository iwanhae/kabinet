package query

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/iwanhae/kabinet/internal/catalog"
	"github.com/iwanhae/kabinet/internal/compact"
	"github.com/iwanhae/kabinet/internal/schema"
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
	// informer relist would produce), and one event with missing source
	// timestamps/count. Only canonical timestamp and count are derived.
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
	seriesEvent := testEvent("uid-c", "3", ts.Add(2*time.Minute))
	observed := ts.Add(5*time.Minute + 1234*time.Microsecond)
	seriesEvent.Series = &corev1.EventSeries{LastObservedTime: metav1.NewMicroTime(observed)}
	writeSealedSegment(t, walDir, tmpDir, []*corev1.Event{
		seriesEvent,
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
	rows, _, err = executor.RangeQuery(context.Background(),
		`SELECT timestamp FROM $events WHERE metadata.uid = 'uid-c'`, observed.Add(-10*time.Microsecond), observed.Add(10*time.Microsecond))
	if err != nil || len(rows) != 1 {
		t.Fatalf("series timestamp query failed: rows=%v err=%v", rows, err)
	}
	if got, ok := rows[0]["timestamp"].(time.Time); !ok || !got.Equal(observed) {
		t.Fatalf("expected series.lastObservedTime %s, got %v", observed, rows[0]["timestamp"])
	}

	// Canonical timestamp falls back to creationTimestamp while source time
	// fields remain unchanged.
	rows, _, err = executor.RangeQuery(context.Background(),
		`SELECT "count", timestamp, firstTimestamp, lastTimestamp FROM $events WHERE metadata.uid = 'uid-b'`, start, end)
	if err != nil {
		t.Fatalf("normalization query failed: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row for uid-b, got %d", len(rows))
	}
	if rows[0]["count"] != int32(1) {
		t.Fatalf("expected normalized count 1, got %v (%T)", rows[0]["count"], rows[0]["count"])
	}
	if rows[0]["timestamp"] == nil {
		t.Fatalf("expected canonical timestamp from creationTimestamp, got %v", rows[0])
	}
	if rows[0]["firstTimestamp"] != nil || rows[0]["lastTimestamp"] != nil {
		t.Fatalf("expected source timestamps to remain null, got %v", rows[0])
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

	older := testEvent("uid-a", "1", ts)
	older.Message = "older duplicate"
	newer := older.DeepCopy()
	newer.LastTimestamp = metav1.NewTime(ts.Add(30 * time.Second))
	newer.Message = "newer duplicate"

	tieOne := testEvent("uid-tie", "4", ts.Add(45*time.Second))
	tieOne.Message = "first tie"
	tieTwo := tieOne.DeepCopy()
	tieTwo.Message = "second tie"

	p1 := convert("one", []*corev1.Event{older, tieOne, testEvent("uid-b", "2", ts.Add(time.Minute))})
	p2 := convert("two", []*corev1.Event{newer, tieTwo, testEvent("uid-c", "3", ts.Add(2*time.Minute))})

	merged := filepath.Join(tmpDir, "merged.parquet")
	result, err := compact.Run(context.Background(), compact.Job{
		Mode: compact.ModeMerge, Inputs: []string{p1, p2},
		Output: merged, TempDir: tmpDir, MemoryLimitMB: 256,
	})
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}
	if result.Rows != 4 {
		t.Fatalf("expected 4 rows after cross-file dedup, got %d", result.Rows)
	}

	repacked := filepath.Join(tmpDir, "repacked.parquet")
	repackedResult, err := compact.Run(context.Background(), compact.Job{
		Mode: compact.ModeRepack, Inputs: []string{p1, p2},
		Output: repacked, TempDir: tmpDir, MemoryLimitMB: 256,
	})
	if err != nil {
		t.Fatalf("repack failed: %v", err)
	}
	if repackedResult.Rows != 6 {
		t.Fatalf("expected repack to preserve all 6 rows, got %d", repackedResult.Rows)
	}

	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	parquet := "read_parquet(" + schema.QuotePath(merged) + ")"
	var message string
	if err := db.QueryRow("SELECT message FROM " + parquet + " WHERE metadata.uid = 'uid-a'").Scan(&message); err != nil {
		t.Fatal(err)
	}
	if message != "newer duplicate" {
		t.Fatalf("expected latest duplicate, got %q", message)
	}
	if err := db.QueryRow("SELECT message FROM " + parquet + " WHERE metadata.uid = 'uid-tie'").Scan(&message); err != nil {
		t.Fatal(err)
	}
	if message != "second tie" {
		t.Fatalf("expected deterministic filename tie-break, got %q", message)
	}

	var duplicateCount int
	if err := db.QueryRow("SELECT count(*) FROM read_parquet(" + schema.QuotePath(repacked) + ") WHERE metadata.uid = 'uid-a'").Scan(&duplicateCount); err != nil {
		t.Fatal(err)
	}
	if duplicateCount != 2 {
		t.Fatalf("expected repack to preserve both duplicate rows, got %d", duplicateCount)
	}

	rows, err := db.Query("SELECT * FROM " + parquet + " LIMIT 0")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		t.Fatal(err)
	}
	for _, column := range columns {
		if column == "filename" || column == "file_row_number" {
			t.Fatalf("virtual row locator leaked into output schema: %s", column)
		}
	}
}

func TestLegacyParquetMigrationBackfillsTimestampWithinRange(t *testing.T) {
	base := t.TempDir()
	walDir := filepath.Join(base, "wal")
	tmpDir := filepath.Join(base, "tmp")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ts := time.Now().Add(-time.Hour).Truncate(time.Second)
	late := testEvent("late", "1", ts)
	late.Series = &corev1.EventSeries{LastObservedTime: metav1.NewMicroTime(ts.Add(2*time.Minute + 321*time.Microsecond))}
	writeSealedSegment(t, walDir, tmpDir, []*corev1.Event{
		late,
		testEvent("early", "2", ts.Add(time.Minute)),
	})
	segments, err := wal.ListSealed(walDir)
	if err != nil || len(segments) != 1 {
		t.Fatalf("segments=%d err=%v", len(segments), err)
	}
	canonical := filepath.Join(tmpDir, "canonical.parquet")
	if _, err := compact.Run(context.Background(), compact.Job{
		Mode: compact.ModeConvert, Inputs: []string{segments[0].Path}, Output: canonical,
		TempDir: tmpDir, MemoryLimitMB: 256,
	}); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("duckdb", "")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	legacy := filepath.Join(tmpDir, "legacy.parquet")
	if _, err := db.Exec("COPY (SELECT * EXCLUDE (timestamp) FROM read_parquet(" + schema.QuotePath(canonical) + ")) TO " + schema.QuotePath(legacy) + " (FORMAT parquet)"); err != nil {
		t.Fatal(err)
	}
	inspection, err := compact.Run(context.Background(), compact.Job{
		Mode: compact.ModeInspect, LegacyInputs: []string{legacy}, TempDir: tmpDir, MemoryLimitMB: 256,
	})
	if err != nil || inspection.Rows != 2 || inspection.MinUs == 0 || inspection.MaxUs == 0 {
		t.Fatalf("inspection=%+v err=%v", inspection, err)
	}
	startUs := ts.Add(-time.Minute).UnixMicro()
	endUs := ts.Add(3 * time.Minute).UnixMicro()
	migrated := filepath.Join(tmpDir, "migrated.parquet")
	result, err := compact.Run(context.Background(), compact.Job{
		Mode: compact.ModeMigrate, LegacyInputs: []string{legacy}, Output: migrated,
		TempDir: tmpDir, MemoryLimitMB: 256, RangeStartUs: &startUs, RangeEndUs: &endUs,
	})
	if err != nil || result.Rows != 2 {
		t.Fatalf("migration=%+v err=%v", result, err)
	}
	var migratedCount int
	if err := db.QueryRow("SELECT count(*) FROM read_parquet("+schema.QuotePath(migrated)+") WHERE epoch_us(timestamp) >= ? AND epoch_us(timestamp) < ?", startUs, endUs).Scan(&migratedCount); err != nil {
		t.Fatal(err)
	}
	if migratedCount != 2 {
		t.Fatalf("expected both migrated rows inside the requested range, got %d", migratedCount)
	}
	var inversions int
	if err := db.QueryRow(`WITH physical_order AS (
		SELECT timestamp, lag(timestamp) OVER (ORDER BY file_row_number) AS previous
		FROM read_parquet(` + schema.QuotePath(migrated) + `, file_row_number=true)
	) SELECT count(*) FROM physical_order WHERE timestamp < previous`).Scan(&inversions); err != nil {
		t.Fatal(err)
	}
	if inversions != 0 {
		t.Fatalf("hourly migration output is not timestamp-sorted: %d inversions", inversions)
	}
	if result.MaxMs != (late.Series.LastObservedTime.UnixMicro()+999)/1000 {
		t.Fatalf("expected conservative max millisecond, got %d", result.MaxMs)
	}
}
