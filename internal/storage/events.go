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

// AppendEvents inserts a batch of Kubernetes events into the database
func (s *Storage) AppendEvents(k8sEvents []*corev1.Event) error {
	if len(k8sEvents) == 0 {
		return nil
	}

	// BEGIN TRANSACTION
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err) // TODO: retry?
	}
	defer tx.Rollback()

	for _, k8sEvent := range k8sEvents {
		var series any
		if k8sEvent.Series != nil {
			series = map[string]any{
				"count":            k8sEvent.Series.Count,
				"lastObservedTime": k8sEvent.Series.LastObservedTime.Time,
			}
		} else {
			series = nil
		}

		var related any
		if k8sEvent.Related != nil {
			related = map[string]any{
				"kind":            k8sEvent.Related.Kind,
				"namespace":       k8sEvent.Related.Namespace,
				"name":            k8sEvent.Related.Name,
				"uid":             string(k8sEvent.Related.UID),
				"apiVersion":      k8sEvent.Related.APIVersion,
				"resourceVersion": k8sEvent.Related.ResourceVersion,
				"fieldPath":       k8sEvent.Related.FieldPath,
			}
		} else {
			related = nil
		}

		args := []any{
			k8sEvent.Kind,
			k8sEvent.APIVersion,
			map[string]any{
				"name":              k8sEvent.ObjectMeta.Name,
				"namespace":         k8sEvent.ObjectMeta.Namespace,
				"uid":               string(k8sEvent.ObjectMeta.UID),
				"resourceVersion":   k8sEvent.ObjectMeta.ResourceVersion,
				"creationTimestamp": k8sEvent.ObjectMeta.CreationTimestamp.Time,
			},
			map[string]any{
				"kind":            k8sEvent.InvolvedObject.Kind,
				"namespace":       k8sEvent.InvolvedObject.Namespace,
				"name":            k8sEvent.InvolvedObject.Name,
				"uid":             string(k8sEvent.InvolvedObject.UID),
				"apiVersion":      k8sEvent.InvolvedObject.APIVersion,
				"resourceVersion": k8sEvent.InvolvedObject.ResourceVersion,
				"fieldPath":       k8sEvent.InvolvedObject.FieldPath,
			},
			k8sEvent.Reason,
			k8sEvent.Message,
			map[string]any{
				"component": k8sEvent.Source.Component,
				"host":      k8sEvent.Source.Host,
			},
			k8sEvent.FirstTimestamp.Time,
			k8sEvent.LastTimestamp.Time,
			k8sEvent.Count,
			k8sEvent.Type,
			k8sEvent.EventTime.Time,
			series,
			k8sEvent.Action,
			related,
			k8sEvent.ReportingController,
			k8sEvent.ReportingInstance,
		}
		placeholder := "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
		query := fmt.Sprintf("INSERT OR IGNORE INTO kube_events VALUES %s", placeholder)
		_, err := tx.Exec(query, args...)
		if err != nil {
			return fmt.Errorf("failed to batch insert events: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("storage: inserted %d events into kube_events", len(k8sEvents))

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

	var relevantFiles []ParquetFileInfo
	var latestParquetMaxTs int64

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".parquet") || !strings.HasPrefix(file.Name(), "events_") {
			continue
		}

		info, err := file.Info()
		if err != nil {
			log.Printf("storage: could not get file info for %s: %v", file.Name(), err)
			continue
		}

		minTs, maxTs, ok := parseParquetFilename(file.Name())
		if !ok {
			log.Printf("storage: could not parse filename %s, including it just in case.", file.Name())
			relevantFiles = append(relevantFiles, ParquetFileInfo{
				Path: filepath.Join(s.dataDir, file.Name()),
				Size: info.Size(),
			})
			continue
		}

		if maxTs > latestParquetMaxTs {
			latestParquetMaxTs = maxTs
		}

		if maxTs >= queryStartTs && minTs <= queryEndTs {
			relevantFiles = append(relevantFiles, ParquetFileInfo{
				Path: filepath.Join(s.dataDir, file.Name()),
				Size: info.Size(),
			})
		}
	}

	includeKubeEvents := true
	if latestParquetMaxTs > 0 && queryEndTs < latestParquetMaxTs {
		includeKubeEvents = false
	}

	relevantFilePaths := make([]string, len(relevantFiles))
	for i, f := range relevantFiles {
		relevantFilePaths[i] = f.Path
	}

	fromClause, err := buildFromClause(relevantFilePaths, includeKubeEvents, start, end)
	if err != nil {
		log.Println("storage: query time range resulted in no data sources. returning empty result.")
		return "", nil, fmt.Errorf("query time range resulted in no data sources")
	}

	finalQuery := strings.Replace(query, "$events", fromClause, 1)
	return finalQuery, relevantFiles, nil
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
