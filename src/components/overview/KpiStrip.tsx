import React from "react";
import { useEventsQuery } from "../../hooks/useEventsQuery";
import { useUrlParams } from "../../hooks/useUrlParams";
import {
  buildKpiQuery,
  FAILED_POD_REASONS,
  NODE_ISSUE_REASONS,
  STORAGE_REASONS,
  type KpiRow,
} from "../../lib/sql/overview";
import { encodeFilters } from "../../lib/filters/urlCodec";
import type { FilterChip } from "../../lib/filters/model";
import { formatCount } from "../../utils/format";
import { Alert, Skeleton, cx } from "../../ui";
import styles from "./KpiStrip.module.css";

type DrillChips = Array<Omit<FilterChip, "id">>;

interface KpiDef {
  key: keyof KpiRow;
  label: string;
  /** Marks the value as belonging to the hot (warning) axis. */
  hot?: boolean;
  chips?: DrillChips;
  format?: (row: KpiRow) => string;
}

const KPIS: KpiDef[] = [
  { key: "total_events", label: "Events" },
  {
    key: "warning_events",
    label: "Warning rate",
    hot: true,
    chips: [{ field: "type", op: "eq", values: ["Warning"] }],
    format: (row) =>
      row.total_events > 0
        ? `${((row.warning_events / row.total_events) * 100).toFixed(1)}%`
        : "0%",
  },
  { key: "active_namespaces", label: "Namespaces" },
  { key: "distinct_objects", label: "Objects" },
  {
    key: "failed_pods",
    label: "Failed pods",
    hot: true,
    chips: [{ field: "reason", op: "in", values: FAILED_POD_REASONS }],
  },
  {
    key: "restarts",
    label: "Restarts",
    hot: true,
    chips: [{ field: "reason", op: "eq", values: ["BackOff"] }],
  },
  {
    key: "node_issues",
    label: "Node issues",
    hot: true,
    chips: [
      { field: "type", op: "eq", values: ["Warning"] },
      { field: "reason", op: "in", values: NODE_ISSUE_REASONS },
    ],
  },
  {
    key: "storage_events",
    label: "Storage",
    hot: true,
    chips: [{ field: "reason", op: "in", values: STORAGE_REASONS }],
  },
];

const KpiStrip: React.FC = () => {
  const { updateParams } = useUrlParams();
  const { data, error, isLoading } = useEventsQuery<KpiRow>(buildKpiQuery(), {
    scope: "overview",
  });

  if (error) {
    return <Alert tone="error">Failed to load metrics: {error.message}</Alert>;
  }

  const row = data?.[0];

  const drill = (chips: DrillChips) => {
    updateParams(
      { filters: encodeFilters(chips), where: undefined },
      "/p/discover",
    );
  };

  return (
    <div className={styles.strip}>
      {KPIS.map((kpi) => {
        const value =
          row === undefined
            ? null
            : (kpi.format?.(row) ?? formatCount(row[kpi.key]));
        const hot = kpi.hot && row !== undefined && row[kpi.key] > 0;
        const content = (
          <>
            <span className={styles.value}>
              {isLoading || value === null ? (
                <Skeleton width={72} height={26} />
              ) : (
                value
              )}
            </span>
            <span className={styles.label}>{kpi.label}</span>
          </>
        );
        if (kpi.chips) {
          return (
            <button
              key={kpi.key}
              type="button"
              className={cx(styles.kpi, hot && styles.hot)}
              onClick={() => drill(kpi.chips!)}
              title={`Explore: ${kpi.label}`}
            >
              {content}
            </button>
          );
        }
        return (
          <div key={kpi.key} className={cx(styles.kpi, hot && styles.hot)}>
            {content}
          </div>
        );
      })}
    </div>
  );
};

export default KpiStrip;
