import React, { useMemo } from "react";
import dayjs from "dayjs";
import { useTimeRange } from "../../hooks/useUrlParams";
import { useFilters } from "../../hooks/useFilters";
import {
  useDimensionBuckets,
  foldOther,
  OTHER_KEY,
} from "../../hooks/useDimensionBuckets";
import { bucketEnd } from "../../utils/time";
import { formatCount } from "../../utils/format";
import { Alert, Skeleton } from "../../ui";
import { EChart } from "./EChart";
import { useChartTokens } from "./chartTheme";

const TOP_N = 20;

const hexToRgb = (hex: string): [number, number, number] => {
  const h = hex.replace("#", "");
  return [
    parseInt(h.slice(0, 2), 16),
    parseInt(h.slice(2, 4), 16),
    parseInt(h.slice(4, 6), 16),
  ];
};

const mixRgb = (
  a: [number, number, number],
  b: [number, number, number],
  t: number,
): [number, number, number] => [
  Math.round(a[0] + (b[0] - a[0]) * t),
  Math.round(a[1] + (b[1] - a[1]) * t),
  Math.round(a[2] + (b[2] - a[2]) * t),
];

/**
 * The Cabinet: namespaces as drawers (rows), time as columns.
 * Cell opacity encodes volume (log scale); hue encodes warning ratio
 * (blue → red, fully hot at ≥50% warnings). Click a cell to open that
 * drawer in Explore, narrowed to the cell's time bucket.
 */
const CabinetHeatmap: React.FC = () => {
  const { from, to } = useTimeRange();
  const { drill } = useFilters();
  const tokens = useChartTokens();

  const { buckets, rows, interval, isLoading, error } = useDimensionBuckets(
    "namespace",
    40,
  );

  const { namespaces, cells, maxTotal } = useMemo(() => {
    const folded = foldOther(rows, buckets.length, TOP_N);
    const namespaces = folded.map((r) => r.key);
    const cells: { x: number; y: number; total: number; warnings: number }[] =
      [];
    let maxTotal = 0;
    folded.forEach((row, y) => {
      row.trend.forEach((total, x) => {
        if (total === 0) return;
        maxTotal = Math.max(maxTotal, total);
        cells.push({ x, y, total, warnings: row.warnTrend[x] });
      });
    });
    return { namespaces, cells, maxTotal };
  }, [rows, buckets]);

  const labelFormat = useMemo(() => {
    const hours = dayjs(to).diff(dayjs(from), "hour");
    if (hours <= 24) return "HH:mm";
    if (hours <= 24 * 7) return "MM/DD HH:mm";
    return "MM/DD";
  }, [from, to]);

  const option = useMemo(() => {
    const steel = hexToRgb(tokens.steel);
    const signal = hexToRgb(tokens.signal);
    const axisLabel = {
      color: tokens.inkMuted,
      fontFamily: tokens.fontMono,
      fontSize: 10,
    };
    return {
      animation: false,
      grid: { left: 118, right: 8, top: 8, bottom: 24 },
      xAxis: {
        type: "category",
        data: buckets.map((b) => dayjs(b).format(labelFormat)),
        axisLabel,
        axisLine: { lineStyle: { color: tokens.line } },
        axisTick: { show: false },
        splitArea: { show: false },
      },
      yAxis: {
        type: "category",
        data: namespaces,
        inverse: true,
        axisLabel: { ...axisLabel, width: 108, overflow: "truncate" },
        axisLine: { lineStyle: { color: tokens.line } },
        axisTick: { show: false },
      },
      tooltip: {
        backgroundColor: tokens.card,
        borderColor: tokens.line,
        textStyle: { color: tokens.ink, fontSize: 12 },
        formatter: (params: unknown) => {
          const p = params as { value: [number, number, number] };
          const [x, y] = p.value;
          const cell = cells.find((c) => c.x === x && c.y === y);
          if (!cell) return "";
          const pct =
            cell.total > 0
              ? ((cell.warnings / cell.total) * 100).toFixed(1)
              : "0.0";
          return [
            `<b>${namespaces[y]}</b>`,
            dayjs(buckets[x]).format("MMM DD, HH:mm"),
            `Total <b>${formatCount(cell.total)}</b>`,
            `Warning <b>${formatCount(cell.warnings)}</b> (${pct}%)`,
          ].join("<br/>");
        },
      },
      series: [
        {
          // Custom series instead of the heatmap series: it renders per-cell
          // computed colors without requiring a visualMap component.
          type: "custom",
          renderItem: (params: unknown, api: unknown) => {
            const p = params as { dataIndex: number };
            const a = api as {
              value: (i: number) => number;
              coord: (v: number[]) => number[];
              size: (v: number[]) => number[];
            };
            const [cx, cy] = a.coord([a.value(0), a.value(1)]);
            const [w, h] = a.size([1, 1]);
            const cell = cells[p.dataIndex];
            const ratio = cell.total > 0 ? cell.warnings / cell.total : 0;
            const [r, g, b] = mixRgb(steel, signal, Math.min(1, ratio / 0.5));
            const alpha =
              maxTotal > 0
                ? 0.12 + 0.88 * (Math.log1p(cell.total) / Math.log1p(maxTotal))
                : 0.12;
            return {
              type: "rect",
              shape: {
                x: cx - w / 2 + 0.5,
                y: cy - h / 2 + 0.5,
                width: Math.max(1, w - 1),
                height: Math.max(1, h - 1),
              },
              style: { fill: `rgba(${r}, ${g}, ${b}, ${alpha})` },
            };
          },
          data: cells.map((cell) => [cell.x, cell.y, cell.total]),
        },
      ],
    };
  }, [buckets, namespaces, cells, maxTotal, labelFormat, tokens]);

  const onEvents = useMemo(
    () => ({
      click: (params: unknown) => {
        const p = params as { value?: [number, number, number] };
        if (!p.value) return;
        const [x, y] = p.value;
        const ns = namespaces[y];
        const bucket = buckets[x];
        if (ns === undefined || bucket === undefined) return;
        const chips =
          ns === OTHER_KEY
            ? [
                {
                  field: "namespace" as const,
                  op: "notIn" as const,
                  values: namespaces.filter((n) => n !== OTHER_KEY),
                },
              ]
            : [
                {
                  field: "namespace" as const,
                  op: "eq" as const,
                  values: [ns],
                },
              ];
        drill(chips, {
          from: dayjs(bucket).toISOString(),
          to: bucketEnd(bucket, interval).toISOString(),
        });
      },
    }),
    [namespaces, buckets, interval, drill],
  );

  const height = Math.max(220, namespaces.length * 20 + 60);

  if (isLoading && rows.length === 0) return <Skeleton height={280} />;
  if (error) {
    return <Alert tone="error">Failed to load heatmap: {error.message}</Alert>;
  }
  if (cells.length === 0) {
    return (
      <div
        style={{
          height: 220,
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          color: "var(--ink-faint)",
          fontSize: 13,
        }}
      >
        No events in the selected time range
      </div>
    );
  }

  return <EChart option={option} height={height} onEvents={onEvents} />;
};

export default CabinetHeatmap;
