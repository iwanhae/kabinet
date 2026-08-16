// Package query implements the read path. It resolves the $events macro into
// a UNION of canonical Parquet files (from the catalog) and raw WAL segments
// (sealed files plus a stable snapshot of the active one) selected by time
// range, and executes the result on a read-only in-memory DuckDB connection.
package query

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/iwanhae/kabinet/internal/catalog"
	"github.com/iwanhae/kabinet/internal/schema"
	"github.com/iwanhae/kabinet/internal/wal"
	_ "github.com/marcboeker/go-duckdb/v2"
)

// SourceFile is one data source consulted by a query.
type SourceFile struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// Result contains metadata about a query execution.
type Result struct {
	Duration time.Duration
	Files    []SourceFile
}

// Executor plans and runs range queries.
type Executor struct {
	db     *sql.DB
	cat    *catalog.Catalog
	wal    *wal.Writer
	walDir string
}

// New creates an executor backed by an in-memory DuckDB instance that only
// ever reads external files.
func New(cat *catalog.Catalog, walWriter *wal.Writer, walDir string) (*Executor, error) {
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("failed to open duckdb: %w", err)
	}
	return &Executor{db: db, cat: cat, wal: walWriter, walDir: walDir}, nil
}

// Close releases the DuckDB connection.
func (e *Executor) Close() {
	if err := e.db.Close(); err != nil {
		log.Printf("query: error closing database: %v", err)
	}
}

// buildEventsQuery replaces the $events macro with a FROM clause covering
// every data source overlapping [start, end].
func (e *Executor) buildEventsQuery(query string, start, end time.Time) (string, []SourceFile, error) {
	between := fmt.Sprintf(
		"lastTimestamp BETWEEN TIMESTAMPTZ '%s' AND TIMESTAMPTZ '%s'",
		start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339),
	)

	var sources []string
	var files []SourceFile

	// Archived Parquet.
	if parquet := e.cat.Overlapping(start, end); len(parquet) > 0 {
		paths := make([]string, len(parquet))
		for i, f := range parquet {
			paths[i] = f.Path
			files = append(files, SourceFile{Path: f.Path, Size: f.Size})
		}
		sources = append(sources, fmt.Sprintf("SELECT * FROM %s WHERE %s", schema.ParquetSource(paths), between))
	}

	// Raw WAL: sealed segments first, then a stable snapshot of the active
	// segment (this order avoids counting in-flight data twice when a seal
	// races the query).
	var jsonlPaths []string
	sealed, err := wal.ListSealed(e.walDir)
	if err != nil {
		return "", nil, err
	}
	for _, seg := range sealed {
		if seg.Overlaps(start, end) {
			jsonlPaths = append(jsonlPaths, seg.Path)
			files = append(files, SourceFile{Path: seg.Path, Size: seg.Size})
		}
	}
	if e.wal != nil {
		snap, ok, err := e.wal.Snapshot()
		if err != nil {
			log.Printf("query: failed to snapshot active segment (skipping): %v", err)
		} else if ok && snap.Overlaps(start, end) {
			jsonlPaths = append(jsonlPaths, snap.Path)
			files = append(files, SourceFile{Path: snap.Path, Size: snap.Size})
		}
	}
	if len(jsonlPaths) > 0 {
		sources = append(sources, fmt.Sprintf("SELECT * FROM %s WHERE %s", schema.JSONLSource(jsonlPaths), between))
	}

	var fromClause string
	if len(sources) == 0 {
		fromClause = schema.EmptySource()
	} else {
		fromClause = "(" + strings.Join(sources, " UNION ALL BY NAME ") + ")"
	}

	// ReplaceAll: queries may reference $events multiple times (CTEs, subqueries).
	return strings.ReplaceAll(query, "$events", fromClause), files, nil
}

// RangeQuery executes a query with the $events macro over [start, end] and
// returns all rows.
func (e *Executor) RangeQuery(ctx context.Context, query string, start, end time.Time) ([]map[string]any, *Result, error) {
	if ctx.Err() != nil {
		return nil, nil, fmt.Errorf("failed fast: %w", ctx.Err())
	}

	finalQuery, files, err := e.buildEventsQuery(query, start, end)
	if err != nil {
		return nil, nil, err
	}

	log.Printf("query: executing range query: %s", finalQuery)

	now := time.Now()
	rows, err := e.db.QueryContext(ctx, finalQuery)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	results, err := serializeRows(rows)
	if err != nil {
		return nil, nil, err
	}
	return results, &Result{Duration: time.Since(now), Files: files}, nil
}

// StreamEvents selects full events in [start, end] (optionally filtered by
// where) ordered by lastTimestamp, streaming each row to handler without
// buffering the result set.
func (e *Executor) StreamEvents(ctx context.Context, where string, start, end time.Time, handler func(map[string]any) error) (*Result, error) {
	baseQuery := "SELECT * FROM $events"
	if strings.TrimSpace(where) != "" {
		baseQuery += " WHERE " + where
	}
	baseQuery += " ORDER BY lastTimestamp"

	finalQuery, files, err := e.buildEventsQuery(baseQuery, start, end)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	rows, err := e.db.QueryContext(ctx, finalQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		rowValues := make([]any, len(columns))
		rowPtrs := make([]any, len(columns))
		for i := range rowValues {
			rowPtrs[i] = &rowValues[i]
		}
		if err := rows.Scan(rowPtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		rowData := make(map[string]any, len(columns))
		for i, colName := range columns {
			val := rowValues[i]
			if b, ok := val.([]byte); ok {
				rowData[colName] = string(b)
			} else {
				rowData[colName] = val
			}
		}
		if err := handler(rowData); err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return &Result{Duration: time.Since(now), Files: files}, nil
}
