import { useCallback, useMemo } from "react";
import useSWRInfinite from "swr/infinite";
import { useTimeRange } from "./useUrlParams";
import { useRefresh } from "../contexts/RefreshContext";
import { postQuery, type QueryMeta } from "../lib/api/queryClient";
import { recordQueryMeta } from "../stores/queryMetaStore";
import {
  buildPageQuery,
  cursorFromRow,
  type PageCursor,
} from "../lib/sql/pagination";
import type { EventListRow } from "../types/events";

export const PAGE_SIZE = 100;
const HOUR_MS = 60 * 60 * 1000;

interface PageState {
  windowEnd: string;
  /** Retained while paging within the same window. */
  windowStart?: string;
  cursor: PageCursor | null;
}

interface EventsPage {
  rows: EventListRow[];
  nextState: PageState | null;
  meta?: QueryMeta;
}

type PageKey = readonly [
  "events/page",
  string, // whereSql
  string, // from
  string, // to
  number, // refreshKey
  string, // PageState JSON
];

/**
 * Browse backwards in bounded time windows, never sorting the entire archive
 * to return a single page. Empty windows grow up to one day to skip gaps.
 */
export function useEventsInfinite(whereSql: string) {
  const { from, to } = useTimeRange();
  const { refreshKey } = useRefresh();

  const getKey = useCallback(
    (_index: number, prev: EventsPage | null): PageKey | null => {
      if (prev && !prev.nextState) return null;
      const state = prev?.nextState ?? { windowEnd: to, cursor: null };
      return [
        "events/page",
        whereSql,
        from,
        to,
        refreshKey,
        JSON.stringify(state),
      ] as const;
    },
    [whereSql, from, to, refreshKey],
  );

  const fetcher = useCallback(async (key: PageKey): Promise<EventsPage> => {
    const [, where, fromIso, , , stateJson] = key;
    const fromMs = Date.parse(fromIso);
    let { windowEnd, windowStart, cursor } = JSON.parse(stateJson) as PageState;
    let emptyWindows = 0;
    let meta: QueryMeta | undefined;

    while (Date.parse(windowEnd) > fromMs) {
      const start =
        windowStart ??
        new Date(
          Math.max(
            fromMs,
            Date.parse(windowEnd) - Math.min(2 ** emptyWindows, 24) * HOUR_MS,
          ),
        ).toISOString();
      const sql = buildPageQuery(where, windowEnd, cursor, PAGE_SIZE);
      const response = await postQuery<EventListRow>(sql, start, windowEnd);
      recordQueryMeta("explore", sql, response.meta);
      meta = response.meta;

      if (response.results.length > 0) {
        const nextState: PageState | null =
          response.results.length === PAGE_SIZE
            ? {
                windowEnd,
                windowStart: start,
                cursor: cursorFromRow(
                  response.results[response.results.length - 1],
                ),
              }
            : Date.parse(start) > fromMs
              ? { windowEnd: start, cursor: null }
              : null;
        return { rows: response.results, nextState, meta };
      }

      // A cursor exhausted this window; the next window starts at its boundary.
      windowEnd = start;
      windowStart = undefined;
      cursor = null;
      emptyWindows++;
    }

    return { rows: [], nextState: null, meta };
  }, []);

  const { data, error, size, setSize, isValidating, isLoading } =
    useSWRInfinite<EventsPage, Error>(getKey, fetcher, {
      revalidateFirstPage: false,
      revalidateAll: false,
      revalidateOnFocus: false,
    });

  const events = useMemo(() => {
    const byUid = new Map<string, number>();
    const out: EventListRow[] = [];
    (data ?? []).forEach((page) => {
      page.rows.forEach((row) => {
        const uid = row.metadata.uid;
        const existing = byUid.get(uid);
        if (existing === undefined) {
          byUid.set(uid, out.length);
          out.push(row);
        } else if (
          Number(row.metadata.resourceVersion) >
          Number(out[existing].metadata.resourceVersion)
        ) {
          out[existing] = row;
        }
      });
    });
    return out;
  }, [data]);

  const lastPage = data?.[data.length - 1];
  const isReachingEnd = Boolean(lastPage && !lastPage.nextState);
  const isLoadingMore =
    isValidating && size > 0 && data !== undefined && data.length < size;

  const loadMore = useCallback(() => {
    if (!isReachingEnd && !isValidating) void setSize((s) => s + 1);
  }, [isReachingEnd, isValidating, setSize]);

  return {
    events,
    loadMore,
    isLoadingInitial: isLoading,
    isLoadingMore,
    isReachingEnd,
    error: error ?? undefined,
    meta: lastPage?.meta,
  };
}
