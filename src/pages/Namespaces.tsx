import React, { useMemo, useState } from "react";
import { useUrlParams } from "../hooks/useUrlParams";
import {
  useNamespaceBuckets,
  foldOther,
  OTHER_NS,
  type NamespaceStat,
} from "../hooks/useNamespaceBuckets";
import { encodeFilters } from "../lib/filters/urlCodec";
import { intervalToSql } from "../utils/time";
import { formatCount } from "../utils/format";
import { Alert, Card, Skeleton, Sparkline, TextInput, cx } from "../ui";
import styles from "./Namespaces.module.css";

const TOP_N = 50;

type SortKey = "ns" | "total" | "warnings" | "warnRate";

const sortValue = (row: NamespaceStat, key: SortKey): number | string => {
  switch (key) {
    case "ns":
      return row.ns;
    case "total":
      return row.total;
    case "warnings":
      return row.warnings;
    case "warnRate":
      return row.total > 0 ? row.warnings / row.total : 0;
  }
};

const HEADERS: { key: SortKey; label: string; align?: "right" }[] = [
  { key: "ns", label: "Namespace" },
  { key: "total", label: "Events", align: "right" },
  { key: "warnings", label: "Warnings", align: "right" },
  { key: "warnRate", label: "Warning rate", align: "right" },
];

const Namespaces: React.FC = () => {
  const { updateParams } = useUrlParams();
  const { buckets, rows, interval, isLoading, error } = useNamespaceBuckets(40);
  const [search, setSearch] = useState("");
  const [sort, setSort] = useState<{ key: SortKey; dir: "asc" | "desc" }>({
    key: "total",
    dir: "desc",
  });

  const folded = useMemo(
    () => foldOther(rows, buckets.length, TOP_N),
    [rows, buckets],
  );

  const visible = useMemo(() => {
    const needle = search.trim().toLowerCase();
    const filtered = needle
      ? folded.filter((r) => r.ns.toLowerCase().includes(needle))
      : folded;
    return [...filtered].sort((a, b) => {
      const va = sortValue(a, sort.key);
      const vb = sortValue(b, sort.key);
      const cmp =
        typeof va === "string"
          ? va.localeCompare(vb as string)
          : (va as number) - (vb as number);
      return sort.dir === "asc" ? cmp : -cmp;
    });
  }, [folded, search, sort]);

  const toggleSort = (key: SortKey) => {
    setSort((prev) =>
      prev.key === key
        ? { key, dir: prev.dir === "desc" ? "asc" : "desc" }
        : { key, dir: key === "ns" ? "asc" : "desc" },
    );
  };

  const drill = (row: NamespaceStat) => {
    const chips =
      row.ns === OTHER_NS
        ? [
            {
              field: "namespace" as const,
              op: "notIn" as const,
              values: folded.filter((r) => r.ns !== OTHER_NS).map((r) => r.ns),
            },
          ]
        : [
            {
              field: "namespace" as const,
              op: "eq" as const,
              values: [row.ns],
            },
          ];
    updateParams(
      { filters: encodeFilters(chips), where: undefined },
      "/p/discover",
    );
  };

  return (
    <div className={styles.page}>
      <div className={styles.toolbar}>
        <TextInput
          className={styles.search}
          placeholder="Filter namespaces…"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        <span className={styles.meta}>
          {formatCount(rows.length)} namespaces · {intervalToSql(interval)}{" "}
          buckets
          {rows.length > TOP_N ? ` · showing top ${TOP_N} + (other)` : ""}
        </span>
      </div>

      {error && (
        <Alert tone="error">Failed to load namespaces: {error.message}</Alert>
      )}

      <Card flush>
        {isLoading && rows.length === 0 ? (
          <div style={{ padding: 16 }}>
            <Skeleton height={240} />
          </div>
        ) : visible.length === 0 ? (
          <div className={styles.empty}>
            {search
              ? "No namespaces match the filter"
              : "No events in the selected time range"}
          </div>
        ) : (
          <table className={styles.table}>
            <thead>
              <tr>
                {HEADERS.map((h) => {
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
                    key={row.ns}
                    className={styles.row}
                    onClick={() => drill(row)}
                  >
                    <td className={cx(styles.td, styles.ns)} title={row.ns}>
                      {row.ns}
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

export default Namespaces;
