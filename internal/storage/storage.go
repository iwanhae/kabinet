package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/iwanhae/kube-event-analyzer/internal/utils"
	_ "github.com/marcboeker/go-duckdb/v2"
	corev1 "k8s.io/api/core/v1"
)

type Storage struct {
	db *sql.DB

	dbPath  string
	dataDir string

	eventCh chan *corev1.Event

	wg *sync.WaitGroup
}

func New(ctx context.Context, dbPath string) (*Storage, error) {
	dataDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Remove the WAL file
	// This is duckdb's bug that failed to recover from a crash.
	os.Remove(path.Join(dataDir, "events.db.wal"))

	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_WRITE")
	if err != nil {
		return nil, fmt.Errorf("failed to create database connection: %w", err)
	}
	if _, err := db.Exec(createTableSQL); err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	s := &Storage{
		db:      db,
		dbPath:  dbPath,
		dataDir: dataDir,
		eventCh: make(chan *corev1.Event, 2000),
		wg:      &sync.WaitGroup{},
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.runBatchInserter(ctx)
	}()

	return s, nil
}

func (s *Storage) Stats(ctx context.Context) map[string]any {
	errs := utils.MultiError{}

	dataDirSize, err := s.dataDirSize()
	if err != nil {
		errs.Add(fmt.Errorf("storage: error getting data directory size: %v", err))
		dataDirSize = 0
	}

	statsTempFiles, err := func() ([]map[string]any, error) {
		rows, err := s.db.QueryContext(ctx, "FROM duckdb_temporary_files();")
		if err != nil {
			return nil, fmt.Errorf("storage: error getting temporary files: %v", err)
		}
		defer rows.Close()
		return serializeRows(rows)
	}()
	if err != nil {
		errs.Add(err)
	}

	statsMemory, err := func() ([]map[string]any, error) {
		rows, err := s.db.QueryContext(ctx, "SELECT * FROM duckdb_memory();")
		if err != nil {
			return nil, fmt.Errorf("storage: error getting memory usage: %v", err)
		}
		defer rows.Close()
		return serializeRows(rows)
	}()
	if err != nil {
		errs.Add(err)
	}

	return map[string]any{
		"db_stats":          s.db.Stats(),
		"event_channel":     len(s.eventCh),
		"data_dir":          s.dataDir,
		"data_dir_size":     dataDirSize,
		"duckdb_temp_files": statsTempFiles,
		"duckdb_memory":     statsMemory,
		"errors":            errs,
	}
}

func (s *Storage) Wait() {
	s.wg.Wait()
	log.Println("storage: all background tasks finished")
}

func (s *Storage) Close() {
	log.Println("storage: closing storage...")
	if err := s.db.Close(); err != nil {
		log.Printf("storage: error closing database: %v", err)
	}
	log.Println("storage: closed successfully")
}

func (s *Storage) LifecycleManager(ctx context.Context, interval time.Duration, limitBytes int64) {
	log.Printf("storage: starting data lifecycle manager with interval %v and size limit %d bytes", interval, limitBytes)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Println("storage: running scheduled data lifecycle management...")
			if err := s.Archive(ctx); err != nil {
				log.Printf("storage: error during data archival: %v", err)
			}
			if err := s.EnforceRetention(limitBytes); err != nil {
				log.Printf("storage: error during retention enforcement: %v", err)
			}
		case <-ctx.Done():
			log.Println("storage: stopping data lifecycle manager.")
			return
		}
	}
}

