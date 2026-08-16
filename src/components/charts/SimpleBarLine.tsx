import React, { useMemo } from "react";
import { EChart } from "./EChart";
import { useChartTokens } from "./chartTheme";
import { formatCompact } from "../../utils/format";

export interface SimpleBarLineProps {
  type: "bar" | "line";
  content: Record<string, unknown>[];
  height?: number;
}

const LABEL_KEYS = ["label", "name", "date"];

/**
 * Generic bar/line chart over an array of row objects.
 * The label column is auto-detected (label/name/date, else the first key);
 * every other numeric-ish column becomes a series.
 */
export const SimpleBarLine: React.FC<SimpleBarLineProps> = ({
  type,
  content,
  height = 280,
}) => {
  const tokens = useChartTokens();

  const option = useMemo(() => {
    if (content.length === 0) return {};
    const keys = Object.keys(content[0]);
    const labelKey = keys.find((k) => LABEL_KEYS.includes(k)) ?? keys[0];
    const seriesKeys = keys.filter((k) => k !== labelKey);
    const palette = [
      tokens.steel,
      tokens.signal,
      tokens.steelStrong,
      tokens.signalStrong,
      tokens.inkFaint,
    ];
    const axisLabel = {
      color: tokens.inkMuted,
      fontFamily: tokens.fontMono,
      fontSize: 10,
    };

    return {
      animation: false,
      grid: { left: 44, right: 8, top: 28, bottom: 24 },
      legend:
        seriesKeys.length > 1
          ? { top: 0, textStyle: { color: tokens.inkMuted, fontSize: 11 } }
          : undefined,
      xAxis: {
        type: "category",
        data: content.map((row) => String(row[labelKey])),
        axisLabel,
        axisLine: { lineStyle: { color: tokens.line } },
        axisTick: { show: false },
      },
      yAxis: {
        type: "value",
        axisLabel: { ...axisLabel, formatter: (v: number) => formatCompact(v) },
        splitLine: { lineStyle: { color: tokens.line } },
      },
      tooltip: {
        trigger: "axis",
        backgroundColor: tokens.card,
        borderColor: tokens.line,
        textStyle: { color: tokens.ink, fontSize: 12 },
      },
      series: seriesKeys.map((key, i) => ({
        name: key,
        type,
        data: content.map((row) => Number(row[key]) || 0),
        itemStyle: { color: palette[i % palette.length] },
        lineStyle:
          type === "line"
            ? { color: palette[i % palette.length], width: 2 }
            : undefined,
        symbol: "none",
      })),
    };
  }, [content, type, tokens]);

  if (content.length === 0) return null;
  return <EChart option={option} height={height} />;
};
