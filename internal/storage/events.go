package storage

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
)

// AppendEvent adds a single event to the storage channel
func (s *Storage) AppendEvent(ctx context.Context, k8sEvent *corev1.Event) error {
	select {
	case s.eventCh <- k8sEvent:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context cancelled")
	}
}

// runBatchInserter runs the background batch inserter goroutine
func (s *Storage) runBatchInserter(ctx context.Context) {
	time.Sleep(time.Duration(5-time.Now().Second()%5) * time.Second) // no special reason for this, just to make logs easier to read
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	batch := make([]*corev1.Event, 0, 1000)

	for {
		select {
		case <-ctx.Done():
			log.Println("storage: context cancelled, flushing remaining events...")
			if len(batch) > 0 {
				if err := s.AppendEvents(batch); err != nil {
					log.Printf("storage: error appending remaining events: %v", err)
				}
			}
			s.Close()
			return
		case event := <-s.eventCh:
			batch = append(batch, event)
			if len(batch) >= 1000 {
				if err := s.AppendEvents(batch); err != nil {
					log.Printf("storage: error appending events: %v", err)
				}
				batch = make([]*corev1.Event, 0, 1000)
			}
		case <-ticker.C:
			if len(batch) > 0 {
				if err := s.AppendEvents(batch); err != nil {
					log.Printf("storage: error appending events on tick: %v", err)
				}
				batch = make([]*corev1.Event, 0, 1000)
			}
		}
	}
}

// AppendEvents writes a batch of Kubernetes events to JSONL files
func (s *Storage) AppendEvents(k8sEvents []*corev1.Event) error {
	if len(k8sEvents) == 0 {
		return nil
	}

	for _, event := range k8sEvents {
		if err := s.jsonlWriter.WriteEvent(event); err != nil {
			return fmt.Errorf("failed to write event to JSONL: %w", err)
		}
	}

	log.Printf("storage: wrote %d events to JSONL", len(k8sEvents))
	return nil
}

// buildEventsQuery constructs the final SQL query for $events within the given time range.
// It returns the query with the $events macro replaced by the appropriate FROM clause
// and the list of Parquet files involved.
func (s *Storage) buildEventsQuery(query string, start, end time.Time) (string, []ParquetFileInfo, error) {
	files, err := os.ReadDir(s.dataDir)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read data directory: %w", err)
	}

	queryStartTs := start.Unix()
	queryEndTs := end.Unix()

	var jsonlFiles []string
	var parquetFiles []ParquetFileInfo

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Handle JSONL files
		if strings.HasSuffix(file.Name(), ".jsonl") && strings.HasPrefix(file.Name(), "events_") {
			// For simplicity, include all JSONL files in queries
			// The WHERE clause will filter by time range
			jsonlFiles = append(jsonlFiles, filepath.Join(s.dataDir, file.Name()))
			continue
		}

		// Handle Parquet files
		if strings.HasSuffix(file.Name(), ".parquet") && strings.HasPrefix(file.Name(), "events_") {
			info, err := file.Info()
			if err != nil {
				log.Printf("storage: could not get file info for %s: %v", file.Name(), err)
				continue
			}

			minTs, maxTs, ok := parseParquetFilename(file.Name())
			if !ok {
				log.Printf("storage: could not parse filename %s, including it just in case.", file.Name())
				parquetFiles = append(parquetFiles, ParquetFileInfo{
					Path: filepath.Join(s.dataDir, file.Name()),
					Size: info.Size(),
				})
				continue
			}

			// Only include parquet files that overlap with query time range
			if maxTs >= queryStartTs && minTs <= queryEndTs {
				parquetFiles = append(parquetFiles, ParquetFileInfo{
					Path: filepath.Join(s.dataDir, file.Name()),
					Size: info.Size(),
				})
			}
		}
	}

	// Build file paths for buildFromClause
	parquetFilePaths := make([]string, len(parquetFiles))
	for i, f := range parquetFiles {
		parquetFilePaths[i] = f.Path
	}

	fromClause, err := buildFromClause(jsonlFiles, parquetFilePaths, start, end)
	if err != nil {
		log.Println("storage: query time range resulted in no data sources. returning empty result.")
		return "", nil, fmt.Errorf("query time range resulted in no data sources")
	}

	finalQuery := strings.Replace(query, "$events", fromClause, 1)
	return finalQuery, parquetFiles, nil
}

// StreamEvents executes the built events query and streams each row to the provided handler
// without loading all rows into memory.
func (s *Storage) StreamEvents(ctx context.Context, where string, start, end time.Time, handler func(map[string]any) error) (*RangeQueryResult, error) {
	baseQuery := "SELECT * FROM $events"
	if strings.TrimSpace(where) != "" {
		baseQuery += " WHERE " + where
	}
	baseQuery += " ORDER BY lastTimestamp"

	finalQuery, files, err := s.buildEventsQuery(baseQuery, start, end)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	rows, err := s.db.QueryContext(ctx, finalQuery)
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

	return &RangeQueryResult{Duration: time.Since(now), Files: files}, nil
}
