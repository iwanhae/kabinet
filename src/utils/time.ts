import dayjs from "dayjs";

export interface Interval {
  value: number;
  unit: "second" | "minute" | "hour" | "day" | "week";
  seconds: number;
}

const iv = (value: number, unit: Interval["unit"], seconds: number): Interval =>
  ({ value, unit, seconds }) as const;

const INTERVALS: Interval[] = [
  iv(10, "second", 10),
  iv(15, "second", 15),
  iv(30, "second", 30),
  iv(1, "minute", 60),
  iv(5, "minute", 300),
  iv(15, "minute", 900),
  iv(30, "minute", 1800),
  iv(1, "hour", 3600),
  iv(3, "hour", 10800),
  iv(6, "hour", 21600),
  iv(12, "hour", 43200),
  iv(1, "day", 86400),
  iv(1, "week", 604800),
  iv(2, "week", 1209600),
];

/**
 * Picks the smallest bucket interval that keeps the bucket count at or below
 * `targetBuckets` for the given time range.
 */
export function getDynamicInterval(
  from: string,
  to: string,
  targetBuckets = 50,
): Interval {
  const durationInSeconds = dayjs(to).diff(dayjs(from), "second");
  if (durationInSeconds <= 0) {
    return INTERVALS[0];
  }
  const targetSeconds = durationInSeconds / targetBuckets;
  return (
    INTERVALS.find((i) => i.seconds >= targetSeconds) ??
    INTERVALS[INTERVALS.length - 1]
  );
}

/** DuckDB interval literal, e.g. "15 minute". DuckDB accepts singular units. */
export function intervalToSql(interval: Interval): string {
  return `${interval.value} ${interval.unit}`;
}

/** End of the bucket that starts at `bucketStartIso`. */
export function bucketEnd(
  bucketStartIso: string,
  interval: Interval,
): dayjs.Dayjs {
  return dayjs(bucketStartIso).add(interval.value, interval.unit);
}
