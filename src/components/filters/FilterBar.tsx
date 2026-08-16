import React, { useEffect, useRef, useState } from "react";
import { Code2, Download, ListFilter, Plus, X } from "lucide-react";
import { Button, Chip, TextArea } from "../../ui";
import { useFilters } from "../../hooks/useFilters";
import { useTimeRange } from "../../hooks/useUrlParams";
import { FIELD_DEFS } from "../../lib/filters/fields";
import { OPERATOR_LABELS, type FilterChip } from "../../lib/filters/model";
import { compileFilters } from "../../lib/filters/compile";
import FilterChipEditor from "./FilterChipEditor";
import styles from "./FilterBar.module.css";

const chipLabel = (chip: FilterChip): string => {
  const def = FIELD_DEFS[chip.field];
  const value =
    chip.values.length > 1 ? `(${chip.values.join(", ")})` : chip.values[0];
  return `${def.label} ${OPERATOR_LABELS[chip.op]} ${value}`;
};

/**
 * Global filter bar. Filters live in URL params and apply to every data tab
 * (Overview, Namespaces, Nodes, Components, Explore) — like the time range.
 */
const FilterBar: React.FC = () => {
  const {
    state,
    whereSql,
    addChip,
    updateChip,
    removeChip,
    setRawMode,
    setChipsMode,
    clear,
  } = useFilters();
  const { from, to } = useTimeRange();

  const addButtonRef = useRef<HTMLButtonElement>(null);
  const [editorOpen, setEditorOpen] = useState(false);
  const [editing, setEditing] = useState<FilterChip | null>(null);
  const [editAnchor, setEditAnchor] = useState<HTMLElement | null>(null);
  const [rawDraft, setRawDraft] = useState(state.rawWhere);

  // The bar persists across navigation — keep the draft in sync when the
  // URL changes underneath it.
  useEffect(() => {
    setRawDraft(state.rawWhere);
  }, [state.rawWhere]);

  const downloadHref = `/download?${new URLSearchParams({
    where: whereSql,
    from,
    to,
  }).toString()}`;

  const openAdd = () => {
    setEditing(null);
    setEditAnchor(addButtonRef.current);
    setEditorOpen(true);
  };

  const openEdit = (chip: FilterChip, anchor: HTMLElement) => {
    setEditing(chip);
    setEditAnchor(anchor);
    setEditorOpen(true);
  };

  const switchToRaw = () => {
    setRawMode(compileFilters(state));
  };

  const switchToChips = () => {
    // Raw SQL cannot be parsed back into chips; confirm before discarding.
    if (
      state.rawWhere.trim() &&
      state.rawWhere.trim() !== "1=1" &&
      !window.confirm(
        "Discard the raw WHERE clause and start with empty filters?",
      )
    ) {
      return;
    }
    setChipsMode([]);
  };

  if (state.mode === "raw") {
    return (
      <div className={styles.bar}>
        <form
          className={styles.rawForm}
          onSubmit={(e) => {
            e.preventDefault();
            setRawMode(rawDraft.trim() || "1=1");
          }}
        >
          <span className={styles.rawPrefix}>
            … FROM $events WHERE (applies to every tab)
          </span>
          <TextArea
            mono
            rows={2}
            value={rawDraft}
            onChange={(e) => setRawDraft(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) {
                setRawMode(rawDraft.trim() || "1=1");
              }
            }}
            placeholder="type = 'Warning' AND metadata.namespace = 'kube-system'"
          />
          <div className={styles.rawActions}>
            <Button variant="ghost" size="sm" onClick={switchToChips}>
              <ListFilter size={14} />
              Switch to filters
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => window.open(downloadHref, "_blank", "noopener")}
            >
              <Download size={14} />
              Download
            </Button>
            <Button variant="solid" size="sm" type="submit">
              Run query
            </Button>
          </div>
        </form>
      </div>
    );
  }

  return (
    <div className={styles.bar}>
      {state.chips.map((chip) => (
        <Chip
          key={chip.id}
          onClick={(e) => openEdit(chip, e.currentTarget)}
          onRemove={() => removeChip(chip.id)}
          title={chipLabel(chip)}
        >
          {chipLabel(chip)}
        </Chip>
      ))}

      <Button ref={addButtonRef} variant="outline" size="sm" onClick={openAdd}>
        <Plus size={14} />
        Add filter
      </Button>

      {state.chips.length > 0 && (
        <Button variant="ghost" size="sm" onClick={clear}>
          <X size={14} />
          Clear
        </Button>
      )}

      <div className={styles.spacer} />

      <Button variant="ghost" size="sm" onClick={switchToRaw}>
        <Code2 size={14} />
        SQL
      </Button>
      <Button
        variant="outline"
        size="sm"
        onClick={() => window.open(downloadHref, "_blank", "noopener")}
      >
        <Download size={14} />
        Download
      </Button>

      <FilterChipEditor
        open={editorOpen}
        anchorEl={editAnchor}
        onClose={() => setEditorOpen(false)}
        initial={editing ?? undefined}
        onSubmit={(chip) => {
          if (editing) updateChip(editing.id, chip);
          else addChip(chip);
        }}
      />
    </div>
  );
};

export default FilterBar;
