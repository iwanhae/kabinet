import { TS_EXPR } from "./expr";

// Monday 1970-01-12 UTC aligns with DuckDB's 2000-01-03 origin for every
// fixed-width interval we offer, including the two-week bucket. Keeping this
// close to the Unix epoch makes integer offsets positive for modern events.
const BUCKET_ORIGIN_SECONDS = 950_400;

/** Group on a number instead of converting every event to a timestamp. */
export const bucketNumberSql = (intervalSeconds: number): string =>
  `FLOOR((epoch_us(${TS_EXPR}) - ${BUCKET_ORIGIN_SECONDS * 1_000_000}) / ${intervalSeconds * 1_000_000})::BIGINT`;

/** Restore the timestamp only after GROUP BY has reduced the row count. */
export const bucketTimestampSql = (intervalSeconds: number): string =>
  `to_timestamp(bucket_number * ${intervalSeconds} + ${BUCKET_ORIGIN_SECONDS})`;
