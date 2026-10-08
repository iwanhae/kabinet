import { TS_EXPR } from "./expr";
import { bucketNumberSql, bucketTimestampSql } from "./buckets";
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

/** All KPI-strip numbers in a single scan, within the global filters. */
export const buildKpiQuery = (whereSql: string): string => `
  SELECT
    COUNT(*) AS total_events,
    COUNT(*) FILTER (WHERE type = 'Warning') AS warning_events,
    COUNT(*) FILTER (WHERE reason IN (${inList(FAILED_POD_REASONS)})) AS failed_pods,
    COUNT(*) FILTER (WHERE reason = 'BackOff') AS restarts,
    COUNT(*) FILTER (WHERE type = 'Warning' AND reason IN (${inList(NODE_ISSUE_REASONS)})) AS node_issues,
    COUNT(*) FILTER (WHERE reason IN (${inList(STORAGE_REASONS)})) AS storage_events
  FROM $events
  WHERE (${whereSql})
`;

export interface KpiRow {
  total_events: number;
  warning_events: number;
  failed_pods: number;
  restarts: number;
  node_issues: number;
  storage_events: number;
}

/** Timeline counts by UTC bucket and event type. */
export const buildTimelineQuery = (
  intervalSeconds: number,
  whereSql: string,
): string => `
  WITH grouped AS (
    SELECT
      ${bucketNumberSql(intervalSeconds)} AS bucket_number,
      type,
      COUNT(*) AS count
    FROM $events
    WHERE (${whereSql})
    GROUP BY 1, 2
  )
  SELECT
    ${bucketTimestampSql(intervalSeconds)} AS time_bucket,
    type,
    count
  FROM grouped
  ORDER BY 1, 2
`;

/**
 * Per-dimension (namespace/node/component), per-bucket counts in a single
 * scan. `dimExpr` must come from the FIELD_DEFS whitelist — never user input.
 * Top-N capping and the "(other)" fold happen client-side (see
 * useDimensionBuckets) so heatmap, tables, and overview summaries share one
 * query per dimension.
 */
export const buildDimensionBucketsQuery = (
  dimExpr: string,
  intervalSeconds: number,
  whereSql: string,
): string => `
  WITH grouped AS (
    SELECT
      ${dimExpr} AS dim,
      ${bucketNumberSql(intervalSeconds)} AS bucket_number,
      COUNT(*) AS total,
      COUNT(*) FILTER (WHERE type = 'Warning') AS warnings
    FROM $events
    WHERE ${dimExpr} IS NOT NULL AND (${whereSql})
    GROUP BY 1, 2
  )
  SELECT
    dim,
    ${bucketTimestampSql(intervalSeconds)} AS bucket,
    total,
    warnings
  FROM grouped
  ORDER BY 1, 2
`;

export interface DimensionBucketRow {
  dim: string;
  bucket: string;
  total: number;
  warnings: number;
}

/**
 * Reason deltas vs the previous period. The request window must be doubled
 * (start = from - (to - from)) so the previous period is in scan range.
 */
export const buildTopMoversQuery = (
  fromIso: string,
  limit: number,
  whereSql: string,
): string => `
  SELECT
    reason,
    COUNT(*) FILTER (WHERE ${TS_EXPR} >= TIMESTAMPTZ '${escapeSqlString(fromIso)}') AS current_count,
    COUNT(*) FILTER (WHERE ${TS_EXPR} <  TIMESTAMPTZ '${escapeSqlString(fromIso)}') AS previous_count
  FROM $events
  WHERE (${whereSql})
  GROUP BY reason
  ORDER BY abs(current_count - previous_count) DESC
  LIMIT ${limit}
`;

export interface TopMoverRow {
  reason: string;
  current_count: number;
  previous_count: number;
}
