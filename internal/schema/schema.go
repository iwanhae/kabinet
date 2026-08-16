// Package schema is the single source of truth for the canonical event
// schema. The WAL stores raw Kubernetes Event JSON; everything that reads it
// (the compactor converting to Parquet, the query planner reading recent
// segments) projects rows into this canonical shape using the SQL fragments
// defined here.
package schema

import (
	"fmt"
	"strings"
)

const (
	objectRefType = `STRUCT(kind VARCHAR, namespace VARCHAR, "name" VARCHAR, uid VARCHAR, apiVersion VARCHAR, resourceVersion VARCHAR, fieldPath VARCHAR)`
	metadataType  = `STRUCT("name" VARCHAR, "namespace" VARCHAR, uid VARCHAR, resourceVersion VARCHAR, creationTimestamp TIMESTAMPTZ)`
	sourceType    = `STRUCT(component VARCHAR, host VARCHAR)`
	seriesType    = `STRUCT("count" INTEGER, lastObservedTime TIMESTAMPTZ)`
)

// Column is one canonical output column.
type Column struct {
	Name string
	Type string
	// rawExpr projects the column out of a raw K8s Event JSON row.
	// Empty means the raw column is taken as-is.
	rawExpr string
}

// Columns lists every canonical column in output order. The rawExpr fallbacks
// replace the field normalization that used to happen at collection time
// (empty firstTimestamp/lastTimestamp/count on some event sources).
var Columns = []Column{
	{Name: "kind", Type: "VARCHAR", rawExpr: `COALESCE(kind, 'Event')`},
	{Name: "apiVersion", Type: "VARCHAR", rawExpr: `COALESCE(apiVersion, 'v1')`},
	{Name: "metadata", Type: metadataType},
	{Name: "involvedObject", Type: objectRefType},
	{Name: "reason", Type: "VARCHAR"},
	{Name: "message", Type: "VARCHAR"},
	{Name: "source", Type: sourceType},
	{Name: "firstTimestamp", Type: "TIMESTAMPTZ", rawExpr: `COALESCE(firstTimestamp, metadata.creationTimestamp)`},
	{Name: "lastTimestamp", Type: "TIMESTAMPTZ", rawExpr: `COALESCE(lastTimestamp, firstTimestamp, metadata.creationTimestamp)`},
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
const DedupQualify = `QUALIFY row_number() OVER (PARTITION BY metadata.uid, metadata.resourceVersion ORDER BY lastTimestamp DESC) = 1`

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

// EmptySource returns a zero-row relation with the canonical schema so that
// queries over a time range with no data still resolve with correct columns.
func EmptySource() string {
	selects := make([]string, len(Columns))
	for i, c := range Columns {
		selects[i] = fmt.Sprintf("NULL::%s AS %s", c.Type, quoteIdent(c.Name))
	}
	return fmt.Sprintf("(SELECT %s WHERE 1 = 0)", strings.Join(selects, ", "))
}
