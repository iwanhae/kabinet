import React, { useEffect, useMemo, useState } from "react";
import { Button, Popover, Select, TextInput, Spinner } from "../../ui";
import { ALL_FIELDS, FIELD_DEFS } from "../../lib/filters/fields";
import {
  OPERATOR_LABELS,
  type FilterChip,
  type FilterField,
  type FilterOperator,
} from "../../lib/filters/model";
import { useFieldValues } from "../../hooks/useFieldValues";
import styles from "./FilterChipEditor.module.css";

export interface FilterChipEditorProps {
  open: boolean;
  anchorEl: HTMLElement | null;
  onClose: () => void;
  /** Present when editing an existing chip. */
  initial?: FilterChip;
  onSubmit: (chip: Omit<FilterChip, "id">) => void;
}

const isMulti = (op: FilterOperator) => op === "in" || op === "notIn";

const FilterChipEditor: React.FC<FilterChipEditorProps> = ({
  open,
  anchorEl,
  onClose,
  initial,
  onSubmit,
}) => {
  const [field, setField] = useState<FilterField>(
    initial?.field ?? "namespace",
  );
  const [op, setOp] = useState<FilterOperator>(initial?.op ?? "eq");
  const [valueText, setValueText] = useState(initial?.values.join(", ") ?? "");

  useEffect(() => {
    if (!open) return;
    setField(initial?.field ?? "namespace");
    setOp(initial?.op ?? "eq");
    setValueText(initial?.values.join(", ") ?? "");
  }, [open, initial]);

  const def = FIELD_DEFS[field];
  const { values: suggestions, isLoading } = useFieldValues(
    field,
    open && def.suggest !== "none",
  );

  const values = useMemo(
    () =>
      isMulti(op)
        ? valueText
            .split(",")
            .map((v) => v.trim())
            .filter(Boolean)
        : valueText.trim()
          ? [valueText.trim()]
          : [],
    [op, valueText],
  );

  const filteredSuggestions = useMemo(() => {
    const needle = (
      isMulti(op) ? (valueText.split(",").pop() ?? "") : valueText
    )
      .trim()
      .toLowerCase();
    return suggestions
      .filter((s) => !needle || s.toLowerCase().includes(needle))
      .slice(0, 30);
  }, [suggestions, valueText, op]);

  const pickSuggestion = (value: string) => {
    if (isMulti(op)) {
      const parts = valueText.split(",").map((v) => v.trim());
      parts[parts.length - 1] = value;
      setValueText(parts.filter(Boolean).join(", ") + ", ");
    } else {
      setValueText(value);
    }
  };

  const submit = () => {
    if (values.length === 0) return;
    onSubmit({ field, op, values });
    onClose();
  };

  return (
    <Popover open={open} anchorEl={anchorEl} onClose={onClose}>
      <div className={styles.editor}>
        <div className={styles.row}>
          <Select
            label="Field"
            value={field}
            onChange={(e) => {
              const next = e.target.value as FilterField;
              setField(next);
              if (!FIELD_DEFS[next].ops.includes(op)) {
                setOp(FIELD_DEFS[next].ops[0]);
              }
              setValueText("");
            }}
          >
            {ALL_FIELDS.map((f) => (
              <option key={f.key} value={f.key}>
                {f.label}
              </option>
            ))}
          </Select>
          <Select
            label="Operator"
            value={op}
            onChange={(e) => setOp(e.target.value as FilterOperator)}
          >
            {def.ops.map((o) => (
              <option key={o} value={o}>
                {OPERATOR_LABELS[o]}
              </option>
            ))}
          </Select>
        </div>

        <TextInput
          label={isMulti(op) ? "Values (comma separated)" : "Value"}
          mono
          value={valueText}
          onChange={(e) => setValueText(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") submit();
          }}
          placeholder={def.suggest === "enum" ? "Normal / Warning" : ""}
          autoFocus
        />

        {def.suggest !== "none" && (
          <div className={styles.suggestions}>
            {isLoading ? (
              <div style={{ padding: 8, textAlign: "center" }}>
                <Spinner size={16} />
              </div>
            ) : (
              filteredSuggestions.map((s) => (
                <button
                  key={s}
                  type="button"
                  className={styles.suggestion}
                  onClick={() => pickSuggestion(s)}
                >
                  {s}
                </button>
              ))
            )}
          </div>
        )}

        <div className={styles.footer}>
          <Button variant="ghost" size="sm" onClick={onClose}>
            Cancel
          </Button>
          <Button
            variant="solid"
            size="sm"
            disabled={values.length === 0}
            onClick={submit}
          >
            {initial ? "Update filter" : "Add filter"}
          </Button>
        </div>
      </div>
    </Popover>
  );
};

export default FilterChipEditor;
