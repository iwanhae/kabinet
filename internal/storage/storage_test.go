package storage

import (
	"testing"
)

func TestBuildFromClause(t *testing.T) {
	testCases := []struct {
		name              string
		relevantFiles     []string
		includeKubeEvents bool
		expectedClause    string
		expectError       bool
	}{
		{
			name:              "Only kube_events",
			relevantFiles:     []string{},
			includeKubeEvents: true,
			expectedClause:    "(SELECT * FROM kube_events)",
			expectError:       false,
		},
		{
			name:              "Only single parquet file",
			relevantFiles:     []string{"/data/file1.parquet"},
			includeKubeEvents: false,
			expectedClause:    "(SELECT * FROM read_parquet(['/data/file1.parquet']))",
			expectError:       false,
		},
		{
			name:              "Only multiple parquet files",
			relevantFiles:     []string{"/data/file1.parquet", "/data/file2.parquet"},
			includeKubeEvents: false,
			expectedClause:    "(SELECT * FROM read_parquet(['/data/file1.parquet', '/data/file2.parquet']))",
			expectError:       false,
		},
		{
			name:              "kube_events and multiple parquet files",
			relevantFiles:     []string{"/data/file1.parquet", "/data/file2.parquet"},
			includeKubeEvents: true,
			expectedClause:    "(SELECT * FROM kube_events UNION BY NAME SELECT * FROM read_parquet(['/data/file1.parquet', '/data/file2.parquet']))",
			expectError:       false,
		},
		{
			name:              "No sources",
			relevantFiles:     []string{},
			includeKubeEvents: false,
			expectedClause:    "",
			expectError:       true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clause, err := buildFromClause(tc.relevantFiles, tc.includeKubeEvents)

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
