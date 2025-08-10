# Repository Guidelines

## Project Summary
Kabinet is a single‑binary tool that collects Kubernetes events in real time, stores recent data in DuckDB, and archives history to ZSTD‑compressed Parquet. It exposes a REST API for SQL queries that unify live and historical data and serves a React UI for dashboards and ad‑hoc exploration. Designed to be lightweight and fast, it avoids ELK‑style complexity while providing rich analytics, a query builder, and time‑range aware fetching.

For more details, please refer to the [README](README.md).

## Project Structure & Module Organization
- `internal/api`: HTTP server and handlers (`/query`, `/stats`).
- `internal/collector`: Watches Kubernetes events (client-go informer).
- `internal/storage`: DuckDB + Parquet lifecycle (archive, compact, prune).
- `src/`: React + TypeScript UI (components, hooks, pages, utils, theme).
- `data/`: Default DuckDB/Parquet storage (mounted in Docker).
- `main.go`: App entrypoint; serves API and embedded frontend.

## Build, Test, and Development Commands
- Install deps: `go mod download` | `npm install`.
- Dev servers: `npm run dev` (UI at http://localhost:5173), `go run main.go` (API at http://localhost:8080).
- Production build: `npm run build` then `go build -o kabinet main.go`.
- Docker: `docker build -t kabinet .` and run with volume mounts for `~/.kube/config` and `./data`.
- Lint (frontend): `npm run lint`.
- Tests (if present): `go test ./...` and UI tests via your chosen runner.

## Agent Checklist (Before Finishing Work)
- Run frontend lint: `npm run lint`
- Verify backend builds: `go build -o main .`

## Coding Style & Naming Conventions
- Go: `go fmt ./...`; idiomatic Go; packages lowercase, exported `CamelCase`, errors wrapped with context.
- TypeScript/React: ESLint; components `PascalCase` (e.g., `TimeRangePicker.tsx`), hooks `useX`, utility files `camelCase.ts`.
- UI: Prefer MUI `styled` and theme tokens; keep layout/logic separated via hooks.

## Testing Guidelines
- Go: Add unit tests for storage and API handlers (`*_test.go`), focus on time‑range and archival logic.
- Frontend: Add tests for hooks/components where feasible (data fetching and time range behavior).
- Aim for small, deterministic tests; include table-driven tests in Go.

## Commit & Pull Request Guidelines
- Use Conventional Commits: `feat|fix|refactor|docs|chore(scope): message` (e.g., `fix(storage): use UNION ALL`).
- PRs: concise description, linked issues, reproduction steps, and screenshots for UI changes.
- Keep changes focused; update docs when flags, endpoints, or UI flows change.

## Security & Configuration Tips
- Kube access: mount read-only kubeconfig; avoid committing secrets.
- Storage: control footprint via `STORAGE_LIMIT_GB`; persist `data/`.
- Performance: always include narrow `start`/`end` in queries; prefer `$events` for unified data.

## Development Guides
- Frontend: see `DEVELOPMENT_GUIDE_FRONTED.md` for stack, routing, `useEventsQuery`, time‑range handling, and component patterns.
- API/Querying: see `DEVELOPMENT_QUERY_GUIDE.md` for `POST /query` payload/response, `$events` table usage, and performance advice.
