package storage

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/marcboeker/go-duckdb/v2"
	corev1 "k8s.io/api/core/v1"
)

type Storage struct {
	db        *sql.DB
	conn      driver.Conn
	appender  *duckdb.Appender
	dataDir   string
	archiveMu sync.Mutex
}

func New(ctx context.Context, dbPath string) (*Storage, error) {
	dataDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	connector, err := duckdb.NewConnector(dbPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create duckdb connector: %w", err)
	}

	db := sql.OpenDB(connector)
	if _, err := db.Exec(createTableSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	conn, err := connector.Connect(ctx)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	appender, err := duckdb.NewAppenderFromConn(conn, "", "kube_events")
	if err != nil {
		conn.Close()
		db.Close()
		return nil, fmt.Errorf("failed to create appender: %w", err)
	}

	return &Storage{
		db:       db,
		conn:     conn,
		appender: appender,
		dataDir:  dataDir,
	}, nil
}

func (s *Storage) Close() {
	s.appender.Close()
	s.conn.Close()
	s.db.Close()
}

func (s *Storage) ManageDataLifecycle(ctx context.Context, interval time.Duration, limitBytes int64) {
	log.Printf("Starting data lifecycle manager with interval %v and size limit %d bytes", interval, limitBytes)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Println("Running scheduled data lifecycle management...")
			if err := s.Archive(ctx); err != nil {
				log.Printf("Error during data archival: %v", err)
			}
			if err := s.EnforceRetention(limitBytes); err != nil {
				log.Printf("Error during retention enforcement: %v", err)
			}
		case <-ctx.Done():
			log.Println("Stopping data lifecycle manager.")
			return
		}
	}
}

func (s *Storage) Archive(ctx context.Context) error {
	s.archiveMu.Lock()
	defer s.archiveMu.Unlock()

	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM kube_events").Scan(&count); err != nil {
		return fmt.Errorf("failed to count rows in kube_events: %w", err)
	}

	if count == 0 {
		log.Println("No new events to archive.")
		return nil
	}

	archiveTableName := fmt.Sprintf("kube_events_archive_%d", time.Now().UnixNano())
	log.Printf("Archiving %d events to table %s", count, archiveTableName)

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback on error

	if _, err := tx.ExecContext(ctx, fmt.Sprintf("ALTER TABLE kube_events RENAME TO %s", archiveTableName)); err != nil {
		return fmt.Errorf("failed to rename table: %w", err)
	}

	if _, err := tx.ExecContext(ctx, createTableSQL); err != nil {
		return fmt.Errorf("failed to create new kube_events table: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("Successfully swapped kube_events table with %s.", archiveTableName)

	go s.processArchivedTable(archiveTableName)

	return nil
}

func (s *Storage) processArchivedTable(tableName string) {
	log.Printf("Starting background processing for table: %s", tableName)
	parquetFileName := filepath.Join(s.dataDir, fmt.Sprintf("%s.parquet", tableName))

	// Using a new connection for the background task
	conn, err := s.db.Conn(context.Background())
	if err != nil {
		log.Printf("Error getting connection for background processing: %v", err)
		return
	}
	defer conn.Close()

	copySQL := fmt.Sprintf("COPY %s TO '%s' (FORMAT 'parquet', COMPRESSION 'zstd');", tableName, parquetFileName)
	if _, err := conn.ExecContext(context.Background(), copySQL); err != nil {
		log.Printf("Error exporting table %s to parquet: %v", tableName, err)
		// Don't drop the table if copy fails, to allow for manual recovery
		return
	}
	log.Printf("Successfully exported table %s to %s", tableName, parquetFileName)

	dropSQL := fmt.Sprintf("DROP TABLE %s", tableName)
	if _, err := conn.ExecContext(context.Background(), dropSQL); err != nil {
		log.Printf("Error dropping archived table %s: %v", tableName, err)
		return
	}
	log.Printf("Successfully dropped archived table %s", tableName)
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

	// Sort by name, which is timestamp-based, so oldest first
	sort.Slice(parquetFiles, func(i, j int) bool {
		return parquetFiles[i].Name() < parquetFiles[j].Name()
	})

	var totalSize int64
	for _, file := range parquetFiles {
		info, err := file.Info()
		if err != nil {
			log.Printf("Could not get file info for %s: %v", file.Name(), err)
			continue
		}
		totalSize += info.Size()
	}

	log.Printf("Current parquet files size: %d bytes. Limit: %d bytes.", totalSize, limitBytes)

	for totalSize > limitBytes {
		if len(parquetFiles) == 0 {
			break
		}
		oldestFile := parquetFiles[0]
		parquetFiles = parquetFiles[1:]

		info, err := oldestFile.Info()
		if err != nil {
			log.Printf("Could not get file info for deletion candidate %s: %v", oldestFile.Name(), err)
			continue
		}

		filePath := filepath.Join(s.dataDir, oldestFile.Name())
		if err := os.Remove(filePath); err != nil {
			log.Printf("Failed to delete oldest parquet file %s: %v", filePath, err)
			// Stop trying to delete if one fails
			break
		}

		totalSize -= info.Size()
		log.Printf("Deleted oldest parquet file: %s. New total size: %d bytes", oldestFile.Name(), totalSize)
	}

	return nil
}

func (s *Storage) AppendEvent(k8sEvent *corev1.Event) error {
	var series map[string]any
	if k8sEvent.Series != nil {
		series = map[string]any{
			"count":            k8sEvent.Series.Count,
			"lastObservedTime": k8sEvent.Series.LastObservedTime.Time,
		}
	} else {
		series = map[string]any{"count": nil, "lastObservedTime": nil}
	}

	var related map[string]any
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
		related = map[string]any{
			"kind":            nil,
			"namespace":       nil,
			"name":            nil,
			"uid":             nil,
			"apiVersion":      nil,
			"resourceVersion": nil,
			"fieldPath":       nil,
		}
	}

	err := s.appender.AppendRow(
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
	)

	if err != nil {
		return fmt.Errorf("failed to append row: %w", err)
	}
	if err := s.appender.Flush(); err != nil {
		log.Printf("failed to flush appender: %v", err)
	}
	return nil
}

func (s *Storage) GetLastResourceVersion() (string, error) {
	var resourceVersion string
	// Order by eventTime DESC and then resourceVersion DESC to handle events with the same timestamp.
	// We are casting resourceVersion to a UINTEGER for sorting because Kubernetes resourceVersions are
	// large numbers represented as strings.
	err := s.db.QueryRow("SELECT metadata.resourceVersion FROM kube_events ORDER BY eventTime DESC, TRY_CAST(metadata.resourceVersion AS UINTEGER) DESC LIMIT 1").Scan(&resourceVersion)
	if err != nil {
		if err == sql.ErrNoRows {
			// If the table is empty, we don't have a resource version to start from.
			return "", nil
		}
		return "", fmt.Errorf("failed to query last resource version: %w", err)
	}
	return resourceVersion, nil
}
