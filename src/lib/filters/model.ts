export type FilterField =
  | "namespace"
  | "type"
  | "reason"
  | "kind"
  | "objectName"
  | "message"
  | "component"
  | "host";

export type FilterOperator =
  | "eq"
  | "neq"
  | "in"
  | "notIn"
  | "contains"
  | "notContains";

export interface FilterChip {
  /** React key only — never serialized. */
  id: string;
  field: FilterField;
  op: FilterOperator;
  /** eq/neq/contains use values[0]; in/notIn use all. */
  values: string[];
}

export type FilterMode = "chips" | "raw";

export interface FilterState {
  mode: FilterMode;
  chips: FilterChip[];
  /** User-authored WHERE clause, used when mode === "raw". */
  rawWhere: string;
}

export const newChipId = (): string =>
  typeof crypto !== "undefined" && "randomUUID" in crypto
    ? crypto.randomUUID()
    : Math.random().toString(36).slice(2);

export const OPERATOR_LABELS: Record<FilterOperator, string> = {
  eq: "=",
  neq: "≠",
  in: "in",
  notIn: "not in",
  contains: "contains",
  notContains: "not contains",
};
