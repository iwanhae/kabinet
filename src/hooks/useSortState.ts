import { useMemo } from "react";
import { useSearch } from "wouter";
import { useUrlParams } from "./useUrlParams";
import type { SortKey, SortSpec } from "../lib/sql/pagination";

const SORT_KEYS: SortKey[] = ["ts", "count", "namespace", "reason"];
const DEFAULT_SORT: SortSpec = { key: "ts", dir: "desc" };

export function useSortState() {
  const search = useSearch();
  const { updateParams } = useUrlParams();

  const sort: SortSpec = useMemo(() => {
    const raw = new URLSearchParams(search).get("sort");
    if (!raw) return DEFAULT_SORT;
    const [key, dir] = raw.split(":");
    if (!SORT_KEYS.includes(key as SortKey)) return DEFAULT_SORT;
    return { key: key as SortKey, dir: dir === "asc" ? "asc" : "desc" };
  }, [search]);

  const setSort = (next: SortSpec) => {
    const isDefault =
      next.key === DEFAULT_SORT.key && next.dir === DEFAULT_SORT.dir;
    updateParams({ sort: isDefault ? undefined : `${next.key}:${next.dir}` });
  };

  /** Header-click behavior: toggle direction on the active column. */
  const toggleSort = (key: SortKey) => {
    if (sort.key === key) {
      setSort({ key, dir: sort.dir === "desc" ? "asc" : "desc" });
    } else {
      setSort({ key, dir: "desc" });
    }
  };

  return { sort, setSort, toggleSort };
}
