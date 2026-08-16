import { useSearch } from "wouter";

export interface NavigationOptions {
  page:
    | ""
    | "insight"
    | "discover"
    | "namespaces"
    | "nodes"
    | "components"
    | "mcp";
  params?: {
    where?: string;
    from?: string;
    to?: string;
    [key: string]: string | undefined;
  };
}

/**
 * Builds hrefs that carry the global state — time range (from/to) and
 * filters (filters/where) — across pages.
 */
export const useNavigation = () => {
  const search = useSearch();
  const searchParams = new URLSearchParams(search);
  const fromParam = searchParams.get("from") || "now-30m";
  const toParam = searchParams.get("to") || "now";
  const filtersParam = searchParams.get("filters");
  const whereParam = searchParams.get("where");

  return (options: NavigationOptions): string => {
    if (options.page === "insight") options.page = "";
    const base = options.page ? `/p/${options.page}` : "/";
    const params: Record<string, string> = {
      from: fromParam,
      to: toParam,
    };
    if (filtersParam) params.filters = filtersParam;
    if (whereParam) params.where = whereParam;
    Object.entries(options.params ?? {}).forEach(([key, value]) => {
      if (value !== undefined) params[key] = value;
    });
    return `${base}?${new URLSearchParams(params).toString()}`;
  };
};
