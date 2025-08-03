package storage

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
)

// serializeRows converts database rows to a slice of maps
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
