import { create } from "zustand";
import { useLocation } from "wouter";
import dayjs, { type ManipulateType } from "dayjs";

export interface TimeRange {
  from: string;
  to: string;
}

interface TimeRangeState extends TimeRange {
  rawFrom: string;
  rawTo: string;
  refreshInterval: number | null;
  setTimeRange: (timeRange: TimeRange) => void;
  setRefreshInterval: (interval: number | null) => void;
}

let refreshTimer: number | null = null;

const parseTime = (timeStr: string): dayjs.Dayjs => {
  if (timeStr === "now") {
    return dayjs();
  }
  if (timeStr.startsWith("now-")) {
    const match = timeStr.match(/^now-(\d+)(m|h|d)$/);
    if (match) {
      const amount = parseInt(match[1], 10);
      const unit = match[2] as ManipulateType;
      return dayjs().subtract(amount, unit);
    }
  }
  return dayjs(timeStr);
};

const isRelativeTime = (timeStr: string): boolean => {
  return timeStr.startsWith("now");
};

export const useTimeRangeStore = create<TimeRangeState>((set, get) => {
  const startRefresh = () => {
    if (refreshTimer) {
      clearInterval(refreshTimer);
      refreshTimer = null;
    }

    const { rawFrom, rawTo, refreshInterval } = get();
    if (refreshInterval && (isRelativeTime(rawFrom) || isRelativeTime(rawTo))) {
      refreshTimer = window.setInterval(() => {
        set({
          from: parseTime(rawFrom).toISOString(),
          to: parseTime(rawTo).toISOString(),
        });
      }, refreshInterval);
    }
  };

  const initialState = {
    from: parseTime("now-30m").toISOString(),
    to: parseTime("now").toISOString(),
    rawFrom: "now-30m",
    rawTo: "now",
    refreshInterval: 15000, // 15 seconds
    setTimeRange: ({ from, to }: TimeRange) => {
      set({
        rawFrom: from,
        rawTo: to,
        from: parseTime(from).toISOString(),
        to: parseTime(to).toISOString(),
      });
      startRefresh();
    },
    setRefreshInterval: (interval: number | null) => {
      set({ refreshInterval: interval });
      startRefresh();
    },
  };

  // Start the refresh interval on initial load
  setTimeout(startRefresh, 0);

  return initialState;
});

export const useTimeRangeFromUrl = () => {
  const [, setLocation] = useLocation();
  const { setTimeRange } = useTimeRangeStore();

  return (from: string, to: string) => {
    setTimeRange({ from, to });
    const currentPath = window.location.pathname;
    setLocation(
      `${currentPath}?from=${encodeURIComponent(from)}&to=${encodeURIComponent(
        to,
      )}`,
    );
  };
};
