import { useSearch } from "wouter";

export interface NavigationOptions {
  page: "" | "insight" | "discover" | "namespaces" | "agent";
  params?: {
    where?: string;
    from?: string;
    to?: string;
    [key: string]: string | undefined;
  };
}

/**
 * Builds hrefs that carry the current time range (from/to) across pages.
 */
export const useNavigation = () => {
  const search = useSearch();
  const searchParams = new URLSearchParams(search);
  const fromParam = searchParams.get("from") || "now-30m";
  const toParam = searchParams.get("to") || "now";

  return (options: NavigationOptions): string => {
    if (options.page === "insight") options.page = "";
    const base =
      options.page === "agent"
        ? "/agent"
        : options.page
          ? `/p/${options.page}`
          : "/";
    const params: Record<string, string> = {
      from: fromParam,
      to: toParam,
    };
    Object.entries(options.params ?? {}).forEach(([key, value]) => {
      if (value !== undefined) params[key] = value;
    });
    return `${base}?${new URLSearchParams(params).toString()}`;
  };
};
