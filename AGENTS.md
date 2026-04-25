# Repository Guidelines

## Project Summary
Kabinet collects Kubernetes events in real time, stores them in DuckDB, archives to ZSTD-compressed Parquet, and serves a React UI for dashboards, SQL exploration, and AI-powered investigation.

## Architecture

**Two binaries:**
- `cmd/server/main.go` — main server: K8s event collector, DuckDB storage lifecycle, REST API, embedded React frontend (`//go:embed all:dist`)
- `cmd/merger/main.go` — standalone CLI for compacting Parquet files; invoked by the server's lifecycle manager via `exec.Command`

**Backend packages:**
- `internal/api` — HTTP handlers: `/query`, `/stats`, `/download` (gzipped JSONL), `/metrics` (Prometheus), `/debug/pprof/*`, SPA fallback routing
- `internal/collector` — K8s `client-go` SharedInformer watching all events, sends to buffered channel (2000)
- `internal/storage` — DuckDB + Parquet lifecycle: batch insert (1000-event batches, 5s flush), archive at 122,880 rows (table-swap transaction), compact small files (<128MB) via `merger`, retention pruning
- `internal/config` — Env vars: `STORAGE_LIMIT_GB` (default 10), `LISTEN_PORT` (default 8080)
- `internal/metrics` — Prometheus counter `kabinet_events_collected_total`
- `internal/utils` — `MultiError` type

**Frontend (`src/`):**
- React 19, TypeScript 5.8 (strict), Vite 7, MUI 7, SWR, Wouter (router), Zustand (state)
- Routes: `/` (Insight dashboard), `/p/discover` (SQL query builder), `/agent` (AI investigation via OpenAI)
- `useEventsQuery<T>(query)` — primary data hook; SWR-based, auto-injects time range from URL params (`from`, `to` in relative format like `now-30m`)
- `src/lib/agent/` — OpenAI integration for AI-powered K8s event investigation

## Build, Test, and Development Commands

**Dev:**
```
npm install && npm run dev     # UI at http://localhost:5173 (proxies /query, /download to backend)
go run ./cmd/server/main.go    # API at http://localhost:8080
```

**Build:**
```
npm run build                              # outputs to dist/
cp -r dist/ cmd/server/dist/               # frontend must be here for go:embed
go build -o server ./cmd/server/main.go    # main binary
go build -o merger ./cmd/merger/main.go    # parquet compaction tool
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

- `$events` macro in queries expands to `UNION ALL BY NAME` of live `kube_events` + relevant Parquet files filtered by time range — always include narrow `start`/`end`
- Storage lifecycle runs every 1 minute; archiving uses table-swap transaction (rename → create fresh → export to parquet in background)
- ESLint flat config (`eslint.config.js`); Prettier runs via `eslint-plugin-prettier` (no separate `.prettierrc`)
- Go module: `github.com/iwanhae/kabinet`, requires Go 1.25
- Conventional Commits: `feat|fix|refactor|docs|chore(scope): message`

## Agent Checklist (Before Finishing Work)
- `npm run lint` — must pass (lint + typecheck)
- `go build ./cmd/server/` — must compile
- `go test ./...` — must pass

## Development Guides
- Frontend: `DEVELOPMENT_GUIDE_FRONTED.md` — stack, routing, `useEventsQuery`, time-range handling, component patterns
- API/Querying: `DEVELOPMENT_QUERY_GUIDE.md` — `POST /query` payload/response, `$events` usage, performance tips
