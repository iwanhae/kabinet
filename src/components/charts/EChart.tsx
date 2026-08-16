import React, { useEffect, useRef } from "react";
import { echarts, type EChartsCoreOption } from "./echartsSetup";
import type { EChartsType } from "echarts/core";

export interface EChartProps {
  option: EChartsCoreOption;
  height: number | string;
  /** Event name -> handler, e.g. { click, brushEnd }. */
  onEvents?: Record<string, (params: unknown) => void>;
  /** Runs once after init — for dispatchAction setup (brush cursor etc). */
  onInit?: (chart: EChartsType) => void;
}

export const EChart: React.FC<EChartProps> = ({
  option,
  height,
  onEvents,
  onInit,
}) => {
  const divRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<EChartsType | null>(null);
  const onInitRef = useRef(onInit);
  onInitRef.current = onInit;

  useEffect(() => {
    const div = divRef.current;
    if (!div) return;
    const chart = echarts.init(div);
    chartRef.current = chart;
    onInitRef.current?.(chart);

    const observer = new ResizeObserver(() => chart.resize());
    observer.observe(div);
    return () => {
      observer.disconnect();
      chart.dispose();
      chartRef.current = null;
    };
  }, []);

  useEffect(() => {
    chartRef.current?.setOption(option, { notMerge: true });
  }, [option]);

  useEffect(() => {
    const chart = chartRef.current;
    if (!chart || !onEvents) return;
    const entries = Object.entries(onEvents);
    entries.forEach(([event, handler]) => chart.on(event, handler));
    return () => {
      if (chart.isDisposed()) return;
      entries.forEach(([event, handler]) => chart.off(event, handler));
    };
  }, [onEvents]);

  return <div ref={divRef} style={{ height, width: "100%" }} />;
};
