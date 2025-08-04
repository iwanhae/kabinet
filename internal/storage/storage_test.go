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
		name              string
		relevantFiles     []string
		includeKubeEvents bool
		from              time.Time
		to                time.Time
		expectedClause    string
		expectError       bool
	}{
		{
			name:              "Only kube_events",
			relevantFiles:     []string{},
			includeKubeEvents: true,
			from:              from,
			to:                to,
			expectedClause:    fmt.Sprintf("(SELECT * FROM kube_events WHERE lastTimestamp BETWEEN '%s' AND '%s')", fromStr, toStr),
			expectError:       false,
		},
		{
			name:              "Only single parquet file",
			relevantFiles:     []string{"/data/file1.parquet"},
			includeKubeEvents: false,
			from:              from,
			to:                to,
			expectedClause:    fmt.Sprintf("(SELECT * FROM read_parquet(['/data/file1.parquet']) WHERE lastTimestamp BETWEEN '%s' AND '%s')", fromStr, toStr),
			expectError:       false,
		},
		{
			name:              "Only multiple parquet files",
			relevantFiles:     []string{"/data/file1.parquet", "/data/file2.parquet"},
			includeKubeEvents: false,
			from:              from,
			to:                to,
			expectedClause:    fmt.Sprintf("(SELECT * FROM read_parquet(['/data/file1.parquet', '/data/file2.parquet']) WHERE lastTimestamp BETWEEN '%s' AND '%s')", fromStr, toStr),
			expectError:       false,
		},
		{
			name:              "kube_events and multiple parquet files",
			relevantFiles:     []string{"/data/file1.parquet", "/data/file2.parquet"},
			includeKubeEvents: true,
			from:              from,
			to:                to,
			expectedClause:    fmt.Sprintf("(SELECT * FROM kube_events WHERE lastTimestamp BETWEEN '%s' AND '%s' UNION BY NAME SELECT * FROM read_parquet(['/data/file1.parquet', '/data/file2.parquet']) WHERE lastTimestamp BETWEEN '%s' AND '%s')", fromStr, toStr, fromStr, toStr),
			expectError:       false,
		},
		{
			name:              "No sources",
			relevantFiles:     []string{},
			includeKubeEvents: false,
			from:              from,
			to:                to,
			expectedClause:    "",
			expectError:       true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clause, err := buildFromClause(tc.relevantFiles, tc.includeKubeEvents, tc.from, tc.to)

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
