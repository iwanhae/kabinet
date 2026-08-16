# Repository Guidelines

## Project Summary
Kabinet collects Kubernetes events in real time into a zstd-compressed JSONL write-ahead log, compacts them into deduplicated ZSTD Parquet files, and serves a React UI for dashboards, SQL exploration, and AI-powered investigation.

## Architecture

Three strictly separated layers that communicate only through the filesystem and the in-memory catalog:

1. **Ingest** (`internal/wal`) — raw K8s Event JSON appended to `data/wal/*.jsonl.zst` segments. Each batch flush is one complete zstd frame (crash loses at most the last unflushed batch; a torn tail is truncated at recovery). Segments rotate by time/size, are sealed by rename, and are named by their event-time range (`events_<minMs>_<maxMs>.jsonl.zst`).
2. **Manage** (`internal/lifecycle` + `internal/compact`) — a scheduler converts sealed segments to L1 Parquet and merges L1 files into L2 (`data/archive/l1`, `l2`), deduplicating by `(metadata.uid, metadata.resourceVersion)` on every pass. Heavy work runs in the `compactor` subprocess (JSON job on stdin, result on stdout) with DuckDB `memory_limit` + disk spill, an RSS self-kill watchdog, and `oom_score_adj=1000` so OOM takes out the compactor, never the server. Retention deletes oldest archive files past `STORAGE_LIMIT_GB`.
3. **Query** (`internal/query`) — resolves the `$events` macro into a `UNION ALL BY NAME` of overlapping Parquet files (via catalog) and raw WAL segments (sealed + a stable snapshot of the active one), executed on a read-only in-memory DuckDB.

**Two binaries:**
- `cmd/server/main.go` — main server: K8s event collector, WAL, lifecycle scheduler, REST API, embedded React frontend (`//go:embed all:dist`)
- `cmd/compactor/main.go` — subprocess for convert/merge jobs; located via `KABINET_COMPACTOR_PATH`, next to the server binary, `PATH`, or `/usr/local/bin`

**Backend packages:**
- `internal/schema` — single source of truth for the canonical event schema: `read_json` column specs, raw-JSON→canonical projection SQL (timestamp/count fallbacks), dedup `QUALIFY` clause
- `internal/collector` — K8s `client-go` SharedInformer (Add + Update handlers) watching all events
- `internal/wal` — segment writer, sealing, crash recovery, active-segment snapshots
- `internal/catalog` — in-memory index of archive Parquet files; the directory is the source of truth (time range + seq encoded in filenames), rebuilt by scan at startup
- `internal/compact` — convert/merge job execution (runs inside `compactor`) + OOM guard
- `internal/lifecycle` — scheduling, compactor subprocess invocation, retention, grace-period deletes
- `internal/query` — `$events` planning and execution, row serialization/streaming
- `internal/api` — HTTP handlers: `/query`, `/stats`, `/download` (gzipped JSONL), `/metrics` (Prometheus), `/debug/pprof/*`, SPA fallback routing
- `internal/config` — Env vars: `DATA_DIR` (default `data`), `STORAGE_LIMIT_GB` (10), `LISTEN_PORT` (8080), `WAL_ROTATE_SECONDS` (60), `WAL_ROTATE_MB` (8), `COMPACT_INTERVAL_SECONDS` (600), `COMPACT_MEMORY_LIMIT_MB` (512), `MERGE_TARGET_MB` (128)
- `internal/metrics` — Prometheus counter `kabinet_events_collected_total`
- `internal/utils` — `MultiError` type

**Dedup semantics:** one row per `(metadata.uid, metadata.resourceVersion)`, enforced at every compaction/merge pass — never at query time, so recently re-listed events may appear duplicated until their segments are compacted.

**Frontend (`src/`):**
- React 19, TypeScript 5.8 (strict), Vite 7, SWR, Wouter (router) — no component library: custom primitives in `src/ui/` styled with CSS Modules + design tokens (`src/styles/tokens.css`, dark mode = `[data-theme="dark"]` token override)
- Charts: Apache ECharts (tree-shaken via `echarts/core`, wrapper in `src/components/charts/EChart.tsx`); tables: react-virtuoso
- Routes: `/` (Overview: brushable timeline, one-query KPI strip, namespace×time heatmap, top namespaces/movers), `/p/namespaces` (per-namespace counts + trend sparklines), `/p/discover` (Explore: filter chips → virtualized infinite-scroll table → detail panel), `/agent` (AI investigation, tool-calling + streaming, lazy-loaded)
- Data hooks: `useEventsQuery<T>(query, opts?)` (SWR, auto-injects time range from URL params like `now-30m`), `useEventsInfinite(whereSql, sort)` (keyset pagination, cursor = timestamp+uid, client-side uid dedup)
- URL params are the source of truth (`from`/`to`/`filters`/`where`/`sort`/`uid`); filter fields are whitelisted in `src/lib/filters/fields.ts` (FIELD_DEFS registry) and compiled to escaped WHERE clauses
- Time-based SQL must use `TS_EXPR` (`src/lib/sql/expr.ts`) so null-`lastTimestamp` events aren't dropped
- Every query's scan cost (duration/files/bytes) is recorded to a zustand store and shown in the footer `ScanCostBar`

## Build, Test, and Development Commands

**Dev:**
```
npm install && npm run dev     # UI at http://localhost:5173 (proxies /query, /download to backend)
go run ./cmd/server/main.go    # API at http://localhost:8080
```

**Build:**
```
npm run build                                  # outputs to dist/
cp -r dist/ cmd/server/dist/                   # frontend must be here for go:embed
go build -o server ./cmd/server/main.go        # main binary
go build -o compactor ./cmd/compactor/main.go  # compaction subprocess
```

**Verify:**
```
npm run lint              # runs eslint --fix AND tsc -b (type checking included)
go build ./cmd/server/    # verify backend compiles (requires CGO for DuckDB)
go test ./...             # Go tests
```

**Docker:**
```
docker build -t kabinet .  # multi-stage: node build -> go build with CGO_ENABLED=1 -> debian:12-slim runtime
```

## Key Conventions

- `$events` macro in queries expands to `UNION ALL BY NAME` of relevant Parquet files + raw WAL segments filtered by time range — always include narrow `start`/`end`
- The lifecycle scheduler ticks every 1 minute; segments convert when the backlog exceeds 32MB or `COMPACT_INTERVAL_SECONDS`; L1 merges into L2 at `MERGE_TARGET_MB` or 96 files
- Data files are immutable once published; deletions are delayed by a grace period so in-flight queries finish
- ESLint flat config (`eslint.config.js`); Prettier runs via `eslint-plugin-prettier` (no separate `.prettierrc`)
- Go module: `github.com/iwanhae/kabinet`, requires Go 1.25
- Conventional Commits: `feat|fix|refactor|docs|chore(scope): message`

## Agent Checklist (Before Finishing Work)
- `npm run lint` — must pass (lint + typecheck)
- `go build ./cmd/server/` — must compile
- `go test ./...` — must pass

## Development Guides
- Frontend: `DEVELOPMENT_GUIDE_FRONTED.md` — design tokens, `src/ui` primitives, data hooks, filter model, chart conventions
- API/Querying: `DEVELOPMENT_QUERY_GUIDE.md` — `POST /query` payload/response, `$events` usage, performance tips
