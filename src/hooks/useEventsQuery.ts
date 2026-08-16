import useSWR, { type SWRResponse } from "swr";
import { useTimeRange } from "./useUrlParams";
import { useRefresh } from "../contexts/RefreshContext";
import {
  postQuery,
  type QueryResponse,
  type QueryMeta,
} from "../lib/api/queryClient";
import { recordQueryMeta } from "../stores/queryMetaStore";

export type { QueryMeta, QueryResponse };

export interface EventsQueryOptions {
  /** Override the start of the scanned range (ISO). Defaults to the global time range. */
  from?: string;
  /** Override the end of the scanned range (ISO). Defaults to the global time range. */
  to?: string;
  /** Cache scope tag; also keys the scan-cost recording. Defaults to "default". */
  scope?: string;
}

type EventsKey = readonly ["events", string, string, string, string, number];

const fetchPage = async <T>(key: EventsKey): Promise<QueryResponse<T>> => {
  const [, scope, query, from, to] = key;
  const response = await postQuery<T>(query, from, to);
  recordQueryMeta(scope, query, response.meta);
  return response;
};

const useEventsKey = (
  query: string | null,
  opts?: EventsQueryOptions,
): EventsKey | null => {
  const { from: globalFrom, to: globalTo } = useTimeRange();
  const { refreshKey } = useRefresh();

  if (!query) return null;
  return [
    "events",
    opts?.scope ?? "default",
    query,
    opts?.from ?? globalFrom,
    opts?.to ?? globalTo,
    refreshKey,
  ] as const;
};

/**
 * Query the events API within the global (or overridden) time range.
 *
 * @param query SQL to execute against `$events`. Pass null to skip fetching.
 *
 * @example
 * ```tsx
 * const { data } = useEventsQuery<{ reason: string; count: number }>(
 *   "SELECT reason, COUNT(*) as count FROM $events GROUP BY reason",
 * );
 * ```
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export const useEventsQuery = <T extends Record<string, any>>(
  query: string | null,
  opts?: EventsQueryOptions,
): SWRResponse<T[], Error> => {
  const key = useEventsKey(query, opts);
  return useSWR<T[], Error>(
    key,
    async (k: EventsKey) => (await fetchPage<T>(k)).results,
  );
};

/**
 * Same as useEventsQuery, but returns the full response including scan-cost
 * metadata (duration_ms, files, total_files_size_bytes).
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export const useEventsQueryMeta = <T extends Record<string, any>>(
  query: string | null,
  opts?: EventsQueryOptions,
): SWRResponse<QueryResponse<T>, Error> => {
  const key = useEventsKey(query, opts);
  return useSWR<QueryResponse<T>, Error>(key, (k: EventsKey) =>
    fetchPage<T>(k),
  );
};
