import { useMemo } from "react";
import { useEventsQuery } from "./useEventsQuery";
import { useTimeRange } from "./useUrlParams";
import {
  getDynamicInterval,
  intervalToSql,
  type Interval,
} from "../utils/time";
import {
  buildNamespaceBucketsQuery,
  type NamespaceBucketRow,
} from "../lib/sql/overview";

export const OTHER_NS = "(other)";

export interface NamespaceStat {
  ns: string;
  total: number;
  warnings: number;
  /** Per-bucket totals, aligned with `buckets`. */
  trend: number[];
  warnTrend: number[];
}

/**
 * Namespace × time-bucket counts over the global range, aggregated
 * client-side. Components sharing the same targetBuckets share one scan
 * (identical SWR key).
 */
export function useNamespaceBuckets(targetBuckets = 40): {
  buckets: string[];
  rows: NamespaceStat[]; // sorted by total desc
  interval: Interval;
  isLoading: boolean;
  error?: Error;
} {
  const { from, to } = useTimeRange();
  const interval = useMemo(
    () => getDynamicInterval(from, to, targetBuckets),
    [from, to, targetBuckets],
  );

  const { data, error, isLoading } = useEventsQuery<NamespaceBucketRow>(
    buildNamespaceBucketsQuery(intervalToSql(interval)),
    { scope: "overview" },
  );

  const { buckets, rows } = useMemo(() => {
    const raw = data ?? [];
    const buckets = [...new Set(raw.map((r) => r.bucket))].sort();
    const bucketIndex = new Map(buckets.map((b, i) => [b, i]));
    const byNs = new Map<string, NamespaceStat>();
    raw.forEach((r) => {
      let stat = byNs.get(r.ns);
      if (!stat) {
        stat = {
          ns: r.ns,
          total: 0,
          warnings: 0,
          trend: new Array<number>(buckets.length).fill(0),
          warnTrend: new Array<number>(buckets.length).fill(0),
        };
        byNs.set(r.ns, stat);
      }
      const i = bucketIndex.get(r.bucket);
      if (i === undefined) return;
      stat.total += r.total;
      stat.warnings += r.warnings;
      stat.trend[i] += r.total;
      stat.warnTrend[i] += r.warnings;
    });
    const rows = [...byNs.values()].sort((a, b) => b.total - a.total);
    return { buckets, rows };
  }, [data]);

  return { buckets, rows, interval, isLoading, error: error ?? undefined };
}

/** Keeps the top-N rows and folds the rest into a single "(other)" row. */
export function foldOther(
  rows: NamespaceStat[],
  bucketCount: number,
  topN: number,
): NamespaceStat[] {
  if (rows.length <= topN) return rows;
  const head = rows.slice(0, topN);
  const other: NamespaceStat = {
    ns: OTHER_NS,
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
