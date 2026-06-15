package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// Max age of JSONL file before archival (10 minutes)
	maxJSONLFileAgeForArchive = 10 * time.Minute
)

// LifecycleManager manages data lifecycle with periodic archiving and retention enforcement
func (s *Storage) LifecycleManager(ctx context.Context, storageLimitBytes int64) {
	log.Printf("storage: starting data lifecycle manager. check_interval=1m, archive_max_file_age=%s, storage_limit_bytes=%d",
		maxJSONLFileAgeForArchive, storageLimitBytes)

	// Ticker for periodic maintenance (compaction and retention)
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check for old JSONL files and archive them
			archived, err := s.archiveByFileAge(ctx, maxJSONLFileAgeForArchive)
			if err != nil {
				log.Printf("storage: error during file-age-based archival: %v", err)
			}
			if archived {
				log.Println("storage: archival completed, running maintenance...")
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

func (s *Storage) archiveByFileAge(ctx context.Context, maxAge time.Duration) (bool, error) {
	jsonlFiles, err := s.findJSONLFilesOlderThan(maxAge)
	if err != nil {
		return false, fmt.Errorf("failed to find old JSONL files: %w", err)
	}

	if len(jsonlFiles) == 0 {
		return false, nil
	}

	for _, jsonlFile := range jsonlFiles {
		go s.processJSONLToParquet(ctx, jsonlFile)
	}
	return true, nil
}

// findJSONLFilesOlderThan finds JSONL files older than the specified age
func (s *Storage) findJSONLFilesOlderThan(maxAge time.Duration) ([]string, error) {
	files, err := os.ReadDir(s.dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read data directory: %w", err)
	}

	var oldFiles []string

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".jsonl") || !strings.HasPrefix(file.Name(), "events_") {
			continue
		}

		// Skip the current active JSONL file
		if s.jsonlWriter.GetCurrentPath() == filepath.Join(s.dataDir, file.Name()) {
			continue
		}

		info, err := file.Info()
		if err != nil {
			log.Printf("storage: could not get file info for %s: %v", file.Name(), err)
			continue
		}

		// Check file age based on modification time
		if time.Since(info.ModTime()) >= maxAge {
			oldFiles = append(oldFiles, filepath.Join(s.dataDir, file.Name()))
		}
	}

	return oldFiles, nil
}

// processJSONLToParquet converts a JSONL file to Parquet and deletes the original
func (s *Storage) processJSONLToParquet(ctx context.Context, jsonlPath string) {
	log.Printf("storage: starting archival for %s", jsonlPath)

	// Get time range from JSONL for filename
	minTime, maxTime, err := s.getJSONLTimeRange(ctx, jsonlPath)
	if err != nil {
		log.Printf("storage: error getting time range for %s: %v", jsonlPath, err)
		// Fallback to file modification time
		info, err := os.Stat(jsonlPath)
		if err != nil {
			log.Printf("storage: error getting file info for %s: %v", jsonlPath, err)
			return
		}
		minTime = info.ModTime()
		maxTime = info.ModTime()
	}

	// Parquet filename: events_<min_ts>_<max_ts>.parquet
	parquetPath := filepath.Join(s.dataDir, fmt.Sprintf("events_%d_%d.parquet", minTime.Unix(), maxTime.Unix()))

	// Use DuckDB to convert JSONL to Parquet
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		log.Printf("storage: error starting transaction for archival: %v", err)
		return
	}
	defer tx.Rollback()

	copySQL := fmt.Sprintf(`
		COPY (SELECT * FROM read_json_auto('%s'))
		TO '%s' (FORMAT 'parquet', COMPRESSION 'zstd');
	`, jsonlPath, parquetPath)

	if _, err := tx.ExecContext(ctx, copySQL); err != nil {
		log.Printf("storage: error converting JSONL to Parquet: %v", err)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("storage: error committing archival transaction: %v", err)
		return
	}

	log.Printf("storage: successfully created %s from %s", parquetPath, jsonlPath)

	// Delete the original JSONL file
	if err := os.Remove(jsonlPath); err != nil {
		log.Printf("storage: error deleting JSONL file %s: %v", jsonlPath, err)
		return
	}

	log.Printf("storage: successfully archived and deleted %s", jsonlPath)
}

// getJSONLTimeRange gets the min and max lastTimestamp from a JSONL file
func (s *Storage) getJSONLTimeRange(ctx context.Context, jsonlPath string) (time.Time, time.Time, error) {
	query := fmt.Sprintf(`
		SELECT MIN(lastTimestamp) as min_ts, MAX(lastTimestamp) as max_ts
		FROM read_json_auto('%s')
	`, jsonlPath)

	var minTime, maxTime time.Time
	err := s.db.QueryRowContext(ctx, query).Scan(&minTime, &maxTime)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("failed to get time range: %w", err)
	}

	return minTime, maxTime, nil
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

