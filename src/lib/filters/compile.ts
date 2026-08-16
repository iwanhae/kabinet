import type { FilterChip, FilterState } from "./model";
import { FIELD_DEFS } from "./fields";

/** DuckDB string-literal escaping: double the single quotes. */
export const escapeSqlString = (v: string): string => v.replace(/'/g, "''");

/**
 * For ILIKE patterns: additionally neutralize %, _ and the escape char.
 * Must be paired with an explicit ESCAPE '\' clause — DuckDB LIKE has no
 * default escape character.
 */
export const escapeLikeValue = (v: string): string =>
  escapeSqlString(v).replace(/([\\%_])/g, "\\$1");

const quoted = (v: string) => `'${escapeSqlString(v)}'`;

export function compileChip(chip: FilterChip): string {
  const expr = FIELD_DEFS[chip.field].sqlExpr;
  switch (chip.op) {
    case "eq":
      return `${expr} = ${quoted(chip.values[0] ?? "")}`;
    case "neq":
      return `${expr} != ${quoted(chip.values[0] ?? "")}`;
    case "in":
      return `${expr} IN (${chip.values.map(quoted).join(", ")})`;
    case "notIn":
      return `${expr} NOT IN (${chip.values.map(quoted).join(", ")})`;
    case "contains":
      return `${expr} ILIKE '%${escapeLikeValue(chip.values[0] ?? "")}%' ESCAPE '\\'`;
    case "notContains":
      return `${expr} NOT ILIKE '%${escapeLikeValue(chip.values[0] ?? "")}%' ESCAPE '\\'`;
  }
}

/** Compiles the filter state to a WHERE clause body ("1=1" when empty). */
export function compileFilters(state: FilterState): string {
  if (state.mode === "raw") return state.rawWhere.trim() || "1=1";
  if (state.chips.length === 0) return "1=1";
  return state.chips.map((c) => `(${compileChip(c)})`).join(" AND ");
}
