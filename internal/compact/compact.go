// Package compact implements the heavy data-management work: converting raw
// WAL segments to canonical Parquet and merging Parquet files, with
// deduplication applied on every pass. It is executed inside the compactor
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
)

// Job is the unit of work passed to the compactor subprocess as JSON on stdin.
type Job struct {
	Mode          string   `json:"mode"`
	Inputs        []string `json:"inputs"`
	Output        string   `json:"output"`
	TempDir       string   `json:"tempDir"`
	MemoryLimitMB int      `json:"memoryLimitMB"`
}

// Result is written to stdout as JSON when a job succeeds.
type Result struct {
	Rows  int64 `json:"rows"`
	MinMs int64 `json:"minMs"`
	MaxMs int64 `json:"maxMs"`
}

// Run executes a job with a memory-bounded DuckDB instance. Excess memory
// spills to TempDir instead of growing the process.
func Run(ctx context.Context, job Job) (*Result, error) {
	if len(job.Inputs) == 0 {
		return nil, fmt.Errorf("job has no inputs")
	}
	if job.Output == "" {
		return nil, fmt.Errorf("job has no output")
	}

	var source string
	switch job.Mode {
	case ModeConvert:
		source = schema.JSONLSource(job.Inputs)
	case ModeMerge:
		source = schema.ParquetSource(job.Inputs)
	default:
		return nil, fmt.Errorf("unknown job mode: %q", job.Mode)
	}

	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("failed to open duckdb: %w", err)
	}
	defer db.Close()

	pragmas := fmt.Sprintf(
		"SET memory_limit='%dMB'; SET temp_directory=%s; SET preserve_insertion_order=false;",
		job.MemoryLimitMB, schema.QuotePath(job.TempDir),
	)
	if _, err := db.ExecContext(ctx, pragmas); err != nil {
		return nil, fmt.Errorf("failed to configure duckdb: %w", err)
	}

	copySQL := fmt.Sprintf(
		"COPY (SELECT * FROM %s %s ORDER BY lastTimestamp) TO %s (FORMAT parquet, COMPRESSION zstd, ROW_GROUP_SIZE 122880)",
		source, schema.DedupQualify, schema.QuotePath(job.Output),
	)
	if _, err := db.ExecContext(ctx, copySQL); err != nil {
		return nil, fmt.Errorf("failed to write %s: %w", job.Output, err)
	}

	statsSQL := fmt.Sprintf(
		"SELECT count(*), epoch_ms(min(lastTimestamp)), epoch_ms(max(lastTimestamp)) FROM read_parquet(%s)",
		schema.QuotePath(job.Output),
	)
	var rows int64
	var minMs, maxMs sql.NullInt64
	if err := db.QueryRowContext(ctx, statsSQL).Scan(&rows, &minMs, &maxMs); err != nil {
		return nil, fmt.Errorf("failed to read output stats: %w", err)
	}

	return &Result{Rows: rows, MinMs: minMs.Int64, MaxMs: maxMs.Int64}, nil
}
