import type { FilterField, FilterOperator } from "./model";

export interface FieldDef {
  key: FilterField;
  label: string;
  /** SQL expression — whitelisted here, NEVER user input. */
  sqlExpr: string;
  ops: FilterOperator[];
  /** Autocomplete strategy for the value editor. */
  suggest: "distinct" | "enum" | "none";
  enumValues?: string[];
}

export const FIELD_DEFS: Record<FilterField, FieldDef> = {
  namespace: {
    key: "namespace",
    label: "Namespace",
    sqlExpr: "metadata.namespace",
    ops: ["eq", "neq", "in", "notIn"],
    suggest: "distinct",
  },
  type: {
    key: "type",
    label: "Type",
    sqlExpr: "type",
    ops: ["eq", "neq"],
    suggest: "enum",
    enumValues: ["Normal", "Warning"],
  },
  reason: {
    key: "reason",
    label: "Reason",
    sqlExpr: "reason",
    ops: ["eq", "neq", "in", "notIn"],
    suggest: "distinct",
  },
  kind: {
    key: "kind",
    label: "Kind",
    sqlExpr: "involvedObject.kind",
    ops: ["eq", "neq", "in"],
    suggest: "distinct",
  },
  objectName: {
    key: "objectName",
    label: "Object",
    sqlExpr: "involvedObject.name",
    ops: ["eq", "neq", "contains"],
    suggest: "none",
  },
  message: {
    key: "message",
    label: "Message",
    sqlExpr: "message",
    ops: ["contains", "notContains"],
    suggest: "none",
  },
  component: {
    key: "component",
    label: "Component",
    // Controller-emitted events (events.k8s.io) only set reportingComponent.
    sqlExpr: "COALESCE(source.component, reportingComponent)",
    ops: ["eq", "neq", "in"],
    suggest: "distinct",
  },
  host: {
    key: "host",
    label: "Host",
    sqlExpr: "source.host",
    ops: ["eq", "neq", "contains"],
    suggest: "distinct",
  },
};

export const ALL_FIELDS = Object.values(FIELD_DEFS);
