# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Kabinet is a single-binary Kubernetes event filing cabinet. It collects cluster events in real-time via the Kubernetes WATCH API, stores recent data in DuckDB for fast queries, and archives to compressed Parquet files for long-term storage. It serves a React-based web UI for analytics and event exploration.

### Tech Stack

**Backend (Go 1.25+):**
- `client-go` informers for Kubernetes event watching
- DuckDB for real-time analytics + Parquet for archival storage
- Standard library HTTP server with embedded frontend
- Prometheus metrics for monitoring

**Frontend (React 19 + TypeScript):**
- Vite for development/building
- Material-UI (MUI) for components
- SWR for data fetching, Zustand for state management
- ApexCharts for visualizations
- Wouter for routing

## Build and Development Commands

```bash
# Install dependencies
go mod download          # Go backend
npm install              # Frontend

# Development (two terminals)
npm run dev              # Frontend dev server at http://localhost:5173
go run cmd/server/main.go  # Backend API at http://localhost:8080

# Production build
npm run build            # Build frontend to dist/
go build -o kabinet cmd/server/main.go  # Build binary with embedded UI

# Linting
npm run lint             # Frontend ESLint + TypeScript check
go test ./...            # Backend tests
```

**Docker:**
```bash
docker build -t kabinet .
docker run -d -p 8080:8080 -v ~/.kube/config:/root/.kube/config:ro -v $(pwd)/data:/data kabinet
```

## Architecture

### Data Flow

```
K8s API Server → Event Collector (client-go informers) → DuckDB Table → Parquet Archives (zstd)
                                                                           ↓
Users → React Web UI → HTTP API → DuckDB Query Engine → Unified Results
```

### Key Backend Modules

- `internal/collector/`: Kubernetes event collection using client-go informers with automatic reconnection
- `internal/storage/`: DuckDB + Parquet lifecycle management (archive, compact, prune)
- `internal/api/`: HTTP handlers for `/query`, `/stats`, `/download`, and static files
- `internal/config/`: Configuration from environment variables
- `internal/metrics/`: Prometheus metrics collection

### Storage Lifecycle

1. Events are inserted into DuckDB `kube_events` table in batches
2. When table reaches ~1.2M rows, it's archived to a zstd-compressed Parquet file
3. Smaller Parquet files are periodically compacted
4. Oldest files are deleted when storage exceeds `STORAGE_LIMIT_GB` (default: 10GB)
5. Queries automatically union both DuckDB and Parquet data based on time range

### Frontend Structure

- `src/pages/`: Main pages (Insight analytics dashboard, Discover query builder)
- `src/components/`: Reusable UI components
- `src/hooks/`: Custom hooks including `useEventsQuery` for all data fetching
- `src/contexts/`: Global state (theme, refresh)
- `src/types/`: TypeScript definitions

## Important Development Patterns

### Frontend Data Fetching

**Always use `useEventsQuery` hook** for fetching data from the API. It automatically:
- Combines SQL queries with the global time range
- Handles caching, loading, and error states via SWR
- Provides full type safety when passed a result type interface

```tsx
const { data, error, isLoading } = useEventsQuery<YourType>(
  "SELECT reason, COUNT(*) as count FROM $events WHERE type = 'Warning' GROUP BY reason"
);
```

### Time Range Management

The time range is global state stored in URL params and React Context. Components using `useEventsQuery` automatically re-fetch when time range changes. Use the `TimeRangePicker` component for UI control.

### Querying the Backend

The `$events` macro represents all events within the specified time range (unions DuckDB + Parquet). Always include narrow `start`/`end` parameters for performance.

**Timestamp field:** Use `lastTimestamp` for temporal queries (NOT `eventTime`, which is deprecated).

### API Endpoints

- `POST /query`: SQL queries with time range, returns `{results, duration_ms, files, total_files_size_bytes}`
- `GET /stats`: System statistics
- `GET /download`: Export events as gzipped JSON Lines
- `GET /metrics`: Prometheus metrics

### Styling Conventions

- Use MUI `styled` API for complex/reusable components
- Use `sx` prop for simple one-off styles
- Define theme values in `src/theme.ts`
- Components: `PascalCase.tsx`, hooks: `useX.ts`, utilities: `camelCase.ts`

### Go Conventions

- Run `go fmt ./...` before committing
- Packages lowercase, exports `CamelCase`
- Wrap errors with context using `fmt.Errorf`

## Environment Configuration

- `STORAGE_LIMIT_GB`: Max data directory size (default: 10)
- `LISTEN_PORT`: API server port (default: 8080)
- Requires valid `kubeconfig` for Kubernetes access

## Entry Points

- `cmd/server/main.go`: Main application entry point
- `src/main.tsx`: Frontend React entry point

## Development Guides

- `DEVELOPMENT_GUIDE_FRONTED.md`: Frontend development details
- `DEVELOPMENT_QUERY_GUIDE.md`: API usage and SQL query examples
- `AGENTS.md`: Commit conventions and coding standards
