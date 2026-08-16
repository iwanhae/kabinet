import type { FilterChip, FilterField, FilterOperator } from "./model";
import { newChipId } from "./model";
import { FIELD_DEFS } from "./fields";

type ChipTuple = [string, string, string[]];

/** [["namespace","eq",["kube-system"]], ...] */
export const encodeFilters = (chips: Array<Omit<FilterChip, "id">>): string =>
  JSON.stringify(chips.map((c) => [c.field, c.op, c.values]));

/** Parses the `filters` URL param, dropping anything that fails validation. */
export const decodeFilters = (raw: string): FilterChip[] => {
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return [];
  }
  if (!Array.isArray(parsed)) return [];

  const chips: FilterChip[] = [];
  for (const item of parsed as ChipTuple[]) {
    if (!Array.isArray(item) || item.length !== 3) continue;
    const [field, op, values] = item;
    const def = FIELD_DEFS[field as FilterField];
    if (!def) continue;
    if (!def.ops.includes(op as FilterOperator)) continue;
    if (!Array.isArray(values) || !values.every((v) => typeof v === "string"))
      continue;
    if (values.length === 0) continue;
    chips.push({
      id: newChipId(),
      field: field as FilterField,
      op: op as FilterOperator,
      values,
    });
  }
  return chips;
};
