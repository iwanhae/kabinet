import { useMemo } from "react";
import { useEventsQuery } from "./useEventsQuery";
import { useFilters } from "./useFilters";
import { useTimeRange } from "./useUrlParams";
import {
  getDynamicInterval,
  intervalToSql,
  type Interval,
} from "../utils/time";
import {
  buildDimensionBucketsQuery,
  type DimensionBucketRow,
} from "../lib/sql/overview";
import { FIELD_DEFS } from "../lib/filters/fields";

export const OTHER_KEY = "(other)";

/** Group-by dimensions backed by the FIELD_DEFS whitelist. */
export type DimensionField = "namespace" | "host" | "component";

export interface DimensionStat {
  key: string;
  total: number;
  warnings: number;
  /** Per-bucket totals, aligned with `buckets`. */
  trend: number[];
  warnTrend: number[];
}

/**
 * Dimension × time-bucket counts over the global time range AND global
 * filters, aggregated client-side. Components sharing the same dimension +
 * targetBuckets share one scan (identical SWR key).
 */
export function useDimensionBuckets(
  field: DimensionField,
  targetBuckets = 40,
): {
  buckets: string[];
  rows: DimensionStat[]; // sorted by total desc
  interval: Interval;
  isLoading: boolean;
  error?: Error;
} {
  const { from, to } = useTimeRange();
  const { whereSql } = useFilters();
  const interval = useMemo(
    () => getDynamicInterval(from, to, targetBuckets),
    [from, to, targetBuckets],
  );

  const { data, error, isLoading } = useEventsQuery<DimensionBucketRow>(
    buildDimensionBucketsQuery(
      FIELD_DEFS[field].sqlExpr,
      intervalToSql(interval),
      whereSql,
    ),
    { scope: "overview" },
  );

  const { buckets, rows } = useMemo(() => {
    const raw = data ?? [];
    const buckets = [...new Set(raw.map((r) => r.bucket))].sort();
    const bucketIndex = new Map(buckets.map((b, i) => [b, i]));
    const byKey = new Map<string, DimensionStat>();
    raw.forEach((r) => {
      let stat = byKey.get(r.dim);
      if (!stat) {
        stat = {
          key: r.dim,
          total: 0,
          warnings: 0,
          trend: new Array<number>(buckets.length).fill(0),
          warnTrend: new Array<number>(buckets.length).fill(0),
        };
        byKey.set(r.dim, stat);
      }
      const i = bucketIndex.get(r.bucket);
      if (i === undefined) return;
      stat.total += r.total;
      stat.warnings += r.warnings;
      stat.trend[i] += r.total;
      stat.warnTrend[i] += r.warnings;
    });
    const rows = [...byKey.values()].sort((a, b) => b.total - a.total);
    return { buckets, rows };
  }, [data]);

  return { buckets, rows, interval, isLoading, error: error ?? undefined };
}

/** Keeps the top-N rows and folds the rest into a single "(other)" row. */
export function foldOther(
  rows: DimensionStat[],
  bucketCount: number,
  topN: number,
): DimensionStat[] {
  if (rows.length <= topN) return rows;
  const head = rows.slice(0, topN);
  const other: DimensionStat = {
    key: OTHER_KEY,
    total: 0,
    warnings: 0,
    trend: new Array<number>(bucketCount).fill(0),
    warnTrend: new Array<number>(bucketCount).fill(0),
  };
  rows.slice(topN).forEach((r) => {
    other.total += r.total;
    other.warnings += r.warnings;
    r.trend.forEach((v, i) => (other.trend[i] += v));
    r.warnTrend.forEach((v, i) => (other.warnTrend[i] += v));
  });
  return [...head, other];
}
