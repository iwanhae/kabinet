import React, { useMemo, useRef } from "react";
import { TableVirtuoso, type TableComponents } from "react-virtuoso";
import { COLUMNS } from "./columns";
import type { EventResult } from "../../types/events";
import type { SortSpec, SortKey } from "../../lib/sql/pagination";
import { cx } from "../../ui";
import styles from "./EventsVirtualTable.module.css";

export interface EventsVirtualTableProps {
  events: EventResult[];
  sort: SortSpec;
  onSortChange: (key: SortKey) => void;
  onRowClick: (event: EventResult) => void;
  onEndReached: () => void;
  isLoadingMore: boolean;
  isReachingEnd: boolean;
  selectedUid?: string;
}

interface RowContext {
  onRowClick: (event: EventResult) => void;
  selectedUid?: string;
}

const EventsVirtualTable: React.FC<EventsVirtualTableProps> = ({
  events,
  sort,
  onSortChange,
  onRowClick,
  onEndReached,
  isLoadingMore,
  isReachingEnd,
  selectedUid,
}) => {
  // Refs keep the memoized components stable while handlers change.
  const ctxRef = useRef<RowContext>({ onRowClick, selectedUid });
  ctxRef.current = { onRowClick, selectedUid };

  const components = useMemo<TableComponents<EventResult>>(
    () => ({
      Table: (props) => <table {...props} className={styles.table} />,
      TableRow: ({ item, ...props }) => {
        const { onRowClick, selectedUid } = ctxRef.current;
        return (
          <tr
            {...props}
            className={cx(
              styles.row,
              item.type === "Warning" && styles.rowWarning,
              item.metadata.uid === selectedUid && styles.rowSelected,
            )}
            onClick={() => onRowClick(item)}
          />
        );
      },
    }),
    [],
  );

  if (events.length === 0 && isReachingEnd) {
    return <div className={styles.empty}>No events match the filters</div>;
  }

  return (
    <TableVirtuoso<EventResult>
      data={events}
      components={components}
      style={{ height: "100%" }}
      overscan={400}
      endReached={onEndReached}
      fixedHeaderContent={() => (
        <tr>
          {COLUMNS.map((col) => {
            const sortable = col.sortKey !== undefined;
            const active = sortable && sort.key === col.sortKey;
            return (
              <th
                key={col.key}
                className={cx(styles.th, sortable && styles.thSortable)}
                style={col.width ? { width: col.width } : undefined}
                onClick={
                  sortable ? () => onSortChange(col.sortKey!) : undefined
                }
                aria-sort={
                  active
                    ? sort.dir === "asc"
                      ? "ascending"
                      : "descending"
                    : undefined
                }
              >
                {col.label}
                {active && (
                  <span className={styles.sortMark}>
                    {sort.dir === "asc" ? "▲" : "▼"}
                  </span>
                )}
              </th>
            );
          })}
        </tr>
      )}
      itemContent={(_index, event) => (
        <>
          {COLUMNS.map((col) => (
            <td
              key={col.key}
              className={cx(
                styles.td,
                col.mono && styles.mono,
                col.align === "right" && styles.right,
              )}
              title={
                col.key === "message" || col.key === "object"
                  ? String(col.render(event) ?? "")
                  : undefined
              }
            >
              {col.render(event)}
            </td>
          ))}
        </>
      )}
      fixedFooterContent={
        isLoadingMore || !isReachingEnd
          ? () => (
              <tr>
                <td colSpan={COLUMNS.length} className={styles.footer}>
                  {isLoadingMore ? "loading more…" : "scroll for more"}
                </td>
              </tr>
            )
          : undefined
      }
    />
  );
};

export default EventsVirtualTable;
