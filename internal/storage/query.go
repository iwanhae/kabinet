package storage

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

// RangeQuery executes a range query against the storage.
// It executes the query by substituting the $events placeholder with the appropriate FROM clause.
func (s *Storage) RangeQuery(ctx context.Context, query string, start, end time.Time) ([]map[string]any, *RangeQueryResult, error) {
	if ctx.Err() != nil {
		return nil, nil, fmt.Errorf("failed fast: %w", ctx.Err())
	}

	finalQuery, files, err := s.buildEventsQuery(query, start, end)
	if err != nil {
		return nil, nil, err
	}

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
		Files:    files,
	}, nil
}

func buildFromClause(relevantFiles []string, includeKubeEvents bool, from, to time.Time) (string, error) {
	var fromSources []string
	if includeKubeEvents {
		fromSources = append(fromSources, fmt.Sprintf("SELECT * FROM kube_events WHERE lastTimestamp BETWEEN TIMESTAMPTZ '%s' AND TIMESTAMPTZ '%s'", from.Format(time.RFC3339), to.Format(time.RFC3339)))
	}

	if len(relevantFiles) > 0 {
		quotedFiles := make([]string, len(relevantFiles))
		for i, p := range relevantFiles {
			quotedFiles[i] = fmt.Sprintf("'%s'", p)
		}
		parquetSource := fmt.Sprintf("SELECT * FROM read_parquet([%s]) WHERE lastTimestamp BETWEEN TIMESTAMPTZ '%s' AND TIMESTAMPTZ '%s'", strings.Join(quotedFiles, ", "), from.Format(time.RFC3339), to.Format(time.RFC3339))
		fromSources = append(fromSources, parquetSource)
	}

	if len(fromSources) == 0 {
		return "", fmt.Errorf("no data sources for query")
	}

	return fmt.Sprintf("(%s)", strings.Join(fromSources, " UNION ALL BY NAME ")), nil
}
