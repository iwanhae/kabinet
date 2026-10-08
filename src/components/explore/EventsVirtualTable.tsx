import React, { useMemo, useRef } from "react";
import { TableVirtuoso, type TableComponents } from "react-virtuoso";
import { COLUMNS } from "./columns";
import type { EventListRow } from "../../types/events";
import { cx } from "../../ui";
import styles from "./EventsVirtualTable.module.css";

export interface EventsVirtualTableProps {
  events: EventListRow[];
  onRowClick: (event: EventListRow) => void;
  onEndReached: () => void;
  isLoadingMore: boolean;
  isReachingEnd: boolean;
  selectedUid?: string;
}

interface RowContext {
  onRowClick: (event: EventListRow) => void;
  selectedUid?: string;
}

const EventsVirtualTable: React.FC<EventsVirtualTableProps> = ({
  events,
  onRowClick,
  onEndReached,
  isLoadingMore,
  isReachingEnd,
  selectedUid,
}) => {
  // Refs keep the memoized components stable while handlers change.
  const ctxRef = useRef<RowContext>({ onRowClick, selectedUid });
  ctxRef.current = { onRowClick, selectedUid };

  const components = useMemo<TableComponents<EventListRow>>(
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
    <TableVirtuoso<EventListRow>
      data={events}
      components={components}
      style={{ height: "100%" }}
      overscan={400}
      endReached={onEndReached}
      fixedHeaderContent={() => (
        <tr>
          {COLUMNS.map((col) => {
            return (
              <th
                key={col.key}
                className={styles.th}
                style={col.width ? { width: col.width } : undefined}
              >
                {col.label}
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
