package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildEventsQuery(t *testing.T) {
	tmpDir := t.TempDir()
	// create dummy parquet files
	if err := os.WriteFile(filepath.Join(tmpDir, "events-0-100.parquet"), []byte{}, 0644); err != nil {
		t.Fatalf("failed to create parquet file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "events-200-300.parquet"), []byte{}, 0644); err != nil {
		t.Fatalf("failed to create parquet file: %v", err)
	}

	s := &Storage{dataDir: tmpDir}

	start := time.Unix(50, 0).UTC()
	end := time.Unix(150, 0).UTC()

	q := "SELECT * FROM $events WHERE type = 'Warning' ORDER BY lastTimestamp"
	final, files, err := s.buildEventsQuery(q, start, end)
	if err != nil {
		t.Fatalf("buildEventsQuery returned error: %v", err)
	}

	if !strings.Contains(final, "type = 'Warning'") {
		t.Errorf("where clause missing in final query: %s", final)
	}
	if !strings.Contains(final, "ORDER BY lastTimestamp") {
		t.Errorf("order by missing in final query: %s", final)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 relevant file, got %d", len(files))
	}
	if !strings.Contains(final, "events-0-100.parquet") {
		t.Errorf("expected first parquet file in query, got %s", final)
	}
}
