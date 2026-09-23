package compact

import (
	"strings"
	"testing"
)

func TestMigrationHourUsesAllInputsWithHourBoundedSort(t *testing.T) {
	start, end := int64(0), int64(3_600_000_000)
	job := Job{
		LegacyInputs: []string{"old-a.parquet", "old-b.parquet"},
		Inputs:       []string{"wal-a.parquet"},
		RangeStartUs: &start,
		RangeEndUs:   &end,
	}
	sql := migrateSelectSQL(job)
	for _, path := range append(job.LegacyInputs, job.Inputs...) {
		if !strings.Contains(sql, path) {
			t.Fatalf("migration SQL omitted full-level input %q: %s", path, sql)
		}
	}
	if !strings.Contains(sql, "ORDER BY timestamp, metadata.uid, metadata.resourceVersion, filename, file_row_number") {
		t.Fatalf("migration SQL does not sort the selected hour deterministically: %s", sql)
	}
	if !strings.Contains(sql, "epoch_us(timestamp) >= 0") || !strings.Contains(sql, "epoch_us(timestamp) < 3600000000") {
		t.Fatalf("migration SQL does not use a half-open hour: %s", sql)
	}
}

func TestNormalCompactionSelectsDoNotSortFullPayload(t *testing.T) {
	for _, job := range []Job{
		{Mode: ModeConvert, Inputs: []string{"a.jsonl.zst"}},
		{Mode: ModeMerge, Inputs: []string{"a.parquet", "b.parquet"}},
		{Mode: ModeRepack, Inputs: []string{"a.parquet", "b.parquet"}},
	} {
		sql, err := selectSQLForJob(job)
		if err != nil {
			t.Fatal(err)
		}
		upper := strings.ToUpper(sql)
		if strings.Contains(upper, "SELECT * FROM (") && strings.Contains(upper, ") ORDER BY TIMESTAMP") {
			t.Fatalf("%s contains a full-payload sort: %s", job.Mode, sql)
		}
	}
}
