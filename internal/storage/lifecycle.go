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
	"time"
)

// LifecycleManager manages data lifecycle with periodic archiving and retention enforcement
func (s *Storage) LifecycleManager(ctx context.Context, archiveTableSizeMB, storageLimitBytes int64) {
	log.Printf("storage: starting data lifecycle manager. check_interval=1m, archive_table_size_mb=%d, storage_limit_bytes=%d",
		archiveTableSizeMB, storageLimitBytes)

	archiveTableSizeBytes := archiveTableSizeMB * 1024 * 1024

	// Ticker for periodic maintenance (compaction and retention)
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check table size and archive if it exceeds the threshold
			_, err := s.archiveByTableSize(ctx, archiveTableSizeBytes)
			if err != nil {
				log.Printf("storage: error during size-based archival: %v", err)
			}

			// Run maintenance tasks (compaction and retention)
			log.Println("storage: running periodic maintenance...")
			if err := s.runMaintenance(ctx, storageLimitBytes); err != nil {
				log.Printf("storage: error during maintenance: %v", err)
			}
			log.Println("storage: finished periodic maintenance.")

		case <-ctx.Done():
			log.Println("storage: stopping data lifecycle manager.")
			return
		}
	}
}

func (s *Storage) archiveByTableSize(ctx context.Context, archiveTableSizeBytes int64) (bool, error) {
	var tableName string
	var estimatedSize sql.NullInt64
	query := `SELECT table_name, estimated_size FROM duckdb_tables() WHERE table_name='kube_events'`
	err := s.db.QueryRowContext(ctx, query).Scan(&tableName, &estimatedSize)

	if err != nil {
		if err == sql.ErrNoRows {
			// table doesn't exist yet, which is fine
			return false, nil
		}
		return false, fmt.Errorf("failed to query table size: %w", err)
	}

	if estimatedSize.Valid && estimatedSize.Int64 > archiveTableSizeBytes {
		log.Printf("storage: kube_events table size (%d bytes) exceeds threshold (%d bytes). starting archival.", estimatedSize.Int64, archiveTableSizeBytes)
		if err := s.archive(ctx); err != nil {
			return false, fmt.Errorf("failed to archive table: %w", err)
		}
		return true, nil
	} else {
		log.Printf("storage: kube_events table size (%d bytes) is below threshold (%d bytes). skipping archival.", estimatedSize.Int64, archiveTableSizeBytes)
	}
	return false, nil
}

func (s *Storage) runMaintenance(ctx context.Context, storageLimitBytes int64) error {
	if err := s.EnforceRetention(storageLimitBytes); err != nil {
		return fmt.Errorf("retention enforcement failed: %w", err)
	}
	if err := s.CompactParquetFiles(ctx, 128*1024*1024); err != nil {
		return fmt.Errorf("parquet compaction failed: %w", err)
	}
	return nil
}

// archive archives the current kube_events table to parquet files
func (s *Storage) archive(ctx context.Context) error {
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

func (s *Storage) CompactParquetFiles(ctx context.Context, compactThresholdBytes int64) error {
	log.Println("storage: starting parquet file compaction process...")
	defer log.Println("storage: finished parquet file compaction process.")

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

	if len(parquetFiles) < 2 {
		log.Println("storage: not enough parquet files to consider compaction.")
		return nil
	}

	// Sort by timestamp in filename, oldest first.
	sort.Slice(parquetFiles, func(i, j int) bool {
		tsI := extractTimestampFromName(parquetFiles[i].Name())
		tsJ := extractTimestampFromName(parquetFiles[j].Name())
		if tsI == tsJ {
			return parquetFiles[i].Name() < parquetFiles[j].Name()
		}
		return tsI < tsJ
	})

	var batchToMerge []os.DirEntry
	var currentBatchSize int64

	for _, file := range parquetFiles {
		info, err := file.Info()
		if err != nil {
			log.Printf("storage: could not get file info for %s, skipping: %v", file.Name(), err)
			continue
		}

		if info.Size() < compactThresholdBytes {
			batchToMerge = append(batchToMerge, file)
			currentBatchSize += info.Size()
		} else {
			// Current file is large, so we process any batch we've collected so far
			if len(batchToMerge) > 1 && currentBatchSize > compactThresholdBytes {
				if err := s.mergeFileBatch(ctx, batchToMerge); err != nil {
					log.Printf("storage: failed to merge parquet batch: %v. will retry on next cycle.", err)
				}
			}
			// Reset batch after processing or if it wasn't worth processing
			batchToMerge = nil
			currentBatchSize = 0
		}
	}

	// Process the last batch if any exists
	if len(batchToMerge) > 1 {
		if err := s.mergeFileBatch(ctx, batchToMerge); err != nil {
			log.Printf("storage: failed to merge final parquet batch: %v. will retry on next cycle.", err)
		}
	}

	return nil
}

func (s *Storage) mergeFileBatch(ctx context.Context, batch []os.DirEntry) error {
	if len(batch) < 2 {
		return fmt.Errorf("at least two files are required for a merge, got %d", len(batch))
	}

	// Prepare file paths for SQL query and deletion
	filesToMergePaths := make([]string, len(batch))
	for i, file := range batch {
		filesToMergePaths[i] = filepath.Join(s.dataDir, file.Name())
	}

	// Create a new filename based on the time range of the batch
	firstFileMinTs, _, ok1 := parseParquetFilename(batch[0].Name())
	_, lastFileMaxTs, ok2 := parseParquetFilename(batch[len(batch)-1].Name())
	if !ok1 || !ok2 {
		return fmt.Errorf("could not parse timestamps from batch filenames")
	}

	newFileName := fmt.Sprintf("events_%d_%d.parquet", firstFileMinTs, lastFileMaxTs)
	newFilePath := filepath.Join(s.dataDir, newFileName)
	log.Printf("storage: merging %d files into %s", len(batch), newFileName)

	// Build and execute the merge query
	quotedFiles := make([]string, len(filesToMergePaths))
	for i, p := range filesToMergePaths {
		quotedFiles[i] = fmt.Sprintf("'%s'", p)
	}
	// Important: order by lastTimestamp to keep data sorted in the new parquet file.
	copySQL := fmt.Sprintf(`COPY (SELECT * FROM read_parquet([%s]) ORDER BY lastTimestamp) TO '%s' (FORMAT 'parquet', COMPRESSION 'zstd');`,
		strings.Join(quotedFiles, ", "), newFilePath)

	if _, err := s.db.ExecContext(ctx, copySQL); err != nil {
		// Cleanup partially written file if copy fails
		os.Remove(newFilePath)
		return fmt.Errorf("failed to execute merge copy: %w", err)
	}

	// Merge successful, now delete original files
	log.Printf("storage: successfully created merged file %s. Deleting original files...", newFileName)
	for _, path := range filesToMergePaths {
		if err := os.Remove(path); err != nil {
			// This is not ideal, as we now have duplicated data.
			// Log this clearly for manual intervention.
			log.Printf("storage: CRITICAL: failed to delete source file %s after merging. Manual intervention required.", path)
		}
	}

	log.Printf("storage: finished merging batch into %s", newFileName)
	return nil
}

// EnforceRetention enforces data retention by deleting oldest parquet files when size limit is exceeded
func (s *Storage) EnforceRetention(limitBytes int64) error {
	log.Println("storage: enforcing retention policy...")
	defer log.Println("storage: finished enforcing retention policy.")

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
