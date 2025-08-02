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

// Writer handles event writing, archiving, and data retention policies.
type Writer struct {
	mu      sync.Mutex
	db      *sql.DB
	dbPath  string
	dataDir string

	eventCh chan *corev1.Event
	closeCh chan struct{}
	wg      sync.WaitGroup
	closed  bool
}

// NewWriter creates and initializes a new Writer instance.
// It starts its own background goroutine for batch processing which is stopped when Close() is called.
func NewWriter(dbPath string, dataDir string) (*Writer, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// This is a workaround for a DuckDB bug that can fail recovery from a crash.
	os.Remove(dbPath + ".wal")

	db, err := sql.Open("duckdb", dbPath+"?access_mode=READ_WRITE")
	if err != nil {
		return nil, fmt.Errorf("failed to create writer connection: %w", err)
	}
	if _, err := db.Exec(SQLCreateEventsTable); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	w := &Writer{
		db:      db,
		dbPath:  dbPath,
		dataDir: dataDir,
		eventCh: make(chan *corev1.Event, 10000),
		closeCh: make(chan struct{}),
		wg:      sync.WaitGroup{},
	}

	w.wg.Add(1)
	go w.runBatchInserter()

	return w, nil
}

// AppendEvent sends a Kubernetes event to the batch-processing channel.
func (w *Writer) AppendEvent(k8sEvent *corev1.Event) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return fmt.Errorf("writer is closed")
	}

	select {
	case w.eventCh <- k8sEvent:
		return nil
	default:
		return fmt.Errorf("event channel is full")
	}
}

// runBatchInserter collects events from the channel and periodically inserts them into the database.
func (w *Writer) runBatchInserter() {
	defer w.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	batch := make([]*corev1.Event, 0, 1000)

	for {
		select {
		case <-w.closeCh:
			log.Println("writer: shutdown signal received, flushing remaining events...")
			// Drain any remaining events from the channel.
			for len(w.eventCh) > 0 {
				batch = append(batch, <-w.eventCh)
			}
			if len(batch) > 0 {
				if err := w.appendEvents(batch); err != nil {
					log.Printf("writer: error appending remaining events during shutdown: %v", err)
				}
			}
			return
		case event := <-w.eventCh:
			batch = append(batch, event)
			if len(batch) >= 1000 {
				if err := w.appendEvents(batch); err != nil {
					log.Printf("writer: error appending events: %v", err)
				}
				batch = make([]*corev1.Event, 0, 1000) // Reset batch
			}
		case <-ticker.C:
			if len(batch) > 0 {
				if err := w.appendEvents(batch); err != nil {
					log.Printf("writer: error appending events on tick: %v", err)
				}
				batch = make([]*corev1.Event, 0, 1000) // Reset batch
			}
		}
	}
}

