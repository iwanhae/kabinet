import { TS_EXPR } from "./expr";
import { escapeSqlString } from "../filters/compile";

export const FAILED_POD_REASONS = [
  "FailedScheduling",
  "Evicted",
  "FailedCreatePodSandBox",
];

export const NODE_ISSUE_REASONS = [
  "NodeNotReady",
  "NodeHasDiskPressure",
  "Unhealthy",
  "TaintManagerEviction",
  "NodeNotSchedulable",
  "ImageGCFailed",
  "FreeDiskSpaceFailed",
  "FailedSync",
];

export const STORAGE_REASONS = [
  "FailedAttachVolume",
  "FailedMount",
  "VolumeFailedDelete",
];

const inList = (values: string[]) =>
  values.map((v) => `'${escapeSqlString(v)}'`).join(", ");

/** All KPI-strip numbers in a single scan. */
export const buildKpiQuery = (): string => `
  SELECT
    COUNT(*) AS total_events,
    COUNT(*) FILTER (WHERE type = 'Warning') AS warning_events,
    COUNT(DISTINCT metadata.namespace) AS active_namespaces,
    COUNT(DISTINCT involvedObject.name) AS distinct_objects,
    COUNT(*) FILTER (WHERE reason IN (${inList(FAILED_POD_REASONS)})) AS failed_pods,
    COUNT(*) FILTER (WHERE reason = 'BackOff') AS restarts,
    COUNT(*) FILTER (WHERE type = 'Warning' AND reason IN (${inList(NODE_ISSUE_REASONS)})) AS node_issues,
    COUNT(*) FILTER (WHERE reason IN (${inList(STORAGE_REASONS)})) AS storage_events
  FROM $events
`;

export interface KpiRow {
  total_events: number;
  warning_events: number;
  active_namespaces: number;
  distinct_objects: number;
  failed_pods: number;
  restarts: number;
  node_issues: number;
  storage_events: number;
}

/**
 * Per-namespace, per-bucket counts in a single scan. Top-N capping and the
 * "(other)" fold happen client-side (see useNamespaceBuckets) so the heatmap,
 * the namespaces table, and the overview summary share one query.
 */
export const buildNamespaceBucketsQuery = (intervalSql: string): string => `
  SELECT
    metadata.namespace AS ns,
    time_bucket(INTERVAL '${intervalSql}', ${TS_EXPR}) AS bucket,
    COUNT(*) AS total,
    COUNT(*) FILTER (WHERE type = 'Warning') AS warnings
  FROM $events
  WHERE metadata.namespace IS NOT NULL
  GROUP BY 1, 2
  ORDER BY 1, 2
`;

export interface NamespaceBucketRow {
  ns: string;
  bucket: string;
  total: number;
  warnings: number;
}

/**
 * Reason deltas vs the previous period. The request window must be doubled
 * (start = from - (to - from)) so the previous period is in scan range.
 */
export const buildTopMoversQuery = (fromIso: string, limit = 15): string => `
  SELECT
    reason,
    COUNT(*) FILTER (WHERE ${TS_EXPR} >= TIMESTAMPTZ '${escapeSqlString(fromIso)}') AS current_count,
    COUNT(*) FILTER (WHERE ${TS_EXPR} <  TIMESTAMPTZ '${escapeSqlString(fromIso)}') AS previous_count
  FROM $events
  GROUP BY reason
  ORDER BY abs(current_count - previous_count) DESC
  LIMIT ${limit}
`;

export interface TopMoverRow {
  reason: string;
  current_count: number;
  previous_count: number;
}
