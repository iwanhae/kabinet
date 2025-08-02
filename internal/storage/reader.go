package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Reader is responsible for reading events and executing queries.
type Reader struct {
	queryMu     sync.Mutex // Ensures only one query runs at a time to control memory usage.
	dbPath      string     // Path to the writer's database file.
	parquetPath string     // Path to the directory containing parquet files.
}

// RangeQueryResult holds the result and metadata of a range query.
type RangeQueryResult struct {
	Duration time.Duration `json:"duration_ms"`
	Files    []string      `json:"files"`
}

// NewReader creates and initializes a new Reader instance.
func NewReader(dbPath string, parquetPath string) (*Reader, error) {
	// The reader now runs in a separate container, so it should not create directories.
	// It assumes the writer has already created them.
	if _, err := os.Stat(filepath.Dir(dbPath)); os.IsNotExist(err) {
		log.Printf("reader: db directory %s does not exist. waiting for writer to create it.", filepath.Dir(dbPath))
	}
	if _, err := os.Stat(parquetPath); os.IsNotExist(err) {
		log.Printf("reader: parquet directory %s does not exist. waiting for writer to create it.", parquetPath)
	}

	return &Reader{
		dbPath:      dbPath,
		parquetPath: parquetPath,
	}, nil
}

// RangeQuery executes a query against a specified time range.
func (r *Reader) RangeQuery(ctx context.Context, query string, start, end time.Time) (*sql.Rows, *RangeQueryResult, error) {
	r.queryMu.Lock()
	defer r.queryMu.Unlock()

	if ctx.Err() != nil {
		return nil, nil, fmt.Errorf("query context cancelled: %w", ctx.Err())
	}

	// Create a temporary in-memory database for the query.
	conn, err := sql.Open("duckdb", "") // Empty DSN for in-memory
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open in-memory db: %w", err)
	}
	defer conn.Close()

	// ATTACH the writer's database for real-time data.
	attachSQL := fmt.Sprintf("ATTACH '%s' AS hot_storage (READ_ONLY);", r.dbPath)
	if _, err := conn.ExecContext(ctx, attachSQL); err != nil {
		// If the db file doesn't exist yet, we can still query parquet files.
		if os.IsNotExist(err) {
			log.Printf("reader: writer db not found at %s, querying parquet files only.", r.dbPath)
		} else {
			return nil, nil, fmt.Errorf("failed to attach writer db: %w", err)
		}
	}
	defer conn.Exec("DETACH hot_storage;")

	// Find relevant parquet files.
	relevantFiles, err := r.findRelevantParquetFiles(start, end)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find relevant parquet files: %w", err)
	}

	fromClause, err := r.buildFromClause(relevantFiles)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot build query: %w", err)
	}

	finalQuery := strings.Replace(query, "$events", fromClause, 1)
	log.Printf("reader: executing query: %s", finalQuery)

	startTime := time.Now()
	rows, err := conn.QueryContext(ctx, finalQuery)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute query: %w", err)
	}

	result := &RangeQueryResult{
		Duration: time.Since(startTime),
		Files:    relevantFiles,
	}

	return rows, result, nil
}

func (r *Reader) findRelevantParquetFiles(start, end time.Time) ([]string, error) {
	files, err := os.ReadDir(r.parquetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // It's okay if the directory doesn't exist yet.
		}
		return nil, fmt.Errorf("failed to read parquet directory: %w", err)
	}

	queryStartTs := start.Unix()
	queryEndTs := end.Unix()

	var relevantFiles []string
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".parquet") {
			continue
		}

		minTs, maxTs, ok := parseParquetFilename(file.Name())
		if !ok {
			log.Printf("reader: could not parse filename %s, including it just in case.", file.Name())
			relevantFiles = append(relevantFiles, filepath.Join(r.parquetPath, file.Name()))
			continue
		}

		// File range overlaps with query range if: (file_start <= query_end) AND (file_end >= query_start)
		if maxTs >= queryStartTs && minTs <= queryEndTs {
			relevantFiles = append(relevantFiles, filepath.Join(r.parquetPath, file.Name()))
		}
	}
	return relevantFiles, nil
}

func (r *Reader) buildFromClause(parquetFiles []string) (string, error) {
	var fromSources []string

	// Always try to include the hot storage table.
	fromSources = append(fromSources, "SELECT * FROM hot_storage.kube_events")

	if len(parquetFiles) > 0 {
		quotedFiles := make([]string, len(parquetFiles))
		for i, p := range parquetFiles {
			quotedFiles[i] = fmt.Sprintf("'%s'", p)
		}
		parquetSource := fmt.Sprintf("SELECT * FROM read_parquet([%s])", strings.Join(quotedFiles, ", "))
		fromSources = append(fromSources, parquetSource)
	}

	if len(fromSources) == 0 {
		return "", fmt.Errorf("no data sources available for query")
	}

	return fmt.Sprintf("(%s)", strings.Join(fromSources, " UNION ALL BY NAME ")), nil
}

// Close is a no-op for the reader as it doesn't hold any long-lived connections.
func (r *Reader) Close() {
	log.Println("reader: closing.")
}
