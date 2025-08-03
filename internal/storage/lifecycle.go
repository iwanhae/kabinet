package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// LifecycleManager manages data lifecycle with periodic archiving and retention enforcement
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

// Archive archives the current kube_events table to parquet files
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

// processArchivedTable processes an archived table by exporting it to parquet and then dropping it
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

// EnforceRetention enforces data retention by deleting oldest parquet files when size limit is exceeded
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

// dataDirSize calculates the total size of all files in the data directory
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
