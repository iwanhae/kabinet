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

  return {
    state,
    whereSql,
    addChip,
    removeChip,
    updateChip,
    setRawMode,
    setChipsMode,
    clear,
  };
}
