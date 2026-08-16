import React from "react";
import { Accordion, Chip, Drawer, Spinner, cx } from "../../../ui";
import type { EventResult } from "../../../types/events";
import type { FilterField } from "../../../lib/filters/model";
import { DETAIL_SECTIONS, type DetailRow } from "./detailSections";
import styles from "./EventDetailPanel.module.css";

export interface EventDetailPanelProps {
  open: boolean;
  event: EventResult | null;
  onClose: () => void;
  /** Adds an equality filter for the clicked value. */
  onFilter: (field: FilterField, value: string) => void;
}

const Value: React.FC<{
  row: DetailRow;
  event: EventResult;
  onFilter: EventDetailPanelProps["onFilter"];
}> = ({ row, event, onFilter }) => {
  const raw = row.get(event);
  if (raw === undefined || raw === null || raw === "") {
    return <span style={{ color: "var(--ink-faint)" }}>—</span>;
  }
  const text = String(raw);
  if (row.field) {
    return (
      <button
        type="button"
        className={cx(styles.filterable, row.mono && styles.mono)}
        title={`Filter: ${row.label} = ${text}`}
        onClick={() => onFilter(row.field!, text)}
      >
        {text}
      </button>
    );
  }
  return <span className={cx(row.mono && styles.mono)}>{text}</span>;
};

const EventDetailPanel: React.FC<EventDetailPanelProps> = ({
  open,
  event,
  onClose,
  onFilter,
}) => {
  return (
    <Drawer
      open={open}
      onClose={onClose}
      title={
        event ? (
          <>
            <Chip tone={event.type === "Warning" ? "signal" : "steel"}>
              {event.type}
            </Chip>{" "}
            <span className={styles.mono}>{event.reason}</span> ·{" "}
            {event.involvedObject?.name}
          </>
        ) : (
          "Event"
        )
      }
    >
      {!event ? (
        <div className={styles.loading}>
          <Spinner />
        </div>
      ) : (
        <div className={styles.content}>
          <pre className={styles.message}>{event.message}</pre>

          {DETAIL_SECTIONS.map((section) => {
            const rows = section.rows.filter((row) => {
              const v = row.get(event);
              return !(
                section.optional &&
                (v === undefined || v === null || v === "")
              );
            });
            const hasValue = rows.some((row) => {
              const v = row.get(event);
              return v !== undefined && v !== null && v !== "";
            });
            if (section.optional && !hasValue) return null;
            return (
              <section key={section.title}>
                <div className={styles.sectionTitle}>{section.title}</div>
                <table className={styles.table}>
                  <tbody>
                    {rows.map((row) => (
                      <tr key={row.label}>
                        <td className={styles.labelCell}>{row.label}</td>
                        <td className={styles.valueCell}>
                          <Value row={row} event={event} onFilter={onFilter} />
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </section>
            );
          })}

          <Accordion summary="Raw JSON">
            <pre className={styles.message}>
              {JSON.stringify(event, null, 2)}
            </pre>
          </Accordion>
        </div>
      )}
    </Drawer>
  );
};

export default EventDetailPanel;