func (s *Storage) CompactParquetFiles(ctx context.Context, compactThresholdBytes int64) error {
	log.Println("storage: starting parquet file compaction process...")
	defer log.Println("storage: finished parquet file compaction process.")

	files, err := os.ReadDir(s.dataDir)
	if err != nil {
		return fmt.Errorf("failed to read data directory: %w", err)
	}

	var parquetFiles []os.DirEntry
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".parquet") && strings.HasPrefix(file.Name(), "events_") {
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

	// Prepare file paths for merger command
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
	log.Printf("storage: merging %d files into %s using external merger process", len(batch), newFileName)

	// Find merger binary (from PATH or /usr/local/bin)
	mergerPath, err := exec.LookPath("merger")
	if err != nil {
		// Fallback to common installation path in Docker
		mergerPath = "/usr/local/bin/merger"
		if _, err := os.Stat(mergerPath); err != nil {
			return fmt.Errorf("merger binary not found in PATH or /usr/local/bin: %w", err)
		}
	}

	// Build command arguments: merger -o output.parquet input1.parquet input2.parquet ...
	args := []string{"-o", newFilePath}
	args = append(args, filesToMergePaths...)

	// Execute merger as a separate process
	cmd := exec.CommandContext(ctx, mergerPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// Cleanup partially written file if merge fails
		os.Remove(newFilePath)
		return fmt.Errorf("failed to execute merger command: %w", err)
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

// EnforceRetention enforces data retention by deleting oldest files (JSONL or Parquet) when size limit is exceeded
func (s *Storage) EnforceRetention(limitBytes int64) error {
	log.Println("storage: enforcing retention policy...")
	defer log.Println("storage: finished enforcing retention policy.")

	files, err := os.ReadDir(s.dataDir)
	if err != nil {
		return fmt.Errorf("failed to read data directory: %w", err)
	}

	// Collect both JSONL and Parquet files
	type dataFile struct {
		entry os.DirEntry
		ts    int64
	}
	var dataFiles []dataFile

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Handle JSONL files
		if strings.HasSuffix(file.Name(), ".jsonl") && strings.HasPrefix(file.Name(), "events_") {
			// Skip the current active JSONL file
			if s.jsonlWriter.GetCurrentPath() == filepath.Join(s.dataDir, file.Name()) {
				continue
			}

			// Extract timestamp from filename: events_<timestamp>.jsonl
			base := strings.TrimSuffix(file.Name(), ".jsonl")
			parts := strings.Split(base, "_")
			if len(parts) == 2 {
				if ts, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					dataFiles = append(dataFiles, dataFile{entry: file, ts: ts})
					continue
				}
			}

			// Fallback: use modification time
			if info, err := file.Info(); err == nil {
				dataFiles = append(dataFiles, dataFile{entry: file, ts: info.ModTime().Unix()})
			}
			continue
		}

		// Handle Parquet files
		if strings.HasSuffix(file.Name(), ".parquet") && strings.HasPrefix(file.Name(), "events_") {
			ts := extractTimestampFromName(file.Name())
			dataFiles = append(dataFiles, dataFile{entry: file, ts: ts})
		}
	}

	// Sort by timestamp, oldest first
	sort.Slice(dataFiles, func(i, j int) bool {
		tsI := dataFiles[i].ts
		tsJ := dataFiles[j].ts
		if tsI == tsJ {
			return dataFiles[i].entry.Name() < dataFiles[j].entry.Name()
		}
		return tsI < tsJ
	})

	totalSize, err := s.dataDirSize()
	if err != nil {
		return fmt.Errorf("failed to get data directory size: %w", err)
	}

	log.Printf("storage: current data directory size: %d bytes. Limit: %d bytes.", totalSize, limitBytes)

	for totalSize > limitBytes {
		if len(dataFiles) == 0 {
			break
		}
		oldest := dataFiles[0]
		dataFiles = dataFiles[1:]

		info, err := oldest.entry.Info()
		if err != nil {
			log.Printf("storage: could not get file info for deletion candidate %s: %v", oldest.entry.Name(), err)
			continue
		}

		filePath := filepath.Join(s.dataDir, oldest.entry.Name())
		if err := os.Remove(filePath); err != nil {
			log.Printf("storage: failed to delete oldest file %s: %v", filePath, err)
			// Stop trying to delete if one fails
			break
		}

		totalSize -= info.Size()
		log.Printf("storage: deleted oldest file: %s. New total size: %d bytes", oldest.entry.Name(), totalSize)
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
