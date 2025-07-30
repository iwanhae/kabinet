import { create } from "zustand";
import { useLocation } from "wouter";

export interface TimeRange {
  from: string;
  to: string;
}

interface TimeRangeState extends TimeRange {
  setTimeRange: (timeRange: TimeRange) => void;
}

export const useTimeRangeStore = create<TimeRangeState>((set) => ({
  from: "now-30m",
  to: "now",
  setTimeRange: (timeRange) => set(timeRange),
}));

export const useTimeRangeFromUrl = () => {
  const [, setLocation] = useLocation();
  const { setTimeRange } = useTimeRangeStore();

  return (from: string, to: string) => {
    setTimeRange({ from, to });
    const currentPath = window.location.pathname;
    setLocation(
      `${currentPath}?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`,
    );
  };
};
