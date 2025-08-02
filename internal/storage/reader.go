package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const connectionWarmupAfter = 5 * time.Second

// Reader is responsible for reading events and executing queries.
type Reader struct {
	queryMu sync.Mutex // A mutex to serialize read queries, primarily to control memory usage.
	connMgr *connectionManager
	dataDir string
}

// ParquetFileInfo holds metadata about a single Parquet file.
type ParquetFileInfo struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// RangeQueryResult holds the result and metadata of a range query.
type RangeQueryResult struct {
	Duration time.Duration     `json:"duration_ms"`
	Files    []ParquetFileInfo `json:"files"`
}

// NewReader creates and initializes a new Reader instance.
func NewReader(dbPath string) (*Reader, error) {
	dataDir := filepath.Dir(dbPath)
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("data directory %s does not exist. The Writer must be initialized first", dataDir)
	}

	return &Reader{
		connMgr: newConnectionManager(dbPath, connectionWarmupAfter),
		dataDir: dataDir,
	}, nil
}

// RangeQuery executes a query against a specified time range.
func (r *Reader) RangeQuery(ctx context.Context, query string, start, end time.Time) (*sql.Rows, *RangeQueryResult, error) {
	// Rule: Execute only one read query at a time to control memory usage.
	r.queryMu.Lock()
	defer r.queryMu.Unlock()

	if ctx.Err() != nil {
		return nil, nil, fmt.Errorf("failed fast: %w", ctx.Err())
	}

	files, err := os.ReadDir(r.dataDir)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read data directory: %w", err)
	}

	queryStartTs := start.Unix()
	queryEndTs := end.Unix()

	var relevantFiles []ParquetFileInfo
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".parquet") {
			continue
		}
		info, err := file.Info()
		if err != nil {
			log.Printf("reader: could not get file info for %s: %v", file.Name(), err)
			continue
		}

		minTs, maxTs, ok := parseParquetFilename(file.Name())
		if !ok {
			// Include the file just in case, as we can't be sure of its time range.
			log.Printf("reader: could not parse filename %s, including it", file.Name())
			relevantFiles = append(relevantFiles, ParquetFileInfo{Path: filepath.Join(r.dataDir, file.Name()), Size: info.Size()})
			continue
		}

		// File range overlaps with query range if:
		// (file_start <= query_end) AND (file_end >= query_start)
		if maxTs >= queryStartTs && minTs <= queryEndTs {
			relevantFiles = append(relevantFiles, ParquetFileInfo{Path: filepath.Join(r.dataDir, file.Name()), Size: info.Size()})
		}
	}

	fromClause, err := buildFromClause(relevantFiles, true) // Always include the live db file.
	if err != nil {
		return nil, nil, fmt.Errorf("query time range resulted in no data sources")
	}

	finalQuery := strings.Replace(query, "$events", fromClause, 1)

	executor, err := r.connMgr.Get()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get a read connection: %w", err)
	}

	log.Printf("reader: executing range query: %s", finalQuery)
	now := time.Now()
	rows, err := executor.QueryContext(ctx, finalQuery)
	if err != nil {
		return nil, nil, err
	}

	return rows, &RangeQueryResult{
		Duration: time.Since(now),
		Files:    relevantFiles,
	}, nil
}

// Close safely shuts down the connection manager.
func (r *Reader) Close() {
	log.Println("reader: closing...")
	r.connMgr.Close()
	log.Println("reader: closed.")
}

func buildFromClause(relevantFiles []ParquetFileInfo, includeKubeEvents bool) (string, error) {
	var fromSources []string
	if includeKubeEvents {
		fromSources = append(fromSources, SQLSelectFromKubeEvents)
	}

	if len(relevantFiles) > 0 {
		paths := make([]string, len(relevantFiles))
		for i, f := range relevantFiles {
			paths[i] = fmt.Sprintf("'%s'", f.Path)
		}
		parquetSource := fmt.Sprintf(SQLReadFromParquetTemplate, strings.Join(paths, ", "))
		fromSources = append(fromSources, parquetSource)
	}

	if len(fromSources) == 0 {
		return "", fmt.Errorf("no data sources for query")
	}

	return fmt.Sprintf("(%s)", strings.Join(fromSources, " UNION ALL BY NAME ")), nil
}

// parseParquetFilename extracts the min and max unix timestamps from a parquet filename.
// It expects the format "events_MIN_MAX.parquet".
func parseParquetFilename(filename string) (int64, int64, bool) {
	base := strings.TrimSuffix(filename, ".parquet")
	parts := strings.Split(base, "_")
	if len(parts) != 3 || parts[0] != "events" {
		return 0, 0, false
	}
	minTs, errMin := strconv.ParseInt(parts[1], 10, 64)
	maxTs, errMax := strconv.ParseInt(parts[2], 10, 64)
	if errMin != nil || errMax != nil {
		return 0, 0, false
	}
	return minTs, maxTs, true
}