// appendEvents inserts a batch of events into the database within a single transaction.
func (w *Writer) appendEvents(k8sEvents []*corev1.Event) error {
	if len(k8sEvents) == 0 {
		return nil
	}

	tx, err := w.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(fmt.Sprintf(SQLInsertEventValuesTemplate, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"))
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()

	for _, k8sEvent := range k8sEvents {
		// Struct conversions for DuckDB
		var series, related any
		if k8sEvent.Series != nil {
			series = map[string]any{"count": k8sEvent.Series.Count, "lastObservedTime": k8sEvent.Series.LastObservedTime.Time}
		}
		if k8sEvent.Related != nil {
			related = map[string]any{"kind": k8sEvent.Related.Kind, "namespace": k8sEvent.Related.Namespace, "name": k8sEvent.Related.Name, "uid": string(k8sEvent.Related.UID), "apiVersion": k8sEvent.Related.APIVersion, "resourceVersion": k8sEvent.Related.ResourceVersion, "fieldPath": k8sEvent.Related.FieldPath}
		}

		args := []any{
			k8sEvent.Kind, k8sEvent.APIVersion,
			map[string]any{"name": k8sEvent.ObjectMeta.Name, "namespace": k8sEvent.ObjectMeta.Namespace, "uid": string(k8sEvent.ObjectMeta.UID), "resourceVersion": k8sEvent.ObjectMeta.ResourceVersion, "creationTimestamp": k8sEvent.ObjectMeta.CreationTimestamp.Time},
			map[string]any{"kind": k8sEvent.InvolvedObject.Kind, "namespace": k8sEvent.InvolvedObject.Namespace, "name": k8sEvent.InvolvedObject.Name, "uid": string(k8sEvent.InvolvedObject.UID), "apiVersion": k8sEvent.InvolvedObject.APIVersion, "resourceVersion": k8sEvent.InvolvedObject.ResourceVersion, "fieldPath": k8sEvent.InvolvedObject.FieldPath},
			k8sEvent.Reason, k8sEvent.Message,
			map[string]any{"component": k8sEvent.Source.Component, "host": k8sEvent.Source.Host},
			k8sEvent.FirstTimestamp.Time, k8sEvent.LastTimestamp.Time,
			k8sEvent.Count, k8sEvent.Type, k8sEvent.EventTime.Time,
			series, k8sEvent.Action, related,
			k8sEvent.ReportingController, k8sEvent.ReportingInstance,
		}
		if _, err := stmt.Exec(args...); err != nil {
			// It's an IGNORE statement, so errors here are likely connection issues, not constraint violations.
			return fmt.Errorf("failed to execute insert statement: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("writer: inserted %d events into kube_events", len(k8sEvents))
	return nil
}

// LifecycleManager runs background tasks for archiving and retention.
func (w *Writer) LifecycleManager(ctx context.Context, interval time.Duration, limitBytes int64) {
	log.Printf("writer: starting data lifecycle manager with interval %v and size limit %d bytes", interval, limitBytes)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Println("writer: running scheduled data lifecycle management...")
			if err := w.Archive(ctx); err != nil {
				log.Printf("writer: error during data archival: %v", err)
			}
			if err := w.EnforceRetention(limitBytes); err != nil {
				log.Printf("writer: error during retention enforcement: %v", err)
			}
		case <-w.closeCh:
			log.Println("writer: stopping data lifecycle manager.")
			return
		case <-ctx.Done():
			log.Println("writer: stopping data lifecycle manager due to context cancellation.")
			return
		}
	}
}

// Archive moves events from the live DuckDB table to a Parquet file.
func (w *Writer) Archive(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return fmt.Errorf("writer is closed")
	}

	var count int
	if err := w.db.QueryRowContext(ctx, SQLCountKubeEvents).Scan(&count); err != nil {
		return fmt.Errorf("failed to count rows in kube_events: %w", err)
	}

	if count == 0 {
		log.Println("writer: no new events to archive.")
		return nil
	}

	archiveTableName := fmt.Sprintf("kube_events_archive_%d", time.Now().UnixNano())
	log.Printf("writer: archiving %d events to table %s", count, archiveTableName)

	tx, err := w.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Efficiently swap tables
	if _, err := tx.ExecContext(ctx, SQLDropResourceVersionIndex); err != nil {
		return fmt.Errorf("failed to drop index before archival: %w", err)
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(SQLRenameTableToTemplate, archiveTableName)); err != nil {
		return fmt.Errorf("failed to rename table: %w", err)
	}
	if _, err := tx.ExecContext(ctx, SQLCreateEventsTable); err != nil {
		return fmt.Errorf("failed to create new kube_events table: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("writer: successfully swapped kube_events table with %s.", archiveTableName)

	// Process the archived table in the background
	w.wg.Add(1)
	go w.processArchivedTable(context.Background(), archiveTableName)
	return nil
}

// processArchivedTable handles the conversion of an archived table to a Parquet file.
func (w *Writer) processArchivedTable(ctx context.Context, tableName string) {
	defer w.wg.Done()
	log.Printf("writer: archiving: starting background processing for table: %s", tableName)

	var minTime, maxTime time.Time
	query := fmt.Sprintf(SQLSelectMinMaxTimestampTemplate, tableName)
	if err := w.db.QueryRowContext(ctx, query).Scan(&minTime, &maxTime); err != nil {
		log.Printf("writer: failed to get min/max timestamps for table %s: %v. Using current time for filename.", tableName, err)
		minTime, maxTime = time.Now(), time.Now()
	}

	parquetFileName := filepath.Join(w.dataDir, fmt.Sprintf("events_%d_%d.parquet", minTime.Unix(), maxTime.Unix()))

	copySQL := fmt.Sprintf(SQLCopyToParquetTemplate, tableName, parquetFileName)
	if _, err := w.db.ExecContext(ctx, copySQL); err != nil {
		log.Printf("writer: archiving: error exporting table %s to parquet: %v", tableName, err)
		// We don't drop the table if copy fails, allowing for manual recovery or retry.
		return
	}
	log.Printf("writer: archiving: successfully exported table %s to %s", tableName, parquetFileName)

	dropSQL := fmt.Sprintf(SQLDropTableTemplate, tableName)
	if _, err := w.db.ExecContext(ctx, dropSQL); err != nil {
		log.Printf("writer: archiving: error dropping archived table %s: %v", tableName, err)
		return
	}

	log.Printf("writer: archiving: successfully dropped archived table %s", tableName)
}

// EnforceRetention checks the total size of Parquet files and deletes the oldest ones if the limit is exceeded.
func (w *Writer) EnforceRetention(limitBytes int64) error {
	files, err := os.ReadDir(w.dataDir)
	if err != nil {
		return fmt.Errorf("failed to read data directory: %w", err)
	}

	var parquetFiles []os.DirEntry
	var totalSize int64
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".parquet") {
			parquetFiles = append(parquetFiles, file)
			info, err := file.Info()
			if err == nil {
				totalSize += info.Size()
			}
		}
	}

	log.Printf("writer: current parquet files size: %d bytes. Limit: %d bytes.", totalSize, limitBytes)
	if totalSize <= limitBytes {
		return nil
	}

	// Sort by the start timestamp in the filename, oldest first.
	sort.Slice(parquetFiles, func(i, j int) bool {
		tsI, _, okI := parseParquetFilename(parquetFiles[i].Name())
		tsJ, _, okJ := parseParquetFilename(parquetFiles[j].Name())
		if !okI || !okJ || tsI == tsJ {
			return parquetFiles[i].Name() < parquetFiles[j].Name() // Fallback to name sort
		}
		return tsI < tsJ
	})

	for totalSize > limitBytes {
		if len(parquetFiles) == 0 {
			break
		}
		oldestFile := parquetFiles[0]
		parquetFiles = parquetFiles[1:]

		info, err := oldestFile.Info()
		if err != nil {
			log.Printf("writer: could not get file info for deletion candidate %s: %v", oldestFile.Name(), err)
			continue
		}

		filePath := filepath.Join(w.dataDir, oldestFile.Name())
		if err := os.Remove(filePath); err != nil {
			log.Printf("writer: failed to delete oldest parquet file %s. Error: %v", filePath, err)
			break // Stop if a deletion fails
		}

		totalSize -= info.Size()
		log.Printf("writer: deleted oldest parquet file: %s. New total size: %d bytes", oldestFile.Name(), totalSize)
	}
	return nil
}

// Close gracefully shuts down the writer.
func (w *Writer) Close() {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return
	}
	log.Println("writer: closing...")
	w.closed = true
	close(w.closeCh)
	w.mu.Unlock()

	w.wg.Wait()
	if err := w.db.Close(); err != nil {
		log.Printf("writer: error closing writer db: %v", err)
	}
	log.Println("writer: closed.")
}

// Stats returns statistics about the writer component.
func (w *Writer) Stats() map[string]any {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return nil
	}

	var totalSize int64
	files, err := os.ReadDir(w.dataDir)
	if err == nil {
		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".parquet") {
				info, err := file.Info()
				if err == nil {
					totalSize += info.Size()
				}
			}
		}
	}

	return map[string]any{
		"event_channel_len":  len(w.eventCh),
		"db_path":            w.dbPath,
		"parquet_path":       w.dataDir,
		"parquet_size_bytes": totalSize,
		"db_stats":           w.db.Stats(),
	}
}

// parseParquetFilename extracts the min and max unix timestamps from a parquet filename.
// It expects the format "events_MIN_MAX.parquet".
func parseParquetFilename(filename string) (int64, int64, bool) {
	base := strings.TrimSuffix(filename, ".parquet")
	parts := strings.Split(base, "_")
	if len(parts) != 3 || parts[0] != "events" {
		return 0, 0, false
	}
	minTs, errMin := time.Parse(time.RFC3339, parts[1])
	maxTs, errMax := time.Parse(time.RFC3339, parts[2])
	if errMin != nil || errMax != nil {
		// Fallback for old numeric timestamp format
		minTsUnix, errMin := strconv.ParseInt(parts[1], 10, 64)
		maxTsUnix, errMax := strconv.ParseInt(parts[2], 10, 64)
		if errMin != nil || errMax != nil {
			return 0, 0, false
		}
		return minTsUnix, maxTsUnix, true
	}
	return minTs.Unix(), maxTs.Unix(), true
}
