// Package mcpserver exposes the event store to AI assistants over the Model
// Context Protocol (streamable HTTP, mounted at /mcp by the API server).
package mcpserver

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/iwanhae/kabinet/internal/query"
	"github.com/iwanhae/kabinet/internal/schema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// StatsFunc assembles the get_stats payload; cmd/server wires in the same
// function that backs the /stats endpoint.
type StatsFunc func(ctx context.Context) map[string]any

// maxRows caps query_events results so a query without LIMIT cannot flood
// the model's context window.
const maxRows = 1000

type queryInput struct {
	Query string `json:"query" jsonschema:"DuckDB SQL statement selecting FROM $events"`
	Start string `json:"start" jsonschema:"start of the scan window in RFC3339 (e.g. 2026-08-16T00:00:00Z)"`
	End   string `json:"end" jsonschema:"end of the scan window in RFC3339"`
}

type queryOutput struct {
	Results      []map[string]any `json:"results"`
	RowCount     int              `json:"row_count"`
	Truncated    bool             `json:"truncated"`
	DurationMs   int64            `json:"duration_ms"`
	ScannedFiles int              `json:"scanned_files"`
	ScannedBytes int64            `json:"scanned_bytes"`
}

// NewHandler builds the streamable-HTTP MCP handler. The server is stateless:
// every request carries its own session, so the endpoint works behind load
// balancers and needs no session affinity.
func NewHandler(executor *query.Executor, stats StatsFunc) http.Handler {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "kabinet",
		Title:   "Kabinet — Kubernetes event analytics",
		Version: "1.0.0",
	}, &mcp.ServerOptions{Instructions: instructions()})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "query_events",
		Description: queryToolDescription(),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in queryInput) (*mcp.CallToolResult, queryOutput, error) {
		var out queryOutput
		start, err := time.Parse(time.RFC3339, in.Start)
		if err != nil {
			return nil, out, fmt.Errorf("invalid start (want RFC3339): %w", err)
		}
		end, err := time.Parse(time.RFC3339, in.End)
		if err != nil {
			return nil, out, fmt.Errorf("invalid end (want RFC3339): %w", err)
		}

		rows, result, err := executor.RangeQuery(ctx, in.Query, start, end)
		if err != nil {
			return nil, out, err
		}

		if len(rows) > maxRows {
			rows = rows[:maxRows]
			out.Truncated = true
		}
		if rows == nil {
			rows = []map[string]any{}
		}
		out.Results = rows
		out.RowCount = len(rows)
		out.DurationMs = result.Duration.Milliseconds()
		out.ScannedFiles = len(result.Files)
		for _, f := range result.Files {
			out.ScannedBytes += f.Size
		}
		return nil, out, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_stats",
		Description: "Get storage statistics for the event store: WAL ingest state and " +
			"archive (Parquet) file counts and total size. Useful to check how much data " +
			"exists before planning wide-range queries.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, map[string]any, error) {
		return nil, stats(ctx), nil
	})

	return mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return server },
		&mcp.StreamableHTTPOptions{Stateless: true},
	)
}

// schemaDoc renders the canonical event schema from the single source of
// truth in internal/schema, so the tool description never drifts.
func schemaDoc() string {
	var b strings.Builder
	for _, c := range schema.Columns {
		fmt.Fprintf(&b, "  %s %s\n", c.Name, c.Type)
	}
	return b.String()
}

func instructions() string {
	return `Kabinet stores every Kubernetes Event of one cluster and lets you analyze them with DuckDB SQL.

Workflow for investigations:
1. Start with get_stats to see how much data exists.
2. Aggregate first: GROUP BY type/reason/namespace over the relevant window to find anomalies.
3. Drill down: narrow the time window and add WHERE filters, then fetch raw rows with a small LIMIT to read actual messages.

Always pass the narrowest start/end window that answers the question — the window controls how many files are scanned.`
}

func queryToolDescription() string {
	return `Run a read-only DuckDB SQL query over Kubernetes events.

RULES
- Query FROM $events — a macro, not a real table. It expands to exactly the data files overlapping [start, end], so ALWAYS pass the narrowest time window that answers the question.
- start and end are required, RFC3339.
- Use lastTimestamp for all time filters and bucketing. Do NOT use eventTime (frequently NULL).
- Nested fields use dot notation: metadata.namespace, involvedObject.name, source.host.
- type is 'Normal' or 'Warning'. Warnings are where problems live.
- Node name: source.host. Controller/component: COALESCE(source.component, reportingComponent).
- Aggregate first (GROUP BY), then fetch raw rows with a tight WHERE and a small LIMIT. Results are truncated at ` + fmt.Sprint(maxRows) + ` rows ("truncated": true).
- count is the per-event dedup counter (how many times the event repeated); use SUM("count") for true occurrence totals, COUNT(*) for event-row counts.
- Recently ingested events can appear duplicated until background compaction dedups them by (metadata.uid, metadata.resourceVersion). For exact numbers add: QUALIFY row_number() OVER (PARTITION BY metadata.uid, metadata.resourceVersion ORDER BY lastTimestamp DESC) = 1

SCHEMA of $events
` + schemaDoc() + `
EXAMPLES
Warning reasons, most frequent first:
  SELECT reason, COUNT(*) AS c FROM $events WHERE type = 'Warning' GROUP BY reason ORDER BY c DESC LIMIT 20

Timeline of warnings in one namespace (5-minute buckets):
  SELECT time_bucket(INTERVAL 5 MINUTE, lastTimestamp) AS bucket, COUNT(*) AS c
  FROM $events WHERE type = 'Warning' AND metadata.namespace = 'prod'
  GROUP BY bucket ORDER BY bucket

Which pods are failing and why:
  SELECT involvedObject.name AS pod, reason, COUNT(*) AS c
  FROM $events WHERE type = 'Warning' AND involvedObject.kind = 'Pod'
  GROUP BY pod, reason ORDER BY c DESC LIMIT 20

Read raw messages after narrowing down:
  SELECT lastTimestamp, reason, message FROM $events
  WHERE involvedObject.name = 'my-pod-abc123' ORDER BY lastTimestamp DESC LIMIT 50`
}
