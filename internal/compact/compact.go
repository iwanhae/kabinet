// Package compact implements the heavy data-management work: converting raw
// WAL segments to canonical Parquet, merging Parquet files with deduplication,
// and streaming row-preserving repacks. It is executed inside the compactor
// subprocess (cmd/compactor) so an OOM kills the compactor, not the server.
package compact

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/iwanhae/kabinet/internal/schema"
	_ "github.com/marcboeker/go-duckdb/v2"
)

// Job modes.
const (
	ModeConvert = "convert" // raw JSONL segments -> canonical Parquet
	ModeMerge   = "merge"   // canonical Parquet files -> one canonical Parquet
	ModeRepack  = "repack"  // canonical Parquet files -> one canonical Parquet, preserving every row
	ModeInspect = "inspect" // aggregate legacy/canonical Parquet inputs for migration planning
	ModeMigrate = "migrate" // rewrite one half-open timestamp hour into canonical Parquet
)

// Job is the unit of work passed to the compactor subprocess as JSON on stdin.
type Job struct {
	Mode          string   `json:"mode"`
	Inputs        []string `json:"inputs"`
	Output        string   `json:"output"`
	TempDir       string   `json:"tempDir"`
	MemoryLimitMB int      `json:"memoryLimitMB"`
	// LegacyInputs are pre-v2 Parquet files without the timestamp column.
	LegacyInputs []string `json:"legacyInputs,omitempty"`
	// RangeStartUs/RangeEndUs define a half-open timestamp range for migrate.
	RangeStartUs *int64 `json:"rangeStartUs,omitempty"`
	RangeEndUs   *int64 `json:"rangeEndUs,omitempty"`
}

// Result is written to stdout as JSON when a job succeeds.
type Result struct {
	Rows     int64  `json:"rows"`
	MinMs    int64  `json:"minMs"`
	MaxMs    int64  `json:"maxMs"`
	MinUs    int64  `json:"minUs"`
	MaxUs    int64  `json:"maxUs"`
	Checksum uint64 `json:"checksum"`
}

// Run executes a job with a memory-bounded DuckDB instance. Excess memory
// spills to TempDir instead of growing the process.
func Run(ctx context.Context, job Job) (*Result, error) {
	if len(job.Inputs) == 0 && len(job.LegacyInputs) == 0 {
		return nil, fmt.Errorf("job has no inputs")
	}
	if job.Output == "" && job.Mode != ModeInspect {
		return nil, fmt.Errorf("job has no output")
	}

	selectSQL, err := selectSQLForJob(job)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("failed to open duckdb: %w", err)
	}
	defer db.Close()

	pragmas := fmt.Sprintf(
		"SET memory_limit='%dMB'; SET temp_directory=%s; SET threads=1; SET preserve_insertion_order=false;",
		job.MemoryLimitMB, schema.QuotePath(job.TempDir),
	)
	if _, err := db.ExecContext(ctx, pragmas); err != nil {
		return nil, fmt.Errorf("failed to configure duckdb: %w", err)
	}
	if job.Mode == ModeInspect {
		return inspect(ctx, db, migrationSource(job))
	}

	copySQL := fmt.Sprintf(
		"COPY (%s) TO %s (FORMAT parquet, COMPRESSION zstd, ROW_GROUP_SIZE_BYTES '16MB')",
		selectSQL, schema.QuotePath(job.Output),
	)
	if _, err := db.ExecContext(ctx, copySQL); err != nil {
		return nil, fmt.Errorf("failed to write %s: %w", job.Output, err)
	}

	statsSQL := fmt.Sprintf(
		"SELECT count(*), count(timestamp), floor(epoch_us(min(timestamp)) / 1000.0)::BIGINT, ceil(epoch_us(max(timestamp)) / 1000.0)::BIGINT, COALESCE(bit_xor(%s), 0)::UBIGINT FROM read_parquet(%s)",
		schema.RowHashExpr(),
		schema.QuotePath(job.Output),
	)
	var rows int64
	var timestampRows int64
	var minMs, maxMs sql.NullInt64
	var checksum uint64
	if err := db.QueryRowContext(ctx, statsSQL).Scan(&rows, &timestampRows, &minMs, &maxMs, &checksum); err != nil {
		return nil, fmt.Errorf("failed to read output stats: %w", err)
	}
	if timestampRows != rows {
		return nil, fmt.Errorf("output has %d rows without canonical timestamp", rows-timestampRows)
	}

	return &Result{Rows: rows, MinMs: minMs.Int64, MaxMs: maxMs.Int64, Checksum: checksum}, nil
}

func selectSQLForJob(job Job) (string, error) {
	switch job.Mode {
	case ModeConvert:
		return fmt.Sprintf("SELECT * FROM %s %s", schema.JSONLSource(job.Inputs), schema.DedupQualify), nil
	case ModeMerge:
		return mergeSelectSQL(job.Inputs), nil
	case ModeRepack:
		return fmt.Sprintf("SELECT * FROM %s", schema.ParquetSource(job.Inputs)), nil
	case ModeInspect:
		return "", nil
	case ModeMigrate:
		return migrateSelectSQL(job), nil
	default:
		return "", fmt.Errorf("unknown job mode: %q", job.Mode)
	}
}

// mergeSelectSQL keeps the blocking dedup window narrow: only the event key,
// timestamp, and physical row locator participate in it. The second scan
// streams full event rows through a semi join into the Parquet writer instead
// of sorting and spilling the complete payload.
func mergeSelectSQL(paths []string) string {
	source := schema.ParquetSourceWithRowLocation(paths)
	return fmt.Sprintf(`
		WITH winners AS (
			SELECT filename, file_row_number
			FROM %s
			QUALIFY row_number() OVER (
				PARTITION BY metadata.uid, metadata.resourceVersion
				ORDER BY timestamp DESC, filename DESC, file_row_number DESC
			) = 1
		)
		SELECT p.* EXCLUDE (filename, file_row_number)
		FROM %s AS p
		SEMI JOIN winners USING (filename, file_row_number)`, source, source)
}
