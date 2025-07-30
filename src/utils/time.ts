import dayjs from "dayjs";

const intervals: { label: string; seconds: number }[] = [
  { label: "10 second", seconds: 10 },
  { label: "15 second", seconds: 15 },
  { label: "30 second", seconds: 30 },
  { label: "1 minute", seconds: 60 },
  { label: "5 minute", seconds: 300 },
  { label: "15 minute", seconds: 900 },
  { label: "30 minute", seconds: 1800 },
  { label: "1 hour", seconds: 3600 },
  { label: "3 hour", seconds: 10800 },
  { label: "6 hour", seconds: 21600 },
  { label: "12 hour", seconds: 43200 },
  { label: "1 day", seconds: 86400 },
];

/**
 * Calculates the optimal time interval for a chart to ensure a minimum number of data points.
 * @param from - The start of the time range (ISO string).
 * @param to - The end of the time range (ISO string).
 * @param minDataPoints - The minimum number of data points to aim for.
 * @returns A string representing the interval for DuckDB (e.g., '15 second').
 */
export function getDynamicInterval(
  from: string,
  to: string,
  minDataPoints = 20,
): string {
  const fromDate = dayjs(from);
  const toDate = dayjs(to);
  const durationInSeconds = toDate.diff(fromDate, "second");

  if (durationInSeconds <= 0) {
    return intervals[0].label;
  }

  const targetIntervalSeconds = durationInSeconds / minDataPoints;

  const bestIntervalIdx = intervals.findIndex(
    (i) => i.seconds >= targetIntervalSeconds,
  );
  const bestInterval = intervals[bestIntervalIdx - 1];

  return bestInterval
    ? bestInterval.label + "s"
    : intervals[intervals.length - 1].label;
}
