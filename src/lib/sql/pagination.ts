import { escapeSqlString } from "../filters/compile";
import type { EventListRow } from "../../types/events";

export interface PageCursor {
  ts: string;
  uid: string;
}

/** The list only needs the displayed fields; load the full event on selection. */
export function buildPageQuery(
  whereSql: string,
  windowEnd: string,
  cursor: PageCursor | null,
  limit: number,
): string {
  const cursorClause = cursor
    ? ` AND (timestamp < TIMESTAMPTZ '${escapeSqlString(cursor.ts)}' OR
        (timestamp = TIMESTAMPTZ '${escapeSqlString(cursor.ts)}' AND metadata.uid < '${escapeSqlString(cursor.uid)}'))`
    : "";
  return `
    SELECT
      timestamp,
      struct_pack(namespace := metadata.namespace, uid := metadata.uid, resourceVersion := metadata.resourceVersion) AS metadata,
      struct_pack(kind := involvedObject.kind, name := involvedObject.name) AS involvedObject,
      type, reason, "count", LEFT(message, 240) AS message
    FROM $events
    WHERE (${whereSql}) AND timestamp < TIMESTAMPTZ '${escapeSqlString(windowEnd)}'${cursorClause}
    ORDER BY timestamp DESC, metadata.uid DESC
    LIMIT ${limit}
  `;
}

export function cursorFromRow(row: EventListRow): PageCursor {
  return { ts: row.timestamp, uid: row.metadata.uid };
}
