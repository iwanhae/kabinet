import { useSearch } from "wouter";

export interface NavigationOptions {
  page: "" | "insight" | "discover";
  params?: {
    where?: string;
    from?: string;
    to?: string;
    [key: string]: string | undefined;
  };
}

export const useNavigation = () => {
  const search = useSearch();
  const searchParams = new URLSearchParams(search);
  const fromParam = searchParams.get("from") || "now-30m";
  const toParam = searchParams.get("to") || "now";

  return (options: NavigationOptions): string => {
    if (options.page === "insight") options.page = "";
    const href = `/${options.page}?${new URLSearchParams({
      from: fromParam,
      to: toParam,
      ...options.params, // params will override the search params
    }).toString()}`;
    return href;
  };
};
