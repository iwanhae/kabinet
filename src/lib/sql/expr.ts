/**
 * Canonical event timestamp expression.
 *
 * `lastTimestamp` can be null for events.k8s.io/v1 events that only set
 * `eventTime` — every time-based query (sorting, bucketing, keyset cursors)
 * must use this expression or those rows silently vanish.
 */
export const TS_EXPR =
  "COALESCE(lastTimestamp, eventTime, metadata.creationTimestamp)";
