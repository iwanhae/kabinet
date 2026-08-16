import * as echarts from "echarts/core";
import { BarChart, LineChart, CustomChart } from "echarts/charts";
import {
  GridComponent,
  TooltipComponent,
  LegendComponent,
  DataZoomComponent,
  BrushComponent,
  // BrushComponent references the toolbox internally; without this it logs
  // "Component toolbox is used but not imported" on every setOption.
  ToolboxComponent,
} from "echarts/components";
import { CanvasRenderer } from "echarts/renderers";

echarts.use([
  BarChart,
  LineChart,
  // The Cabinet heatmap is a custom series (per-cell computed colors); the
  // built-in heatmap series hard-requires a visualMap component instead.
  CustomChart,
  GridComponent,
  TooltipComponent,
  LegendComponent,
  DataZoomComponent,
  BrushComponent,
  ToolboxComponent,
  CanvasRenderer,
]);

export { echarts };
export type { EChartsCoreOption } from "echarts/core";
