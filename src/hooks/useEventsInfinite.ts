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
  type SortSpec,
} from "../lib/sql/pagination";
import type { EventResult } from "../types/events";

export const PAGE_SIZE = 100;

interface EventsPage {
  rows: EventResult[];
  nextCursor: PageCursor | null;
  meta: QueryMeta;
}

type PageKey = readonly [
  "events/page",
  string, // whereSql
  string, // sort
  string, // from
  string, // to
  number, // refreshKey
  string, // cursor JSON ("" for page 1)
];

/**
 * Keyset-paginated infinite scroll over $events.
 *
 * Pre-compaction WAL duplicates (same uid, different resourceVersion) are
 * deduplicated client-side, keeping the highest resourceVersion — so a page
 * can contribute fewer than PAGE_SIZE visible rows. End-of-data is judged on
 * the raw page length.
 */
export function useEventsInfinite(whereSql: string, sort: SortSpec) {
  const { from, to } = useTimeRange();
  const { refreshKey } = useRefresh();
  const sortStr = `${sort.key}:${sort.dir}`;

  const getKey = useCallback(
    (_index: number, prev: EventsPage | null): PageKey | null => {
      if (prev && prev.rows.length < PAGE_SIZE) return null;
      const cursor = prev?.nextCursor;
      return [
        "events/page",
        whereSql,
        sortStr,
        from,
        to,
        refreshKey,
        cursor ? JSON.stringify(cursor) : "",
      ] as const;
    },
    [whereSql, sortStr, from, to, refreshKey],
  );

  const fetcher = useCallback(async (key: PageKey): Promise<EventsPage> => {
    const [, where, sortKey, fromIso, toIso, , cursorJson] = key;
    const [k, dir] = sortKey.split(":") as [SortSpec["key"], SortSpec["dir"]];
    const cursor: PageCursor | null = cursorJson
      ? JSON.parse(cursorJson)
      : null;
    const sql = buildPageQuery(where, { key: k, dir }, cursor, PAGE_SIZE);
    const response = await postQuery<EventResult>(sql, fromIso, toIso);
    recordQueryMeta("explore", sql, response.meta);
    const rows = response.results;
    return {
      rows,
      nextCursor:
        rows.length > 0
          ? cursorFromRow(rows[rows.length - 1], { key: k, dir })
          : null,
      meta: response.meta,
    };
  }, []);

  const { data, error, size, setSize, isValidating, isLoading } =
    useSWRInfinite<EventsPage, Error>(getKey, fetcher, {
      revalidateFirstPage: false,
      revalidateAll: false,
      revalidateOnFocus: false,
    });

  const events = useMemo(() => {
    const byUid = new Map<string, number>();
    const out: EventResult[] = [];
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
  const isReachingEnd = Boolean(lastPage && lastPage.rows.length < PAGE_SIZE);
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
