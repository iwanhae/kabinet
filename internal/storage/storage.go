package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/marcboeker/go-duckdb/v2"
	corev1 "k8s.io/api/core/v1"
)

type Storage struct {
	writer *sql.DB
	reader *sql.DB

	dataDir   string
	archiveMu sync.Mutex
	eventCh   chan *corev1.Event

	wg *sync.WaitGroup
}

func New(ctx context.Context, dbPath string) (*Storage, error) {
	dataDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	writer, err := sql.Open("duckdb", dbPath+"?access_mode=READ_WRITE&threads=2")
	if err != nil {
		return nil, fmt.Errorf("failed to create writer: %w", err)
	}
	if _, err := writer.Exec(createTableSQL); err != nil {
		writer.Close()
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	reader, err := sql.Open("duckdb", ":memory:?threads=2")
	if err != nil {
		writer.Close()
		return nil, fmt.Errorf("failed to create reader: %w", err)
	}
	if _, err := reader.Exec(fmt.Sprintf("ATTACH '%s' AS read_only", dbPath)); err != nil {
		reader.Close()
		writer.Close()
		return nil, fmt.Errorf("failed to attach database: %w", err)
	}

	s := &Storage{
		writer:  writer,
		reader:  reader,
		dataDir: dataDir,
		eventCh: make(chan *corev1.Event, 10000),
		wg:      &sync.WaitGroup{},
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.runBatchInserter(ctx)
	}()

	return s, nil
}

func (s *Storage) Wait() {
	s.wg.Wait()
	log.Println("storage: all background tasks finished")
}

func (s *Storage) Close() {
	log.Println("storage: closing storage...")
	if err := s.writer.Close(); err != nil {
		log.Printf("storage: error closing database: %v", err)
	}
	if err := s.reader.Close(); err != nil {
		log.Printf("storage: error closing database: %v", err)
	}
	log.Println("storage: closed successfully")
}

func (s *Storage) ManageDataLifecycle(ctx context.Context, interval time.Duration, limitBytes int64) {
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
	s.archiveMu.Lock()
	defer s.archiveMu.Unlock()

	var count int
	if err := s.reader.QueryRowContext(ctx, "SELECT COUNT(*) FROM kube_events").Scan(&count); err != nil {
		return fmt.Errorf("failed to count rows in kube_events: %w", err)
	}

	if count == 0 {
		log.Println("storage: no new events to archive.")
		return nil
	}

	archiveTableName := fmt.Sprintf("kube_events_archive_%d", time.Now().UnixNano())
	log.Printf("storage: archiving %d events to table %s", count, archiveTableName)

	tx, err := s.writer.Begin()
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

	log.Printf("storage: successfully swapped kube_events table with %s.", archiveTableName)

	var minTime, maxTime time.Time
	query := fmt.Sprintf("SELECT MIN(metadata.creationTimestamp), MAX(metadata.creationTimestamp) FROM %s", archiveTableName)
	if err := s.reader.QueryRowContext(ctx, query).Scan(&minTime, &maxTime); err != nil {
		log.Printf("storage: failed to get min/max timestamps for table %s: %v. proceeding with fallback naming", archiveTableName, err)
	}

	go s.processArchivedTable(archiveTableName, minTime, maxTime)

	return nil
}

func (s *Storage) processArchivedTable(tableName string, minTime, maxTime time.Time) {
	log.Printf("storage: starting background processing for table: %s", tableName)

	parquetFileName := filepath.Join(s.dataDir, fmt.Sprintf("events_%d_%d.parquet", minTime.Unix(), maxTime.Unix()))

	// Using a new connection for the background task
	conn, err := s.reader.Conn(context.Background())
	if err != nil {
		log.Printf("storage: error getting connection for background processing: %v", err)
		return
	}
	defer conn.Close()

	copySQL := fmt.Sprintf("COPY %s TO '%s' (FORMAT 'parquet', COMPRESSION 'zstd');", tableName, parquetFileName)
	if _, err := conn.ExecContext(context.Background(), copySQL); err != nil {
		log.Printf("storage: error exporting table %s to parquet: %v", tableName, err)
		// Don't drop the table if copy fails, to allow for manual recovery
		return
	}
	log.Printf("storage: successfully exported table %s to %s", tableName, parquetFileName)

	dropSQL := fmt.Sprintf("DROP TABLE %s", tableName)
	if _, err := conn.ExecContext(context.Background(), dropSQL); err != nil {
		log.Printf("storage: error dropping archived table %s: %v", tableName, err)
		return
	}
	log.Printf("storage: successfully dropped archived table %s", tableName)
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

	var totalSize int64
	for _, file := range parquetFiles {
		info, err := file.Info()
		if err != nil {
			log.Printf("storage: could not get file info for %s: %v", file.Name(), err)
			continue
		}
		totalSize += info.Size()
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

// RangeQuery executes a range query against the storage.
// It executes the query by substituting the $events placeholder with the appropriate FROM clause.
func (s *Storage) RangeQuery(ctx context.Context, query string, start, end time.Time) (*sql.Rows, error) {
	files, err := os.ReadDir(s.dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read data directory: %w", err)
	}

	queryStartTs := start.Unix()
	queryEndTs := end.Unix()

	var relevantFiles []string
	var latestParquetMaxTs int64

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".parquet") {
			continue
		}

		minTs, maxTs, ok := parseParquetFilename(file.Name())
		if !ok {
			log.Printf("storage: could not parse filename %s, including it just in case.", file.Name())
			relevantFiles = append(relevantFiles, filepath.Join(s.dataDir, file.Name()))
			continue
		}

		if maxTs > latestParquetMaxTs {
			latestParquetMaxTs = maxTs
		}

		// File range overlaps with query range if:
		// (file_start <= query_end) AND (file_end >= query_start)
		if maxTs >= queryStartTs && minTs <= queryEndTs {
			relevantFiles = append(relevantFiles, filepath.Join(s.dataDir, file.Name()))
		}
	}

	// Determine if we need to query the live kube_events table.
	// If the user's query range ends before the last archived event, we can skip it.
	includeKubeEvents := true
	if latestParquetMaxTs > 0 && queryEndTs < latestParquetMaxTs {
		includeKubeEvents = false
	}

	fromClause, err := buildFromClause(relevantFiles, includeKubeEvents)
	if err != nil {
		log.Println("storage: query time range resulted in no data sources. returning empty result.")
		return s.reader.QueryContext(ctx, "SELECT * FROM kube_events WHERE 1=0") // Return empty
	}

	finalQuery := strings.Replace(query, "$events", fromClause, 1)
	log.Printf("storage: executing range query: %s", finalQuery)

	return s.reader.QueryContext(ctx, finalQuery)
}

func buildFromClause(relevantFiles []string, includeKubeEvents bool) (string, error) {
	var fromSources []string
	if includeKubeEvents {
		fromSources = append(fromSources, "SELECT * FROM read_only.kube_events")
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
	tx, err := s.writer.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
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
