import { useMemo } from "react";
import { useSearch } from "wouter";
import { useUrlParams } from "./useUrlParams";
import {
  newChipId,
  type FilterChip,
  type FilterState,
} from "../lib/filters/model";
import { compileFilters } from "../lib/filters/compile";
import { decodeFilters, encodeFilters } from "../lib/filters/urlCodec";

/**
 * Filter state derived from the URL (source of truth).
 * `filters` (JSON chips) and `where` (raw SQL) are mutually exclusive;
 * a legacy `where`-only URL opens in raw mode.
 */
export function useFilters() {
  const search = useSearch();
  const { updateParams } = useUrlParams();

  const state: FilterState = useMemo(() => {
    const params = new URLSearchParams(search);
    const filtersParam = params.get("filters");
    const whereParam = params.get("where");
    if (filtersParam) {
      return {
        mode: "chips",
        chips: decodeFilters(filtersParam),
        rawWhere: "",
      };
    }
    if (whereParam) {
      return { mode: "raw", chips: [], rawWhere: whereParam };
    }
    return { mode: "chips", chips: [], rawWhere: "" };
  }, [search]);

  const whereSql = useMemo(() => compileFilters(state), [state]);

  const writeChips = (chips: FilterChip[]) => {
    updateParams({
      filters: chips.length > 0 ? encodeFilters(chips) : undefined,
      where: undefined,
    });
  };

  const addChip = (chip: Omit<FilterChip, "id">) => {
    const chips = state.mode === "chips" ? state.chips : [];
    writeChips([...chips, { ...chip, id: newChipId() }]);
  };

  const removeChip = (id: string) => {
    writeChips(state.chips.filter((c) => c.id !== id));
  };

  const updateChip = (id: string, patch: Partial<Omit<FilterChip, "id">>) => {
    writeChips(state.chips.map((c) => (c.id === id ? { ...c, ...patch } : c)));
  };

  const setRawMode = (rawWhere: string) => {
    updateParams({ where: rawWhere, filters: undefined });
  };

  const setChipsMode = (chips: FilterChip[]) => {
    writeChips(chips);
  };

  const clear = () => {
    updateParams({ filters: undefined, where: undefined });
  };

  /**
   * Drill-down used by aggregate clicks (KPI cards, heatmap cells, dimension
   * rows): merges the new chips into the current global filters — existing
   * chips on the same field are replaced, everything else is kept — and
   * navigates to Explore. Raw mode is discarded (SQL can't be merged).
   */
  const drill = (
    chips: Array<Omit<FilterChip, "id">>,
    extraParams?: { from?: string; to?: string },
  ) => {
    const newFields = new Set(chips.map((c) => c.field));
    const kept =
      state.mode === "chips"
        ? state.chips.filter((c) => !newFields.has(c.field))
        : [];
    const merged = [...kept, ...chips.map((c) => ({ ...c, id: newChipId() }))];
    updateParams(
      {
        filters: encodeFilters(merged),
        where: undefined,
        ...extraParams,
      },
      "/p/discover",
    );
  };

  return {
    state,
    whereSql,
    addChip,
    removeChip,
    updateChip,
    setRawMode,
    setChipsMode,
    clear,
    drill,
  };
}
