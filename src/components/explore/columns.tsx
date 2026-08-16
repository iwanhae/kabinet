import React from "react";
import dayjs from "dayjs";
import { Chip } from "../../ui";
import type { EventResult } from "../../types/events";
import type { SortKey } from "../../lib/sql/pagination";
import { formatCount } from "../../utils/format";

export interface ColumnDef {
  key: string;
  label: string;
  /** Fixed width in px; omit for the flexible column. */
  width?: number;
  sortKey?: SortKey;
  align?: "left" | "right";
  mono?: boolean;
  render: (event: EventResult) => React.ReactNode;
}

export const eventTimestamp = (event: EventResult): string =>
  event.lastTimestamp ?? event.eventTime ?? event.metadata.creationTimestamp;

export const COLUMNS: ColumnDef[] = [
  {
    key: "type",
    label: "Type",
    width: 88,
    render: (e) => (
      <Chip tone={e.type === "Warning" ? "signal" : "steel"}>{e.type}</Chip>
    ),
  },
  {
    key: "time",
    label: "Time",
    width: 148,
    sortKey: "ts",
    mono: true,
    render: (e) => dayjs(eventTimestamp(e)).format("MM-DD HH:mm:ss"),
  },
  {
    key: "namespace",
    label: "Namespace",
    width: 150,
    sortKey: "namespace",
    mono: true,
    render: (e) => e.metadata.namespace ?? "—",
  },
  {
    key: "kind",
    label: "Kind",
    width: 100,
    render: (e) => e.involvedObject?.kind ?? "—",
  },
  {
    key: "object",
    label: "Object",
    width: 220,
    mono: true,
    render: (e) => e.involvedObject?.name ?? "—",
  },
  {
    key: "reason",
    label: "Reason",
    width: 170,
    sortKey: "reason",
    mono: true,
    render: (e) => e.reason,
  },
  {
    key: "count",
    label: "Count",
    width: 64,
    sortKey: "count",
    align: "right",
    mono: true,
    render: (e) => formatCount(e.count ?? 1),
  },
  {
    key: "message",
    label: "Message",
    render: (e) => e.message,
  },
];
