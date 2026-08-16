import React, { useMemo, useState } from "react";
import { useFilters } from "../../hooks/useFilters";
import {
  useDimensionBuckets,
  foldOther,
  OTHER_KEY,
  type DimensionField,
  type DimensionStat,
} from "../../hooks/useDimensionBuckets";
import { intervalToSql } from "../../utils/time";
import { formatCount } from "../../utils/format";
import { Alert, Card, Skeleton, Sparkline, TextInput, cx } from "../../ui";
import styles from "./DimensionPage.module.css";

type SortKey = "key" | "total" | "warnings" | "warnRate";

const sortValue = (row: DimensionStat, key: SortKey): number | string => {
  switch (key) {
    case "key":
      return row.key;
    case "total":
      return row.total;
    case "warnings":
      return row.warnings;
    case "warnRate":
      return row.total > 0 ? row.warnings / row.total : 0;
  }
};

export interface DimensionPageProps {
  /** FIELD_DEFS key that both groups the scan and drives drill-down chips. */
  field: DimensionField;
  /** Column header and search placeholder noun, e.g. "Namespace". */
  noun: string;
  /** Plural noun for the meta line, e.g. "namespaces". */
  nounPlural: string;
  /** Rows kept before folding into "(other)". */
  topN?: number;
}

/**
 * Generic per-dimension analytics table: totals, warnings, warning-rate bar,
 * and a time-trend sparkline per value, with search, sort, and click-to-drill
 * into Explore. Backs the Namespaces / Nodes / Components tabs.
 */
const DimensionPage: React.FC<DimensionPageProps> = ({
  field,
  noun,
  nounPlural,
  topN = 50,
}) => {
  const { drill } = useFilters();
  const { buckets, rows, interval, isLoading, error } = useDimensionBuckets(
    field,
    40,
  );
  const [search, setSearch] = useState("");
  const [sort, setSort] = useState<{ key: SortKey; dir: "asc" | "desc" }>({
    key: "total",
    dir: "desc",
  });

  const headers: { key: SortKey; label: string; align?: "right" }[] = [
    { key: "key", label: noun },
    { key: "total", label: "Events", align: "right" },
    { key: "warnings", label: "Warnings", align: "right" },
    { key: "warnRate", label: "Warning rate", align: "right" },
  ];

  const folded = useMemo(
    () => foldOther(rows, buckets.length, topN),
    [rows, buckets, topN],
  );

  const visible = useMemo(() => {
    const needle = search.trim().toLowerCase();
    // While searching, match against ALL values (not just the folded top-N),
    // so a long-tail node/namespace is still findable.
    const source = needle
      ? rows.filter((r) => r.key.toLowerCase().includes(needle))
      : folded;
    return [...source].sort((a, b) => {
      const va = sortValue(a, sort.key);
      const vb = sortValue(b, sort.key);
      const cmp =
        typeof va === "string"
          ? va.localeCompare(vb as string)
          : (va as number) - (vb as number);
      return sort.dir === "asc" ? cmp : -cmp;
    });
  }, [rows, folded, search, sort]);

  const toggleSort = (key: SortKey) => {
    setSort((prev) =>
      prev.key === key
        ? { key, dir: prev.dir === "desc" ? "asc" : "desc" }
        : { key, dir: key === "key" ? "asc" : "desc" },
    );
  };

  const drillRow = (row: DimensionStat) => {
    const chips =
      row.key === OTHER_KEY
        ? [
            {
              field,
              op: "notIn" as const,
              values: folded
                .filter((r) => r.key !== OTHER_KEY)
                .map((r) => r.key),
            },
          ]
        : [{ field, op: "eq" as const, values: [row.key] }];
    drill(chips);
  };

  return (
    <div className={styles.page}>
      <div className={styles.toolbar}>
        <TextInput
          className={styles.search}
          placeholder={`Filter ${nounPlural}…`}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        <span className={styles.meta}>
          {formatCount(rows.length)} {nounPlural} · {intervalToSql(interval)}{" "}
          buckets
          {!search && rows.length > topN
            ? ` · showing top ${topN} + (other)`
            : ""}
        </span>
      </div>

      {error && (
        <Alert tone="error">
          Failed to load {nounPlural}: {error.message}
        </Alert>
      )}

      <Card flush>
        {isLoading && rows.length === 0 ? (
          <div style={{ padding: 16 }}>
            <Skeleton height={240} />
          </div>
        ) : visible.length === 0 ? (
          <div className={styles.empty}>
            {search
              ? `No ${nounPlural} match the filter`
              : "No events in the selected time range"}
          </div>
        ) : (
          <table className={styles.table}>
            <thead>
              <tr>
                {headers.map((h) => {
                  const active = sort.key === h.key;
                  return (
                    <th
                      key={h.key}
                      className={cx(
                        styles.th,
                        styles.thSortable,
                        h.align === "right" && styles.right,
                      )}
                      onClick={() => toggleSort(h.key)}
                      aria-sort={
                        active
                          ? sort.dir === "asc"
                            ? "ascending"
                            : "descending"
                          : undefined
                      }
                    >
                      {h.label}
                      {active && (
                        <span className={styles.sortMark}>
                          {sort.dir === "asc" ? "▲" : "▼"}
                        </span>
                      )}
                    </th>
                  );
                })}
                <th className={styles.th}>Trend</th>
              </tr>
            </thead>
            <tbody>
              {visible.map((row) => {
                const pct =
                  row.total > 0 ? (row.warnings / row.total) * 100 : 0;
                return (
                  <tr
                    key={row.key}
                    className={styles.row}
                    onClick={() => drillRow(row)}
                  >
                    <td
                      className={cx(styles.td, styles.dimKey)}
                      title={row.key}
                    >
                      {row.key}
                    </td>
                    <td className={cx(styles.td, styles.num)}>
                      {formatCount(row.total)}
                    </td>
                    <td className={cx(styles.td, styles.num)}>
                      {formatCount(row.warnings)}
                    </td>
                    <td className={styles.td}>
                      <span className={styles.warnCell}>
                        <span className={styles.warnBarTrack}>
                          <span
                            className={styles.warnBarFill}
                            style={{ width: `${Math.min(100, pct)}%` }}
                          />
                        </span>
                        <span className={styles.warnPct}>
                          {pct.toFixed(1)}%
                        </span>
                      </span>
                    </td>
                    <td className={styles.td}>
                      <Sparkline
                        points={row.trend}
                        warnPoints={row.warnTrend}
                        width={140}
                        height={28}
                      />
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </Card>
    </div>
  );
};

export default DimensionPage;
