package wal

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	sealedSuffix = ".jsonl.zst"
	openSuffix   = ".jsonl.zst.open"
	filePrefix   = "events_"
)

// Segment is a sealed (immutable) WAL segment containing raw K8s Event JSONL.
// Start/End are the event-time range of its contents (not ingestion time), so
// time-range pruning stays correct when old events arrive late (relists).
type Segment struct {
	Path  string
	Start time.Time
	End   time.Time
	Size  int64
}

// Overlaps reports whether the segment's time range intersects [start, end].
func (s Segment) Overlaps(start, end time.Time) bool {
	return !s.End.Before(start) && !s.Start.After(end)
}

func openName(start time.Time) string {
	return fmt.Sprintf("%s%d%s", filePrefix, start.UnixMilli(), openSuffix)
}

func sealedName(start, end time.Time) string {
	return fmt.Sprintf("%s%d_%d%s", filePrefix, start.UnixMilli(), end.UnixMilli(), sealedSuffix)
}

// parseOpenName extracts the start time from "events_<startMs>.jsonl.zst.open".
func parseOpenName(name string) (time.Time, bool) {
	if !strings.HasPrefix(name, filePrefix) || !strings.HasSuffix(name, openSuffix) {
		return time.Time{}, false
	}
	ms, err := strconv.ParseInt(strings.TrimSuffix(strings.TrimPrefix(name, filePrefix), openSuffix), 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	return time.UnixMilli(ms), true
}

// parseSealedName extracts start/end from "events_<startMs>_<endMs>.jsonl.zst".
func parseSealedName(name string) (start, end time.Time, ok bool) {
	if !strings.HasPrefix(name, filePrefix) || !strings.HasSuffix(name, sealedSuffix) {
		return time.Time{}, time.Time{}, false
	}
	parts := strings.Split(strings.TrimSuffix(strings.TrimPrefix(name, filePrefix), sealedSuffix), "_")
	if len(parts) != 2 {
		return time.Time{}, time.Time{}, false
	}
	startMs, err1 := strconv.ParseInt(parts[0], 10, 64)
	endMs, err2 := strconv.ParseInt(parts[1], 10, 64)
	if err1 != nil || err2 != nil {
		return time.Time{}, time.Time{}, false
	}
	return time.UnixMilli(startMs), time.UnixMilli(endMs), true
}

// ListSealed returns all sealed segments in dir, oldest first.
func ListSealed(dir string) ([]Segment, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read wal directory: %w", err)
	}

	var segments []Segment
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		start, end, ok := parseSealedName(entry.Name())
		if !ok {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		segments = append(segments, Segment{
			Path:  filepath.Join(dir, entry.Name()),
			Start: start,
			End:   end,
			Size:  info.Size(),
		})
	}

	sort.Slice(segments, func(i, j int) bool { return segments[i].Start.Before(segments[j].Start) })
	return segments, nil
}
