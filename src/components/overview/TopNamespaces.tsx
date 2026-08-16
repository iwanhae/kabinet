import React from "react";
import { useUrlParams } from "../../hooks/useUrlParams";
import { useNamespaceBuckets } from "../../hooks/useNamespaceBuckets";
import { encodeFilters } from "../../lib/filters/urlCodec";
import { formatCount } from "../../utils/format";
import { Alert, Skeleton, Sparkline, cx } from "../../ui";
import styles from "./TopNamespaces.module.css";

const TOP_N = 5;

/** Top namespaces by volume with a per-namespace trend sparkline. */
const TopNamespaces: React.FC = () => {
  const { updateParams } = useUrlParams();
  const { rows, isLoading, error } = useNamespaceBuckets(40);

  if (error) {
    return (
      <Alert tone="error">Failed to load namespaces: {error.message}</Alert>
    );
  }

  if (isLoading && rows.length === 0) {
    return (
      <div className={styles.list}>
        {Array.from({ length: TOP_N }, (_, i) => (
          <div key={i} className={styles.row}>
            <Skeleton height={20} />
          </div>
        ))}
      </div>
    );
  }

  const top = rows.slice(0, TOP_N);
  if (top.length === 0) {
    return <div className={styles.empty}>No events in the selected range</div>;
  }

  const drill = (ns: string) => {
    updateParams(
      {
        filters: encodeFilters([
          { field: "namespace", op: "eq", values: [ns] },
        ]),
        where: undefined,
      },
      "/p/discover",
    );
  };

  return (
    <div className={styles.list}>
      {top.map((row) => {
        const pct = row.total > 0 ? (row.warnings / row.total) * 100 : 0;
        return (
          <button
            key={row.ns}
            type="button"
            className={styles.row}
            onClick={() => drill(row.ns)}
            title={`${formatCount(row.total)} events · ${formatCount(row.warnings)} warnings`}
          >
            <span className={styles.ns}>{row.ns}</span>
            <Sparkline
              points={row.trend}
              warnPoints={row.warnTrend}
              width={96}
              height={24}
            />
            <span className={styles.count}>{formatCount(row.total)}</span>
            <span
              className={cx(styles.warnPct, pct >= 10 && styles.warnPctHot)}
            >
              {pct.toFixed(1)}%
            </span>
          </button>
        );
      })}
    </div>
  );
};

export default TopNamespaces;
