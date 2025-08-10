import { useLocation, useSearch } from "wouter";
import { useMemo } from "react";
import { formatTimeRange } from "../utils/timeRange";
import { useRefresh } from "../contexts/RefreshContext";

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

  const updateParams = (
    newParams: Partial<UrlParams>,
    path = "/p/discover",
  ) => {
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
      setLocation("/p/discover");
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
  const search = useSearch(); // URL 쿼리 변경 감지
  const { refreshKey, triggerRefresh } = useRefresh(); // 전역 새로고침 상태

  // search 문자열이나 refreshKey가 바뀔 때마다 시간 범위를 새로 계산
  const { from, to, rawFrom, rawTo } = useMemo(() => {
    const searchParams = new URLSearchParams(search);
    const fromParam = searchParams.get("from") || "now-30m";
    const toParam = searchParams.get("to") || "now";

    return formatTimeRange(fromParam, toParam);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [search, refreshKey]);

  const setTimeRange = (from: string, to: string) => {
    updateParams({ from, to }, window.location.pathname);
  };

  return {
    from,
    to,
    rawFrom,
    rawTo,
    setTimeRange,
    refreshTimeRange: triggerRefresh, // Context의 함수를 그대로 반환
  };
};
