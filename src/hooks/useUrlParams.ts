import { useLocation } from "wouter";

export interface UrlParams {
  from?: string;
  to?: string;
  where?: string;
  query?: string;
}

/**
 * URL 파라미터를 중앙화해서 관리하는 훅
 * 기존 파라미터를 유지하면서 새로운 파라미터를 추가/수정
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

    // undefined 값들은 제거
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
