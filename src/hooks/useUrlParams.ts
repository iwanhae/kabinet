import { useLocation } from "wouter";
import {
  formatTimeRange,
  refreshRelativeTimeRange,
  type ParsedTimeRange,
} from "../utils/timeRange";

export interface UrlParams {
  from?: string;
  to?: string;
  where?: string;
  query?: string;
}

/**
 * A hook for centralized management of URL parameters
 * Adds or updates new parameters while retaining existing ones
 */
export const useUrlParams = () => {
  const [, setLocation] = useLocation();

  const getCurrentParams = (): UrlParams => {
    const searchParams = new URLSearchParams(window.location.search);
    return {
      from: searchParams.get("from") || undefined,
      to: searchParams.get("to") || undefined,
      where: searchParams.get("where") || undefined,
      query: searchParams.get("query") || undefined,
    };
  };

  const updateParams = (newParams: Partial<UrlParams>, path = "/discover") => {
    const currentParams = getCurrentParams();
    const mergedParams = { ...currentParams, ...newParams };

    const searchParams = new URLSearchParams();
    Object.entries(mergedParams).forEach(([key, value]) => {
      if (value !== undefined && value !== null && value !== "") {
        searchParams.set(key, value);
      }
    });

    const queryString = searchParams.toString();
    const newUrl = queryString ? `${path}?${queryString}` : path;
    setLocation(newUrl);
  };

  const clearParams = (keysToKeep?: string[]) => {
    if (!keysToKeep) {
      setLocation("/discover");
      return;
    }

    const currentParams = getCurrentParams();
    const filteredParams: UrlParams = {};

    keysToKeep.forEach((key) => {
      if (key in currentParams) {
        filteredParams[key as keyof UrlParams] =
          currentParams[key as keyof UrlParams];
      }
    });

    updateParams(filteredParams);
  };

  return {
    getCurrentParams,
    updateParams,
    clearParams,
  };
};

// 편의성을 위한 개별 훅들
export const useTimeRangeParams = () => {
  const { updateParams, getCurrentParams } = useUrlParams();

  const setTimeRange = (from: string, to: string) => {
    updateParams({ from, to }, window.location.pathname);
  };

  const getTimeRange = () => {
    const params = getCurrentParams();
    return { from: params.from, to: params.to };
  };

  return { setTimeRange, getTimeRange };
};

export const useQueryParams = () => {
  const { updateParams, getCurrentParams } = useUrlParams();

  const setWhereClause = (where: string) => {
    updateParams({ where });
  };

  const setQuery = (query: string) => {
    updateParams({ query });
  };

  const getQuery = () => {
    const params = getCurrentParams();
    return { where: params.where, query: params.query };
  };

  return { setWhereClause, setQuery, getQuery };
};

export const useTimeRange = () => {
  const { updateParams } = useUrlParams();

  const getCurrentTimeRange = (): ParsedTimeRange => {
    const searchParams = new URLSearchParams(window.location.search);
    const from = searchParams.get("from") || "now-30m";
    const to = searchParams.get("to") || "now";

    return formatTimeRange(from, to);
  };

  const setTimeRange = (from: string, to: string) => {
    updateParams({ from, to }, window.location.pathname);
  };

  const refreshTimeRange = () => {
    const current = getCurrentTimeRange();
    const refreshed = refreshRelativeTimeRange(current.rawFrom, current.rawTo);

    if (refreshed.from !== current.from || refreshed.to !== current.to) {
      setTimeRange(refreshed.rawFrom, refreshed.rawTo);
    }
  };

  return {
    ...getCurrentTimeRange(),
    setTimeRange,
    refreshTimeRange,
  };
};