func (s *Storage) Archive(ctx context.Context) error {

	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM kube_events").Scan(&count); err != nil {
		return fmt.Errorf("failed to count rows in kube_events: %w", err)
	}

	if count == 0 {
		log.Println("storage: no new events to archive.")
		return nil
	}

	archiveTableName := fmt.Sprintf("kube_events_archive_%d", time.Now().UnixNano())
	log.Printf("storage: archiving %d events to table %s", count, archiveTableName)

	err := func() error {
		tx, err := s.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer tx.Rollback() // Rollback on error

		if _, err := tx.ExecContext(ctx, "DROP INDEX IF EXISTS kube_events_resourceVersion_idx"); err != nil {
			return fmt.Errorf("failed to drop index before archival: %w", err)
		}

		if _, err := tx.ExecContext(ctx, fmt.Sprintf("ALTER TABLE kube_events RENAME TO %s", archiveTableName)); err != nil {
			return fmt.Errorf("failed to rename table: %w", err)
		}

		if _, err := tx.ExecContext(ctx, createTableSQL); err != nil {
			return fmt.Errorf("failed to create new kube_events table: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}

		return nil
	}()
	if err != nil {
		return fmt.Errorf("failed to swap tables: %w", err)
	}

	log.Printf("storage: successfully swapped kube_events table with %s.", archiveTableName)

	var minTime, maxTime time.Time
	query := fmt.Sprintf("SELECT MIN(lastTimestamp), MAX(lastTimestamp) FROM %s", archiveTableName)
	if err := s.db.QueryRowContext(ctx, query).Scan(&minTime, &maxTime); err != nil {
		log.Printf("storage: failed to get min/max timestamps for table %s: %v. proceeding with fallback naming", archiveTableName, err)
	}

	go s.processArchivedTable(ctx, archiveTableName, minTime, maxTime)

	return nil
}

func (s *Storage) processArchivedTable(ctx context.Context, tableName string, minTime, maxTime time.Time) {
	log.Printf("storage: archiving: starting background processing for table: %s", tableName)

	parquetFileName := filepath.Join(s.dataDir, fmt.Sprintf("events_%d_%d.parquet", minTime.Unix(), maxTime.Unix()))

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		log.Printf("storage: archiving: error getting connection for background processing: %v", err)
		return
	}
	defer tx.Rollback()

	copySQL := fmt.Sprintf("COPY %s TO '%s' (FORMAT 'parquet', COMPRESSION 'zstd');", tableName, parquetFileName)
	if _, err := tx.ExecContext(ctx, copySQL); err != nil {
		log.Printf("storage: archiving: error exporting table %s to parquet: %v", tableName, err)
		// Don't drop the table if copy fails, to allow for manual recovery
		return
	}
	log.Printf("storage: archiving: successfully exported table %s to %s", tableName, parquetFileName)

	dropSQL := fmt.Sprintf("DROP TABLE %s", tableName)
	if _, err := tx.ExecContext(ctx, dropSQL); err != nil {
		log.Printf("storage: archiving: error dropping archived table %s: %v", tableName, err)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("storage: archiving: error committing transaction: %v", err)
		return
	}

	log.Printf("storage: archiving: successfully dropped archived table %s", tableName)
}

func (s *Storage) dataDirSize() (int64, error) {
	files, err := os.ReadDir(s.dataDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read data directory: %w", err)
	}

	var totalSize int64
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			log.Printf("storage: could not get file info for %s: %v", file.Name(), err)
			continue
		}
		totalSize += info.Size()
	}

	return totalSize, nil
}

func (s *Storage) EnforceRetention(limitBytes int64) error {
	files, err := os.ReadDir(s.dataDir)
	if err != nil {
		return fmt.Errorf("failed to read data directory: %w", err)
	}

	var parquetFiles []os.DirEntry
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".parquet") {
			parquetFiles = append(parquetFiles, file)
		}
	}

	// Sort by the timestamp in the filename, oldest first.
	sort.Slice(parquetFiles, func(i, j int) bool {
		tsI := extractTimestampFromName(parquetFiles[i].Name())
		tsJ := extractTimestampFromName(parquetFiles[j].Name())
		// If timestamps are equal or couldn't be parsed, fallback to name comparison
		if tsI == tsJ {
			return parquetFiles[i].Name() < parquetFiles[j].Name()
		}
		return tsI < tsJ
	})

	totalSize, err := s.dataDirSize()
	if err != nil {
		return fmt.Errorf("failed to get data directory size: %w", err)
	}

	log.Printf("storage: current parquet files size: %d bytes. Limit: %d bytes.", totalSize, limitBytes)

	for totalSize > limitBytes {
		if len(parquetFiles) == 0 {
			break
		}
		oldestFile := parquetFiles[0]
		parquetFiles = parquetFiles[1:]

		info, err := oldestFile.Info()
		if err != nil {
			log.Printf("storage: could not get file info for deletion candidate %s: %v", oldestFile.Name(), err)
			continue
		}

		filePath := filepath.Join(s.dataDir, oldestFile.Name())
		if err := os.Remove(filePath); err != nil {
			log.Printf("storage: failed to delete oldest parquet file %s: %v", filePath, err)
			// Stop trying to delete if one fails
			break
		}

		totalSize -= info.Size()
		log.Printf("storage: deleted oldest parquet file: %s. New total size: %d bytes", oldestFile.Name(), totalSize)
	}

	return nil
}

// extractTimestampFromName parses a parquet filename and returns its start timestamp (in Unix seconds)
// for sorting purposes. It supports "events_MIN_MAX.parquet"
func extractTimestampFromName(filename string) int64 {
	minTs, _, ok := parseParquetFilename(filename)
	if !ok {
		log.Printf("storage: could not extract timestamp from filename: %s", filename)
		return 0 // Place unparsable files at the beginning, though they are unlikely to be sorted correctly.
	}
	return minTs
}

