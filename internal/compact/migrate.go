package compact

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/iwanhae/kabinet/internal/schema"
)

// migrationSource combines pre-v2 Parquet inputs, whose timestamp must be
// projected, with canonical inputs produced while migrating WAL segments.
func migrationSource(job Job) string {
	var sources []string
	if len(job.LegacyInputs) > 0 {
		sources = append(sources, "SELECT * FROM "+schema.LegacyParquetSourceWithRowLocation(job.LegacyInputs))
	}
	if len(job.Inputs) > 0 {
		sources = append(sources, "SELECT * FROM "+schema.ParquetSourceWithRowLocation(job.Inputs))
	}
	return "(" + strings.Join(sources, " UNION ALL BY NAME ") + ")"
}

// migrateSelectSQL rewrites one half-open time partition into canonical rows.
// Every migration job receives the complete level input set; the timestamp
// predicate, rather than legacy filenames, decides membership. Sorting is
// bounded to the selected hour, so the final size-based repack can remain a
// streaming operation without sorting the complete level payload.
func migrateSelectSQL(job Job) string {
	var where []string
	if job.RangeStartUs != nil {
		where = append(where, fmt.Sprintf("epoch_us(timestamp) >= %d", *job.RangeStartUs))
	}
	if job.RangeEndUs != nil {
		where = append(where, fmt.Sprintf("epoch_us(timestamp) < %d", *job.RangeEndUs))
	}
	filter := ""
	if len(where) > 0 {
		filter = " WHERE " + strings.Join(where, " AND ")
	}
	return fmt.Sprintf(
		"SELECT %s FROM %s%s ORDER BY timestamp, metadata.uid, metadata.resourceVersion, filename, file_row_number",
		schema.ColumnList(), migrationSource(job), filter,
	)
}

// inspect performs the single aggregate scan used to plan a level migration.
// Exact microsecond bounds are returned so UTC hour boundaries do not depend
// on rounded archive filenames.
func inspect(ctx context.Context, db *sql.DB, source string) (*Result, error) {
	query := fmt.Sprintf(`
		SELECT count(*), count(timestamp), epoch_us(min(timestamp)), epoch_us(max(timestamp)),
		       COALESCE(bit_xor(%s), 0)::UBIGINT
		FROM %s`, schema.RowHashExpr(), source)
	var result Result
	var timestampRows int64
	var minUs, maxUs sql.NullInt64
	if err := db.QueryRowContext(ctx, query).Scan(&result.Rows, &timestampRows, &minUs, &maxUs, &result.Checksum); err != nil {
		return nil, fmt.Errorf("failed to inspect migration input: %w", err)
	}
	if timestampRows != result.Rows {
		return nil, fmt.Errorf("migration input has %d rows without timestamp", result.Rows-timestampRows)
	}
	if minUs.Valid {
		result.MinUs, result.MaxUs = minUs.Int64, maxUs.Int64
		result.MinMs = result.MinUs / 1000
		result.MaxMs = (result.MaxUs + 999) / 1000
	}
	return &result, nil
}
