package storage

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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

	fromClause, err := buildFromClause(relevantFilePaths, includeKubeEvents, start, end)
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

func buildFromClause(relevantFiles []string, includeKubeEvents bool, from, to time.Time) (string, error) {
	var fromSources []string
	if includeKubeEvents {
		fromSources = append(fromSources, fmt.Sprintf("SELECT * FROM kube_events WHERE lastTimestamp BETWEEN '%s' AND '%s'", from.Format(time.RFC3339), to.Format(time.RFC3339)))
	}

	if len(relevantFiles) > 0 {
		quotedFiles := make([]string, len(relevantFiles))
		for i, p := range relevantFiles {
			quotedFiles[i] = fmt.Sprintf("'%s'", p)
		}
		parquetSource := fmt.Sprintf("SELECT * FROM read_parquet([%s]) WHERE lastTimestamp BETWEEN '%s' AND '%s'", strings.Join(quotedFiles, ", "), from.Format(time.RFC3339), to.Format(time.RFC3339))
		fromSources = append(fromSources, parquetSource)
	}

	if len(fromSources) == 0 {
		return "", fmt.Errorf("no data sources for query")
	}

	return fmt.Sprintf("(%s)", strings.Join(fromSources, " UNION BY NAME ")), nil
}
