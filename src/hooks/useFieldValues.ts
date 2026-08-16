import { useEventsQuery } from "./useEventsQuery";
import { FIELD_DEFS } from "../lib/filters/fields";
import type { FilterField } from "../lib/filters/model";

/**
 * Autocomplete values for a filter field within the current time range.
 * Enum fields answer locally; distinct fields run a DISTINCT scan.
 */
export function useFieldValues(field: FilterField, enabled: boolean) {
  const def = FIELD_DEFS[field];
  const query =
    enabled && def.suggest === "distinct"
      ? `SELECT DISTINCT ${def.sqlExpr} AS value
         FROM $events
         WHERE ${def.sqlExpr} IS NOT NULL
         ORDER BY value
         LIMIT 100`
      : null;

  const { data, isLoading } = useEventsQuery<{ value: string }>(query, {
    scope: "suggest",
  });

  if (def.suggest === "enum") {
    return { values: def.enumValues ?? [], isLoading: false };
  }
  return {
    values: (data ?? []).map((d) => d.value),
    isLoading: def.suggest === "distinct" ? isLoading : false,
  };
}
