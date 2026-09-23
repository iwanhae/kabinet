// Package schema is the single source of truth for the canonical event
// schema. The WAL stores raw Kubernetes Event JSON; everything that reads it
// (the compactor converting to Parquet, the query planner reading recent
// segments) projects rows into this canonical shape using the SQL fragments
// defined here.
package schema

import (
	"fmt"
	"strings"
	"time"
)

const (
	objectRefType = `STRUCT(kind VARCHAR, namespace VARCHAR, "name" VARCHAR, uid VARCHAR, apiVersion VARCHAR, resourceVersion VARCHAR, fieldPath VARCHAR)`
	metadataType  = `STRUCT("name" VARCHAR, "namespace" VARCHAR, uid VARCHAR, resourceVersion VARCHAR, creationTimestamp TIMESTAMPTZ)`
	sourceType    = `STRUCT(component VARCHAR, host VARCHAR)`
	seriesType    = `STRUCT("count" INTEGER, lastObservedTime TIMESTAMPTZ)`
	// TimestampSQLExpr is the only event-time fallback used by Kabinet. Raw
	// and legacy inputs apply it once to create the canonical timestamp column;
	// every downstream operation uses that column directly.
	TimestampSQLExpr = `COALESCE(series.lastObservedTime, lastTimestamp, firstTimestamp, metadata.creationTimestamp)`
)

// ResolveTimestamp is the Go equivalent of TimestampSQLExpr. Nil and zero
// values are treated as absent. False means the event violates the canonical
// timestamp invariant and must not enter the WAL.
func ResolveTimestamp(series, last, first, creation *time.Time) (time.Time, bool) {
	for _, candidate := range []*time.Time{series, last, first, creation} {
		if candidate != nil && !candidate.IsZero() {
			return *candidate, true
		}
	}
	return time.Time{}, false
}

// Column is one canonical output column.
type Column struct {
	Name string
	Type string
	// rawExpr projects the column out of a raw K8s Event JSON row.
	// Empty means the raw column is taken as-is.
	rawExpr string
}

// Columns lists every canonical column in output order. Source timestamp
// fields remain untouched; only timestamp receives event-time fallback.
var Columns = []Column{
	{Name: "kind", Type: "VARCHAR", rawExpr: `COALESCE(kind, 'Event')`},
	{Name: "apiVersion", Type: "VARCHAR", rawExpr: `COALESCE(apiVersion, 'v1')`},
	{Name: "metadata", Type: metadataType},
	{Name: "involvedObject", Type: objectRefType},
	{Name: "reason", Type: "VARCHAR"},
	{Name: "message", Type: "VARCHAR"},
	{Name: "source", Type: sourceType},
	{Name: "timestamp", Type: "TIMESTAMPTZ", rawExpr: TimestampSQLExpr},
	{Name: "firstTimestamp", Type: "TIMESTAMPTZ"},
	{Name: "lastTimestamp", Type: "TIMESTAMPTZ"},
	{Name: "count", Type: "INTEGER", rawExpr: `CASE WHEN "count" IS NULL OR "count" = 0 THEN 1 ELSE "count" END`},
	{Name: "type", Type: "VARCHAR"},
	{Name: "eventTime", Type: "TIMESTAMPTZ"},
	{Name: "series", Type: seriesType},
	{Name: "action", Type: "VARCHAR"},
	{Name: "related", Type: objectRefType},
	{Name: "reportingComponent", Type: "VARCHAR"},
	{Name: "reportingInstance", Type: "VARCHAR"},
}

// DedupQualify keeps exactly one row per event revision. An event revision is
// identified by (metadata.uid, metadata.resourceVersion); duplicates appear
// when the informer relists after a restart.
const DedupQualify = `QUALIFY row_number() OVER (PARTITION BY metadata.uid, metadata.resourceVersion ORDER BY timestamp DESC) = 1`

