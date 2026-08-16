import React, { useMemo } from "react";
import dayjs from "dayjs";
import { useEventsQuery } from "../../hooks/useEventsQuery";
import { useTimeRange } from "../../hooks/useUrlParams";
import { getDynamicInterval, intervalToSql, bucketEnd } from "../../utils/time";
import { TS_EXPR } from "../../lib/sql/expr";
import { formatCount, formatCompact } from "../../utils/format";
import { Alert, Skeleton } from "../../ui";
import { EChart } from "./EChart";
import { useChartTokens } from "./chartTheme";
import styles from "./TimelineHistogram.module.css";

interface TimelineBucket {
  time_bucket: string;
  type: string;
  count: number;
}

interface Props {
  where?: string;
  height?: number;
}

/**
 * Stacked Normal/Warning histogram over the global time range.
 * Drag horizontally (brush) or click a bar to zoom the global range in.
 */
const TimelineHistogram: React.FC<Props> = ({
  where = "1=1",
  height = 260,
}) => {
  const { from, to, setTimeRange } = useTimeRange();
  const tokens = useChartTokens();

  const interval = useMemo(() => getDynamicInterval(from, to, 60), [from, to]);

  const query = useMemo(
    () => `
      SELECT
        time_bucket(INTERVAL '${intervalToSql(interval)}', ${TS_EXPR}) AS time_bucket,
        type,
        COUNT(*) AS count
      FROM $events
      WHERE ${where}
      GROUP BY time_bucket, type
      ORDER BY time_bucket, type
    `,
    [interval, where],
  );

  const { data, error, isLoading } = useEventsQuery<TimelineBucket>(query);

  const { buckets, normal, warning, total } = useMemo(() => {
    const buckets = [...new Set((data ?? []).map((d) => d.time_bucket))].sort();
    const index = new Map(buckets.map((b, i) => [b, i]));
    const normal = new Array<number>(buckets.length).fill(0);
    const warning = new Array<number>(buckets.length).fill(0);
    let total = 0;
    (data ?? []).forEach((d) => {
      const i = index.get(d.time_bucket);
      if (i === undefined) return;
      total += d.count;
      if (d.type === "Warning") warning[i] += d.count;
      else normal[i] += d.count;
    });
    return { buckets, normal, warning, total };
  }, [data]);

  const labelFormat = useMemo(() => {
    const hours = dayjs(to).diff(dayjs(from), "hour");
    if (hours <= 24) return "HH:mm";
    if (hours <= 24 * 7) return "MM/DD HH:mm";
    return "MM/DD";
  }, [from, to]);

  const zoomTo = useMemo(
    () => (startIdx: number, endIdx: number) => {
      const i0 = Math.max(
        0,
        Math.min(Math.round(startIdx), buckets.length - 1),
      );
      const i1 = Math.max(0, Math.min(Math.round(endIdx), buckets.length - 1));
      if (buckets.length === 0) return;
      setTimeRange(
        dayjs(buckets[Math.min(i0, i1)]).toISOString(),
        bucketEnd(buckets[Math.max(i0, i1)], interval).toISOString(),
      );
    },
    [buckets, interval, setTimeRange],
  );

  const onEvents = useMemo(
    () => ({
      click: (params: unknown) => {
        const p = params as { dataIndex?: number };
        if (typeof p.dataIndex === "number") zoomTo(p.dataIndex, p.dataIndex);
      },
      brushEnd: (params: unknown) => {
        const p = params as { areas?: { coordRange?: [number, number] }[] };
        const range = p.areas?.[0]?.coordRange;
        if (range) zoomTo(range[0], range[1]);
      },
    }),
    [zoomTo],
  );

  const option = useMemo(() => {
    const axisLabel = {
      color: tokens.inkMuted,
      fontFamily: tokens.fontMono,
      fontSize: 10,
    };
    return {
      animation: false,
      grid: { left: 44, right: 8, top: 12, bottom: 24 },
      brush: {
        xAxisIndex: 0,
        brushType: "lineX",
        brushMode: "single",
        throttleType: "debounce",
        throttleDelay: 100,
      },
      xAxis: {
        type: "category",
        data: buckets.map((b) => dayjs(b).format(labelFormat)),
        axisLabel,
        axisLine: { lineStyle: { color: tokens.line } },
        axisTick: { show: false },
      },
      yAxis: {
        type: "value",
        axisLabel: {
          ...axisLabel,
          formatter: (v: number) => formatCompact(v),
        },
        splitLine: { lineStyle: { color: tokens.line } },
      },
      tooltip: {
        trigger: "axis",
        backgroundColor: tokens.card,
        borderColor: tokens.line,
        textStyle: { color: tokens.ink, fontSize: 12 },
        formatter: (params: unknown) => {
          const items = params as {
            dataIndex: number;
            seriesName: string;
            value: number;
          }[];
          if (!items.length) return "";
          const i = items[0].dataIndex;
          const w = warning[i] ?? 0;
          const n = normal[i] ?? 0;
          const sum = w + n;
          const pct = sum > 0 ? ((w / sum) * 100).toFixed(1) : "0.0";
          return [
            `<b>${dayjs(buckets[i]).format("MMM DD, HH:mm:ss")}</b>`,
            `Warning&nbsp;&nbsp;<b>${formatCount(w)}</b> (${pct}%)`,
            `Normal&nbsp;&nbsp;&nbsp;<b>${formatCount(n)}</b>`,
            `Total&nbsp;&nbsp;&nbsp;&nbsp;<b>${formatCount(sum)}</b>`,
          ].join("<br/>");
        },
      },
      series: [
        {
          name: "Normal",
          type: "bar",
          stack: "events",
          data: normal,
          itemStyle: { color: tokens.steel },
          barCategoryGap: "15%",
        },
        {
          name: "Warning",
          type: "bar",
          stack: "events",
          data: warning,
          itemStyle: { color: tokens.signal },
        },
      ],
    };
  }, [buckets, normal, warning, labelFormat, tokens]);

  if (isLoading && !data) {
    return <Skeleton height={height} />;
  }

  if (error) {
    return <Alert tone="error">Failed to load timeline: {error.message}</Alert>;
  }

  if (buckets.length === 0) {
    return (
      <div className={styles.empty} style={{ height }}>
        No events in the selected time range
      </div>
    );
  }

  return (
    <div>
      <div className={styles.meta}>
        <span>
          {formatCount(total)} events · {intervalToSql(interval)} buckets · drag
          to zoom
        </span>
        <span className={styles.legend}>
          <span className={styles.legendItem}>
            <span
              className={styles.swatch}
              style={{ background: tokens.steel }}
            />
            Normal
          </span>
          <span className={styles.legendItem}>
            <span
              className={styles.swatch}
              style={{ background: tokens.signal }}
            />
            Warning
          </span>
        </span>
      </div>
      <EChart
        option={option}
        height={height}
        onEvents={onEvents}
        onInit={(chart) => {
          chart.dispatchAction({
            type: "takeGlobalCursor",
            key: "brush",
            brushOption: { brushType: "lineX", brushMode: "single" },
          });
        }}
      />
    </div>
  );
};

export default TimelineHistogram;
