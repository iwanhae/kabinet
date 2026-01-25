package storage

import (
	"fmt"
	"testing"
	"time"
)

func TestBuildFromClause(t *testing.T) {
	from := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	fromStr := from.Format(time.RFC3339)
	toStr := to.Format(time.RFC3339)

	testCases := []struct {
		name           string
		jsonlFiles     []string
		parquetFiles   []string
		from           time.Time
		to             time.Time
		expectedClause string
		expectError    bool
	}{
		{
			name:           "Only JSONL files",
			jsonlFiles:     []string{"/data/events_123456.jsonl"},
			parquetFiles:   []string{},
			from:           from,
			to:             to,
			expectedClause: fmt.Sprintf("(SELECT * FROM read_json_auto(['/data/events_123456.jsonl']) WHERE lastTimestamp BETWEEN TIMESTAMPTZ '%s' AND TIMESTAMPTZ '%s')", fromStr, toStr),
			expectError:    false,
		},
		{
			name:           "Only single parquet file",
			jsonlFiles:     []string{},
			parquetFiles:   []string{"/data/file1.parquet"},
			from:           from,
			to:             to,
			expectedClause: fmt.Sprintf("(SELECT * FROM read_parquet(['/data/file1.parquet']) WHERE lastTimestamp BETWEEN TIMESTAMPTZ '%s' AND TIMESTAMPTZ '%s')", fromStr, toStr),
			expectError:    false,
		},
		{
			name:           "Only multiple parquet files",
			jsonlFiles:     []string{},
			parquetFiles:   []string{"/data/file1.parquet", "/data/file2.parquet"},
			from:           from,
			to:             to,
			expectedClause: fmt.Sprintf("(SELECT * FROM read_parquet(['/data/file1.parquet', '/data/file2.parquet']) WHERE lastTimestamp BETWEEN TIMESTAMPTZ '%s' AND TIMESTAMPTZ '%s')", fromStr, toStr),
			expectError:    false,
		},
		{
			name:           "JSONL and multiple parquet files",
			jsonlFiles:     []string{"/data/events_123456.jsonl", "/data/events_123457.jsonl"},
			parquetFiles:   []string{"/data/file1.parquet", "/data/file2.parquet"},
			from:           from,
			to:             to,
			expectedClause: fmt.Sprintf("(SELECT * FROM read_json_auto(['/data/events_123456.jsonl', '/data/events_123457.jsonl']) WHERE lastTimestamp BETWEEN TIMESTAMPTZ '%s' AND TIMESTAMPTZ '%s' UNION ALL BY NAME SELECT * FROM read_parquet(['/data/file1.parquet', '/data/file2.parquet']) WHERE lastTimestamp BETWEEN TIMESTAMPTZ '%s' AND TIMESTAMPTZ '%s')", fromStr, toStr, fromStr, toStr),
			expectError:    false,
		},
		{
			name:           "No sources",
			jsonlFiles:     []string{},
			parquetFiles:   []string{},
			from:           from,
			to:             to,
			expectedClause: "",
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clause, err := buildFromClause(tc.jsonlFiles, tc.parquetFiles, tc.from, tc.to)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected an error, but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Did not expect an error, but got: %v", err)
				}
				if clause != tc.expectedClause {
					t.Errorf("Expected clause:\n%s\nGot:\n%s", tc.expectedClause, clause)
				}
			}
		})
	}
}
