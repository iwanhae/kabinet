import { TS_EXPR } from "./expr";
import { escapeSqlString } from "../filters/compile";
import type { EventResult } from "../../types/events";

export type SortKey = "ts" | "count" | "namespace" | "reason";

export interface SortSpec {
  key: SortKey;
  dir: "asc" | "desc";
}

export interface PageCursor {
  ts: string;
  uid: string;
  /** Value of the primary sort column for non-time sorts. */
  extra?: string | number;
}

interface CursorKey {
  expr: string;
  dir: "asc" | "desc";
  literal: string;
}

/** COALESCEd so every key is a total order (NULLs would break keyset math). */
const SORT_EXPRS: Record<
  SortKey,
  { expr: string; kind: "ts" | "str" | "num" }
> = {
  ts: { expr: TS_EXPR, kind: "ts" },
  count: { expr: 'COALESCE("count", 1)', kind: "num" },
  namespace: { expr: "COALESCE(metadata.namespace, '')", kind: "str" },
  reason: { expr: "COALESCE(reason, '')", kind: "str" },
};

const literalFor = (kind: "ts" | "str" | "num", value: string | number) =>
  kind === "num"
    ? String(Number(value) || 0)
    : kind === "ts"
      ? `TIMESTAMPTZ '${escapeSqlString(String(value))}'`
      : `'${escapeSqlString(String(value))}'`;

const cursorKeys = (sort: SortSpec, cursor: PageCursor): CursorKey[] => {
  const keys: CursorKey[] = [];
  if (sort.key !== "ts") {
    const def = SORT_EXPRS[sort.key];
    keys.push({
      expr: def.expr,
      dir: sort.dir,
      literal: literalFor(def.kind, cursor.extra ?? ""),
    });
  }
  keys.push({
    expr: TS_EXPR,
    dir: sort.key === "ts" ? sort.dir : "desc",
    literal: literalFor("ts", cursor.ts),
  });
  keys.push({
    expr: "metadata.uid",
    dir: sort.key === "ts" ? sort.dir : "desc",
    literal: literalFor("str", cursor.uid),
  });
  return keys;
};

/**
 * Lexicographic keyset predicate, expanded as nested OR/AND (instead of a
 * tuple comparison) so the leading strict comparison stays eligible for
 * parquet zone-map pruning.
 */
const cursorPredicate = (keys: CursorKey[]): string => {
  const [head, ...rest] = keys;
  const cmp = head.dir === "desc" ? "<" : ">";
  const strict = `${head.expr} ${cmp} ${head.literal}`;
  if (rest.length === 0) return strict;
  return `(${strict} OR (${head.expr} = ${head.literal} AND ${cursorPredicate(rest)}))`;
};

const orderBy = (sort: SortSpec): string => {
  const dir = sort.dir.toUpperCase();
  if (sort.key === "ts") {
    return `${TS_EXPR} ${dir}, metadata.uid ${dir}`;
  }
  return `${SORT_EXPRS[sort.key].expr} ${dir}, ${TS_EXPR} DESC, metadata.uid DESC`;
};

export function buildPageQuery(
  whereSql: string,
  sort: SortSpec,
  cursor: PageCursor | null,
  limit: number,
): string {
  const cursorClause = cursor
    ? ` AND ${cursorPredicate(cursorKeys(sort, cursor))}`
    : "";
  return `
    SELECT * FROM $events
    WHERE (${whereSql})${cursorClause}
    ORDER BY ${orderBy(sort)}
    LIMIT ${limit}
  `;
}

/** Extracts the cursor for the next page from the last raw row of a page. */
export function cursorFromRow(row: EventResult, sort: SortSpec): PageCursor {
  const ts =
    row.lastTimestamp ?? row.eventTime ?? row.metadata.creationTimestamp;
  const cursor: PageCursor = { ts, uid: row.metadata.uid };
  if (sort.key === "count") cursor.extra = row.count ?? 1;
  if (sort.key === "namespace") cursor.extra = row.metadata.namespace ?? "";
  if (sort.key === "reason") cursor.extra = row.reason ?? "";
  return cursor;
}
