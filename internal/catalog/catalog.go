// Package catalog indexes the archived Parquet files. The directory is the
// source of truth — every file's time range and sequence number are encoded
// in its name (events_<minMs>_<maxMs>_<seq>.parquet) and its level in its
// subdirectory (l1/, l2/) — so the catalog is rebuilt by scanning at startup.
// The lifecycle manager is the only writer; the query planner only reads.
package catalog

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Levels of the archive hierarchy.
const (
	L1 = 1 // direct output of WAL segment conversion (small files)
	L2 = 2 // merged long-term files
)

// File is one archived Parquet file.
type File struct {
	Path  string
	Level int
	Min   time.Time
	Max   time.Time
	Size  int64
	Rows  int64 // 0 when unknown (rebuilt from a directory scan)
}

// Catalog is the in-memory index of archived Parquet files.
type Catalog struct {
	mu    sync.RWMutex
	dir   string
	files map[string]File // keyed by absolute path
	seq   int64
}

// Open scans the archive directory (creating level subdirectories as needed)
// and builds the catalog.
func Open(dir string) (*Catalog, error) {
	c := &Catalog{dir: dir, files: make(map[string]File)}

	for _, level := range []int{L1, L2} {
		levelDir := c.LevelDir(level)
		if err := os.MkdirAll(levelDir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create archive directory %s: %w", levelDir, err)
		}
		entries, err := os.ReadDir(levelDir)
		if err != nil {
			return nil, fmt.Errorf("failed to read archive directory %s: %w", levelDir, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			min, max, seq, ok := parseFileName(entry.Name())
			if !ok {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			path := filepath.Join(levelDir, entry.Name())
			c.files[path] = File{Path: path, Level: level, Min: min, Max: max, Size: info.Size()}
			if seq > c.seq {
				c.seq = seq
			}
		}
	}
	return c, nil
}

// LevelDir returns the directory holding files of the given level.
func (c *Catalog) LevelDir(level int) string {
	return filepath.Join(c.dir, fmt.Sprintf("l%d", level))
}

// NextSeq returns a monotonically increasing sequence number, used to keep
// file names unique even when time ranges collide.
func (c *Catalog) NextSeq() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seq++
	return c.seq
}

// Add registers a file.
func (c *Catalog) Add(f File) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.files[f.Path] = f
}

// Remove unregisters a file, returning it if present.
func (c *Catalog) Remove(path string) (File, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	f, ok := c.files[path]
	delete(c.files, path)
	return f, ok
}

// Overlapping returns files whose time range intersects [start, end],
// oldest first.
func (c *Catalog) Overlapping(start, end time.Time) []File {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var out []File
	for _, f := range c.files {
		if !f.Max.Before(start) && !f.Min.After(end) {
			out = append(out, f)
		}
	}
	sortByMin(out)
	return out
}

// Level returns all files of the given level, oldest first.
func (c *Catalog) Level(level int) []File {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var out []File
	for _, f := range c.files {
		if f.Level == level {
			out = append(out, f)
		}
	}
	sortByMin(out)
	return out
}

// All returns every file, oldest first (retention order).
func (c *Catalog) All() []File {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]File, 0, len(c.files))
	for _, f := range c.files {
		out = append(out, f)
	}
	sortByMin(out)
	return out
}

// TotalSize returns the combined size of all archived files.
func (c *Catalog) TotalSize() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var total int64
	for _, f := range c.files {
		total += f.Size
	}
	return total
}

// Stats returns archive statistics for the /stats endpoint.
func (c *Catalog) Stats() map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	counts := map[int]int{}
	var total int64
	for _, f := range c.files {
		counts[f.Level]++
		total += f.Size
	}
	return map[string]any{
		"files_l1":         counts[L1],
		"files_l2":         counts[L2],
		"total_size_bytes": total,
	}
}

// FileName builds the canonical archive file name.
func FileName(min, max time.Time, seq int64) string {
	return fmt.Sprintf("events_%d_%d_%d.parquet", min.UnixMilli(), max.UnixMilli(), seq)
}

func parseFileName(name string) (min, max time.Time, seq int64, ok bool) {
	base, found := strings.CutSuffix(name, ".parquet")
	if !found || !strings.HasPrefix(base, "events_") {
		return time.Time{}, time.Time{}, 0, false
	}
	parts := strings.Split(strings.TrimPrefix(base, "events_"), "_")
	if len(parts) != 3 {
		return time.Time{}, time.Time{}, 0, false
	}
	minMs, err1 := strconv.ParseInt(parts[0], 10, 64)
	maxMs, err2 := strconv.ParseInt(parts[1], 10, 64)
	seq, err3 := strconv.ParseInt(parts[2], 10, 64)
	if err1 != nil || err2 != nil || err3 != nil {
		return time.Time{}, time.Time{}, 0, false
	}
	return time.UnixMilli(minMs), time.UnixMilli(maxMs), seq, true
}

func sortByMin(files []File) {
	sort.Slice(files, func(i, j int) bool {
		if files[i].Min.Equal(files[j].Min) {
			return files[i].Path < files[j].Path
		}
		return files[i].Min.Before(files[j].Min)
	})
}
