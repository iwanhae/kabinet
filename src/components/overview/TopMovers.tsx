import React, { useMemo } from "react";
import dayjs from "dayjs";
import { useEventsQuery } from "../../hooks/useEventsQuery";
import { useFilters } from "../../hooks/useFilters";
import { useTimeRange } from "../../hooks/useUrlParams";
import { buildTopMoversQuery, type TopMoverRow } from "../../lib/sql/overview";
import { formatCount } from "../../utils/format";
import { Alert, Skeleton, cx } from "../../ui";
import styles from "./TopMovers.module.css";

/**
 * Reasons with the biggest count change vs the previous period of equal
 * length. Scans a doubled window (previous + current) in one query.
 */
const TopMovers: React.FC = () => {
  const { from, to } = useTimeRange();
  const { whereSql, drill } = useFilters();

  const prevFrom = useMemo(() => {
    const spanSeconds = dayjs(to).diff(dayjs(from), "second");
    return dayjs(from).subtract(spanSeconds, "second").format();
  }, [from, to]);

  const query = useMemo(
    () => buildTopMoversQuery(from, 10, whereSql),
    [from, whereSql],
  );
  const { data, error, isLoading } = useEventsQuery<TopMoverRow>(query, {
    from: prevFrom,
    to,
    scope: "overview",
  });

  if (error) {
    return (
      <Alert tone="error">Failed to load top movers: {error.message}</Alert>
    );
  }

  if (isLoading && !data) {
    return (
      <div className={styles.list}>
        {Array.from({ length: 6 }, (_, i) => (
          <div key={i} className={styles.row}>
            <Skeleton height={14} />
          </div>
        ))}
      </div>
    );
  }

  const rows = (data ?? []).filter(
    (r) => r.current_count > 0 || r.previous_count > 0,
  );

  if (rows.length === 0) {
    return <div className={styles.empty}>No events in either period</div>;
  }

  const drillReason = (reason: string) => {
    drill([{ field: "reason", op: "eq", values: [reason] }]);
  };

  return (
    <div className={styles.list}>
      {rows.map((row) => {
        const delta = row.current_count - row.previous_count;
        const isNew = row.previous_count === 0 && row.current_count > 0;
        const gone = row.current_count === 0 && row.previous_count > 0;
        const pct =
          row.previous_count > 0
            ? Math.round((delta / row.previous_count) * 100)
            : 0;
        return (
          <button
            key={row.reason}
            type="button"
            className={styles.row}
            onClick={() => drillReason(row.reason)}
            title={`current ${formatCount(row.current_count)} · previous ${formatCount(row.previous_count)}`}
          >
            <span className={styles.reason}>{row.reason}</span>
            <span className={styles.count}>
              {formatCount(row.current_count)}
            </span>
            <span
              className={cx(
                styles.delta,
                isNew && styles.new,
                !isNew && delta > 0 && styles.up,
                delta < 0 && styles.down,
              )}
            >
              {isNew
                ? "new"
                : gone
                  ? "gone"
                  : delta === 0
                    ? "±0"
                    : `${delta > 0 ? "▲" : "▼"} ${Math.abs(pct)}%`}
            </span>
          </button>
        );
      })}
    </div>
  );
};

export default TopMovers;