// parseParquetFilename extracts the min and max unix timestamps from a parquet filename.
// It returns minTs, maxTs, and a boolean indicating success.
func parseParquetFilename(filename string) (int64, int64, bool) {
	base := strings.TrimSuffix(filename, ".parquet")
	parts := strings.Split(base, "_")

	if len(parts) < 2 {
		return 0, 0, false
	}

	switch parts[0] {
	case "events":
		if len(parts) != 3 {
			return 0, 0, false
		}
		minTs, errMin := strconv.ParseInt(parts[1], 10, 64)
		maxTs, errMax := strconv.ParseInt(parts[2], 10, 64)
		if errMin != nil || errMax != nil {
			return 0, 0, false
		}
		return minTs, maxTs, true
	case "kube":
		if len(parts) == 4 && parts[1] == "events" && parts[2] == "archive" {
			// Fallback filename format: kube_events_archive_<nanos>.parquet
			nanoTs, err := strconv.ParseInt(parts[3], 10, 64)
			if err != nil {
				return 0, 0, false
			}
			ts := nanoTs / 1e9 // convert nano to unix seconds
			return ts, ts, true
		}
	}
	return 0, 0, false
}

type ParquetFileInfo struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

type RangeQueryResult struct {
	Duration time.Duration
	Files    []ParquetFileInfo
}

// RangeQuery executes a range query against the storage.
// It executes the query by substituting the $events placeholder with the appropriate FROM clause.
func (s *Storage) RangeQuery(ctx context.Context, query string, start, end time.Time) ([]map[string]any, *RangeQueryResult, error) {
	if ctx.Err() != nil {
		return nil, nil, fmt.Errorf("failed fast: %w", ctx.Err())
	}

	files, err := os.ReadDir(s.dataDir)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read data directory: %w", err) // TODO: retry?
	}
	queryStartTs := start.Unix()
	queryEndTs := end.Unix()

	var relevantFiles []ParquetFileInfo
	var latestParquetMaxTs int64

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".parquet") {
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

		// File range overlaps with query range if:
		// (file_start <= query_end) AND (file_end >= query_start)
		if maxTs >= queryStartTs && minTs <= queryEndTs {
			relevantFiles = append(relevantFiles, ParquetFileInfo{
				Path: filepath.Join(s.dataDir, file.Name()),
				Size: info.Size(),
			})
		}
	}

	// Determine if we need to query the live kube_events table.
	// If the user's query range ends before the last archived event, we can skip it.
	includeKubeEvents := true
	if latestParquetMaxTs > 0 && queryEndTs < latestParquetMaxTs {
		includeKubeEvents = false
	}

	relevantFilePaths := make([]string, len(relevantFiles))
	for i, f := range relevantFiles {
		relevantFilePaths[i] = f.Path
	}

	fromClause, err := buildFromClause(relevantFilePaths, includeKubeEvents)
	if err != nil {
		log.Println("storage: query time range resulted in no data sources. returning empty result.")
		return nil, nil, fmt.Errorf("query time range resulted in no data sources")
	}

	finalQuery := strings.Replace(query, "$events", fromClause, 1)
	log.Printf("storage: executing range query: %s", finalQuery)

	now := time.Now()
	rows, err := s.db.QueryContext(ctx, finalQuery)
	if err != nil {
		return nil, nil, err
	}
	results, err := serializeRows(rows)
	if err != nil {
		return nil, nil, err
	}
	return results, &RangeQueryResult{
		Duration: time.Since(now),
		Files:    relevantFiles,
	}, nil
}

func buildFromClause(relevantFiles []string, includeKubeEvents bool) (string, error) {
	var fromSources []string
	if includeKubeEvents {
		fromSources = append(fromSources, "SELECT * FROM kube_events")
	}

	if len(relevantFiles) > 0 {
		quotedFiles := make([]string, len(relevantFiles))
		for i, p := range relevantFiles {
			quotedFiles[i] = fmt.Sprintf("'%s'", p)
		}
		parquetSource := fmt.Sprintf("SELECT * FROM read_parquet([%s])", strings.Join(quotedFiles, ", "))
		fromSources = append(fromSources, parquetSource)
	}

	if len(fromSources) == 0 {
		return "", fmt.Errorf("no data sources for query")
	}

	return fmt.Sprintf("(%s)", strings.Join(fromSources, " UNION BY NAME ")), nil
}

func (s *Storage) AppendEvent(ctx context.Context, k8sEvent *corev1.Event) error {
	select {
	case s.eventCh <- k8sEvent:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context cancelled")
	}
}

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

func serializeRows(rows *sql.Rows) ([]map[string]any, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	var results []map[string]any
	for rows.Next() {
		rowValues := make([]any, len(columns))
		rowPointers := make([]any, len(columns))
		for i := range rowValues {
			rowPointers[i] = &rowValues[i]
		}

		if err := rows.Scan(rowPointers...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		rowData := make(map[string]any, len(columns))
		for i, colName := range columns {
			val := rowValues[i]

			// To keep JSON clean, we handle byte slices (like DuckDB structs)
			if b, ok := val.([]byte); ok {
				rowData[colName] = string(b)
			} else {
				rowData[colName] = val
			}
		}
		results = append(results, rowData)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}
	return results, nil
}
