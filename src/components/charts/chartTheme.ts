import { useMemo } from "react";
import { useTheme } from "../../contexts/ThemeContext";

/**
 * Canvas charts cannot read CSS custom properties, so this mirrors the
 * palette in src/styles/tokens.css. Keep the two files in sync.
 */
export interface ChartTokens {
  ink: string;
  inkMuted: string;
  inkFaint: string;
  line: string;
  card: string;
  cardInset: string;
  accent: string;
  steel: string;
  steelStrong: string;
  signal: string;
  signalStrong: string;
  fontMono: string;
  fontUi: string;
}

const fonts = {
  fontMono: '"IBM Plex Mono", ui-monospace, monospace',
  fontUi: '"Inter Variable", "Inter", system-ui, sans-serif',
};

const light: ChartTokens = {
  ink: "#0f1419",
  inkMuted: "#536471",
  inkFaint: "#8b98a5",
  line: "#e3e8eb",
  card: "#ffffff",
  cardInset: "#eef1f3",
  accent: "#1d9bf0",
  steel: "#2e93fa",
  steelStrong: "#1d6fc4",
  signal: "#f44336",
  signalStrong: "#d32f2f",
  ...fonts,
};

const dark: ChartTokens = {
  ink: "#e7e9ea",
  inkMuted: "#94a3af",
  inkFaint: "#64707b",
  line: "#29343d",
  card: "#171d23",
  cardInset: "#1f2831",
  accent: "#4dabf5",
  steel: "#5ba7f7",
  steelStrong: "#8bc2fa",
  signal: "#f0564a",
  signalStrong: "#f57e74",
  ...fonts,
};

export const useChartTokens = (): ChartTokens => {
  const { isDarkMode } = useTheme();
  return useMemo(() => (isDarkMode ? dark : light), [isDarkMode]);
};
