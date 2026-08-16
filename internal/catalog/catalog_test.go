package catalog

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileNameRoundTrip(t *testing.T) {
	min := time.UnixMilli(1700000000000)
	max := time.UnixMilli(1700000060000)
	name := FileName(min, max, 42)

	gotMin, gotMax, gotSeq, ok := parseFileName(name)
	if !ok {
		t.Fatalf("failed to parse %s", name)
	}
	if !gotMin.Equal(min) || !gotMax.Equal(max) || gotSeq != 42 {
		t.Fatalf("round trip mismatch: %v %v %d", gotMin, gotMax, gotSeq)
	}
}

func TestOpenScanAndOverlap(t *testing.T) {
	dir := t.TempDir()

	t0 := time.UnixMilli(1700000000000)
	mk := func(level int, min, max time.Time, seq int64) {
		path := filepath.Join(dir, "l"+string(rune('0'+level)), FileName(min, max, seq))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mk(L1, t0, t0.Add(time.Minute), 1)
	mk(L2, t0.Add(2*time.Minute), t0.Add(3*time.Minute), 2)

	cat, err := Open(dir)
	if err != nil {
		t.Fatalf("failed to open catalog: %v", err)
	}

	if got := len(cat.All()); got != 2 {
		t.Fatalf("expected 2 files, got %d", got)
	}
	if seq := cat.NextSeq(); seq != 3 {
		t.Fatalf("expected next seq 3, got %d", seq)
	}
	if got := cat.Overlapping(t0.Add(30*time.Second), t0.Add(90*time.Second)); len(got) != 1 || got[0].Level != L1 {
		t.Fatalf("expected only the l1 file to overlap, got %v", got)
	}
	if got := cat.Overlapping(t0.Add(10*time.Minute), t0.Add(20*time.Minute)); len(got) != 0 {
		t.Fatalf("expected no overlap, got %v", got)
	}
	if got := cat.Level(L2); len(got) != 1 {
		t.Fatalf("expected 1 l2 file, got %d", len(got))
	}
}
