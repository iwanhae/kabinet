import { useLocation, useSearch } from "wouter";
import { useMemo } from "react";
import { formatTimeRange } from "../utils/timeRange";
import { useRefresh } from "../contexts/RefreshContext";

export interface UrlParams {
  from?: string;
  to?: string;
  /** JSON-encoded filter chips (chips mode). */
  filters?: string;
  /** Raw WHERE clause (raw mode / legacy links). */
  where?: string;
  /** Sort spec, e.g. "ts:desc". */
  sort?: string;
  /** Selected event uid (detail panel). */
  uid?: string;
  query?: string;
  /** Legacy detail-link param, still honored. */
  resourceVersion?: string;
}

const PARAM_KEYS: (keyof UrlParams)[] = [
  "from",
  "to",
  "filters",
  "where",
  "sort",
  "uid",
  "query",
  "resourceVersion",
];

/**
 * Centralized URL-parameter management. The URL is the source of truth for
 * everything shareable: time range, filters, sort, selection.
 */
export const useUrlParams = () => {
  const [, setLocation] = useLocation();

  const getCurrentParams = (): UrlParams => {
    const searchParams = new URLSearchParams(window.location.search);
    const params: UrlParams = {};
    PARAM_KEYS.forEach((key) => {
      const value = searchParams.get(key);
      if (value) params[key] = value;
    });
    return params;
  };

  const updateParams = (
    newParams: Partial<UrlParams>,
    path: string = window.location.pathname,
  ) => {
    const mergedParams = { ...getCurrentParams(), ...newParams };

    const searchParams = new URLSearchParams();
    Object.entries(mergedParams).forEach(([key, value]) => {
      if (value !== undefined && value !== null && value !== "") {
        searchParams.set(key, value);
      }
    });

    const queryString = searchParams.toString();
    setLocation(queryString ? `${path}?${queryString}` : path);
  };

  return {
    getCurrentParams,
    updateParams,
  };
};

export const useQueryParams = () => {
  const { updateParams, getCurrentParams } = useUrlParams();

  const setWhereClause = (where: string) => {
    // Drill-downs from other pages land on Explore.
    updateParams({ where, filters: undefined }, "/p/discover");
  };

  const getQuery = () => {
    const params = getCurrentParams();
    return { where: params.where, query: params.query };
  };

  return { setWhereClause, getQuery };
};

export const useTimeRange = () => {
  const { updateParams } = useUrlParams();
  const search = useSearch();
  const { refreshKey, triggerRefresh } = useRefresh();

  // Recomputed whenever the URL or the manual refresh counter changes.
  const { from, to, rawFrom, rawTo } = useMemo(() => {
    const searchParams = new URLSearchParams(search);
    const fromParam = searchParams.get("from") || "now-30m";
    const toParam = searchParams.get("to") || "now";

    return formatTimeRange(fromParam, toParam);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [search, refreshKey]);

  const setTimeRange = (from: string, to: string) => {
    updateParams({ from, to });
  };

  return {
    from,
    to,
    rawFrom,
    rawTo,
    setTimeRange,
    refreshTimeRange: triggerRefresh,
  };
};
