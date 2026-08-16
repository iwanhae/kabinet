# Frontend Development Guide

Welcome to the Kabinet frontend! This guide provides the necessary information to get you started with development.

The UI is designed around one constraint: reviewing ~1M events/day. Users never start from raw rows — they start from aggregates (timeline, heatmap, KPIs) and drill down into a virtualized, infinitely-scrolling event stream.

## Tech Stack

- **Framework**: [React 19](https://react.dev/) with [Vite](https://vitejs.dev/)
- **Language**: [TypeScript](https://www.typescriptlang.org/) (strict)
- **UI**: custom lightweight component system (`src/ui/`) — no component library
- **Styling**: CSS Modules + design tokens (`src/styles/tokens.css`), no runtime CSS-in-JS
- **Charts**: [Apache ECharts](https://echarts.apache.org/) (tree-shaken via `echarts/core`, canvas renderer)
- **Table virtualization**: [react-virtuoso](https://virtuoso.dev/) (`TableVirtuoso`)
- **Data Fetching**: [SWR](https://swr.vercel.app/) (+ `swr/infinite` for keyset pagination)
- **State**: URL parameters (source of truth) + React Context + [Zustand](https://zustand.docs.pmnd.rs/) (ephemeral query telemetry only)
- **Icons**: [lucide-react](https://lucide.dev/)
- **Fonts**: Inter Variable (UI) + IBM Plex Mono (data) via `@fontsource`
- **Routing**: [Wouter](https://github.com/molefrog/wouter)
- **Date & Time**: [Day.js](https://day.js.org/)

## Getting Started

```bash
npm install
npm run dev     # http://localhost:5173 (proxies /query, /download to :8080)
npm run lint    # eslint --fix + tsc -b (must pass before finishing work)
```

## Design System

The visual identity: rounded, friendly, blue. Interactive chrome (buttons, focus rings, active nav, selection) uses the `--accent` blue family. Data keeps its own semantic pair: `--steel` blue for Normal events, `--signal` red for Warning events (`--alert` shares the red family for UI errors).

- **Tokens**: `src/styles/tokens.css` defines all colors, spacing, radii, fonts as CSS custom properties. Dark mode is a `[data-theme="dark"]` override of the same tokens (toggled on `<html>` by `ThemeContext`). **Never hardcode hex values in components** — that's what broke dark mode in the previous design.
- **Canvas charts can't read CSS variables**, so `src/components/charts/chartTheme.ts` mirrors the palette in TS. If you change tokens.css, update it too.
- **Typography**: Inter for UI text; IBM Plex Mono is the "data voice" — timestamps, counts, reason codes, SQL, file paths. Use the global `.mono` class or `--font-mono`.
- **Primitives** (`src/ui/`): Button, IconButton, Chip, Card, Alert, Skeleton, Spinner, TextInput/TextArea/Select, Popover, Modal, Drawer, Accordion. Import from `../ui`. Add new primitives there only if used by 2+ features.

## Project Structure

```
src/
├── styles/          # tokens.css (design tokens), global.css (reset, base)
├── ui/              # component primitives (CSS Modules each)
├── lib/
│   ├── api/         # queryClient.ts — postQuery<T> returning results + scan meta
│   ├── filters/     # filter chip model, FIELD_DEFS registry, WHERE compiler, URL codec
│   ├── sql/         # TS_EXPR, keyset pagination builder, overview queries
│   └── agent/       # OpenAI client, tool-calling prompts, /query executor
├── components/
│   ├── charts/      # EChart wrapper, TimelineHistogram, CabinetHeatmap, SimpleBarLine
│   ├── dimension/   # DimensionPage — generic group-by table (Namespaces/Nodes/Components tabs)
│   ├── filters/     # global FilterBar + chip editor (rendered in Layout)
│   ├── explore/     # EventsVirtualTable, columns, detail panel
│   ├── overview/    # KpiStrip, TopMovers
│   ├── agent/       # CaseTranscript, ExhibitCard, Composer, history/settings
│   ├── Layout.tsx   # top bar (wordmark, nav, TimeRangePicker, theme toggle)
│   └── ScanCostBar.tsx  # footer "ledger stamp": last query's ms / files / bytes
├── contexts/        # ThemeContext (data-theme), RefreshContext (manual refresh)
├── hooks/           # useEventsQuery, useEventsInfinite, useFilters, useSortState, …
├── stores/          # queryMetaStore (zustand) — scan-cost telemetry
├── pages/           # Overview (/), Namespaces/Nodes/Components (thin DimensionPage wrappers), Explore (/p/discover), AgentPage (/agent, lazy)
├── types/           # EventResult, agent case-file types
└── utils/           # time buckets, relative time parsing, formatters
```

---

## Core Concepts & Conventions

### 1. Data fetching

All backend access goes through the hooks — never call `fetch` in components.

- **`useEventsQuery<T>(query, opts?)`** (`src/hooks/useEventsQuery.ts`) — single query within the global time range. Pass `null` to skip fetching. `opts.from/to` override the range (used by TopMovers' doubled window); `opts.scope` tags the cache and the scan-cost recording.
- **`useEventsQueryMeta<T>`** — same, but returns `{ results, meta }` including `duration_ms`, `files`, `total_files_size_bytes`.
- **`useEventsInfinite(whereSql, sort)`** (`src/hooks/useEventsInfinite.ts`) — keyset-paginated infinite scroll for the Explore table. Cursor is `(timestamp, metadata.uid)` (timestamps are second-precision, ties are common). Pre-compaction WAL duplicates (same uid) are deduplicated client-side keeping the highest resourceVersion.
- Every fetch records its scan cost into `queryMetaStore`, which `ScanCostBar` renders in the footer.
- Global SWR config lives in `App.tsx` (`keepPreviousData`, dedup, retry). Manual refresh works by folding `RefreshContext`'s counter into every SWR key — do not call `mutate()` globally.

### 2. Time handling

- The global range lives in URL params (`from`, `to`) as raw strings (`now-30m`, ISO). `useTimeRange()` returns both raw and parsed values plus `setTimeRange()`.
- Relative syntax: `now-<n><s|m|h|d|w>` (`src/utils/timeRange.ts`).
- Chart bucketing: `getDynamicInterval(from, to, targetBuckets)` returns a structured `Interval`; render SQL with `intervalToSql()` and compute bucket ends with `bucketEnd()` (`src/utils/time.ts`).
- **Always bucket/sort on `TS_EXPR`** (`src/lib/sql/expr.ts`) — `COALESCE(lastTimestamp, eventTime, metadata.creationTimestamp)` — or events.k8s.io events with null `lastTimestamp` silently vanish.

### 3. Filters (global)

- Filters are **global state like the time range**: the FilterBar renders in `Layout` on every data tab, filter params travel across navigation (`useNavigation`), and every query hook/builder applies `whereSql` (Overview KPIs/timeline/heatmap/movers, dimension tabs, Explore).
- Filter state is chips (`FilterChip { field, op, values }`) serialized into the `?filters=` URL param; raw SQL mode uses `?where=` (legacy links keep working). The two are mutually exclusive.
- Aggregate drill-downs go through `useFilters().drill(chips)`: it merges into the current filters (replacing chips on the same field) and navigates to Explore.
- **`FIELD_DEFS`** (`src/lib/filters/fields.ts`) is the single registry driving the chip editor, autocomplete, and the detail panel's click-to-filter. To add a filterable field, add it there — nowhere else.
- The compiler (`src/lib/filters/compile.ts`) escapes values (`''` doubling; `ILIKE` patterns additionally escape `%_\` with `ESCAPE '\'`). SQL identifiers come only from the registry — never interpolate user input as an identifier.
- Aggregate click-to-drill (KPI cards, heatmap cells, top movers) composes chips via `encodeFilters()` and navigates to `/p/discover`.

### 4. URL is the source of truth

Everything shareable lives in URL params: `from`, `to`, `filters`/`where`, `sort` (`ts:desc`), `uid` (detail panel selection; legacy `resourceVersion` still honored). Derive state from `useSearch()` via memoized hooks (`useFilters`, `useSortState`, `useTimeRange`); mutate only through `updateParams`. Do not duplicate URL state into `useState`.

Zustand is only for ephemeral cross-cutting UI state that has no business in the URL (currently just `queryMetaStore`).

### 5. Charts

- Use the `EChart` wrapper (`src/components/charts/EChart.tsx`) — never `echarts-for-react`. Register new chart/component types in `echartsSetup.ts` (tree-shaking).
- Get colors from `useChartTokens()` so dark mode works.
- `TimelineHistogram` supports brush-drag and bar-click to zoom the global time range; reuse it instead of writing new time histograms.

### 6. Agent page

Client-side OpenAI tool-calling loop (`useInvestigation`): the model calls `run_sql` (each call becomes a numbered **exhibit** in the case file) and `show_chart`; the final analysis streams as markdown. Query results are summarized (row count + first 30 rows) before re-entering model context. Config and case history persist in localStorage. The page is lazy-loaded (`React.lazy`) to keep the OpenAI SDK out of the main bundle.

### 7. Creating new components & pages

- Reusable pieces → `src/components/<feature>/`, primitives → `src/ui/`.
- Pages → `src/pages/` + route in `App.tsx`.
- Complex logic → custom hooks; keep components presentational.
- Styles → CSS Module next to the component, referencing tokens only.

---

By following these guidelines, we can ensure the frontend codebase remains clean, consistent, and easy to maintain.