// QuotePath returns p as a single-quoted SQL string literal.
func QuotePath(p string) string {
	return "'" + strings.ReplaceAll(p, "'", "''") + "'"
}

func quoteIdent(name string) string {
	return `"` + name + `"`
}

func pathList(paths []string) string {
	quoted := make([]string, len(paths))
	for i, p := range paths {
		quoted[i] = QuotePath(p)
	}
	return strings.Join(quoted, ", ")
}

// ndjsonColumnsArg builds the columns={...} argument for read_json so raw K8s
// Event JSON is read with a fixed schema instead of type inference. Fields not
// listed here (labels, annotations, managedFields, ...) are ignored by the
// struct transform.
func ndjsonColumnsArg() string {
	parts := make([]string, len(Columns))
	for i, c := range Columns {
		parts[i] = fmt.Sprintf("%s: '%s'", quoteIdent(c.Name), c.Type)
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// JSONLSource returns a parenthesized relation reading raw K8s Event JSONL
// (zstd-compressed) files projected into the canonical schema.
func JSONLSource(paths []string) string {
	selects := make([]string, len(Columns))
	for i, c := range Columns {
		expr := c.rawExpr
		if expr == "" {
			expr = quoteIdent(c.Name)
		}
		selects[i] = fmt.Sprintf("%s AS %s", expr, quoteIdent(c.Name))
	}
	return fmt.Sprintf(
		"(SELECT %s FROM read_json([%s], format='newline_delimited', compression='zstd', columns=%s))",
		strings.Join(selects, ", "), pathList(paths), ndjsonColumnsArg(),
	)
}

// ParquetSource returns a parenthesized relation reading canonical Parquet files.
func ParquetSource(paths []string) string {
	return fmt.Sprintf("(SELECT * FROM read_parquet([%s]))", pathList(paths))
}

// ParquetSourceWithRowLocation reads canonical Parquet files with DuckDB's
// virtual row-location columns. The (filename, file_row_number) pair uniquely
// identifies a physical input row and lets compaction select winners using a
// narrow projection before streaming the full rows in a second pass.
func ParquetSourceWithRowLocation(paths []string) string {
	return fmt.Sprintf(
		"(SELECT * FROM read_parquet([%s], filename=true, file_row_number=true))",
		pathList(paths),
	)
}

// LegacyParquetSourceWithRowLocation reads pre-v2 Parquet files and projects
// the synthetic timestamp column while retaining physical row locators.
func LegacyParquetSourceWithRowLocation(paths []string) string {
	selects := make([]string, 0, len(Columns)+2)
	for _, c := range Columns {
		expr := quoteIdent(c.Name)
		if c.Name == "timestamp" {
			expr = TimestampSQLExpr
		}
		selects = append(selects, fmt.Sprintf("%s AS %s", expr, quoteIdent(c.Name)))
	}
	selects = append(selects, "filename", "file_row_number")
	return fmt.Sprintf(
		"(SELECT %s FROM read_parquet([%s], filename=true, file_row_number=true, union_by_name=true))",
		strings.Join(selects, ", "), pathList(paths),
	)
}

// ColumnList returns the quoted canonical column names in output order.
func ColumnList() string {
	names := make([]string, len(Columns))
	for i, c := range Columns {
		names[i] = quoteIdent(c.Name)
	}
	return strings.Join(names, ", ")
}

// RowHashExpr returns a stable hash over every canonical column.
func RowHashExpr() string {
	return "hash(" + ColumnList() + ")"
}

// EmptySource returns a zero-row relation with the canonical schema so that
// queries over a time range with no data still resolve with correct columns.
func EmptySource() string {
	selects := make([]string, len(Columns))
	for i, c := range Columns {
		selects[i] = fmt.Sprintf("NULL::%s AS %s", c.Type, quoteIdent(c.Name))
	}
	return fmt.Sprintf("(SELECT %s WHERE 1 = 0)", strings.Join(selects, ", "))
}
