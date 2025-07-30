import { create } from "zustand";
import { useTimeRangeParams } from "../hooks/useUrlParams";
import dayjs, { type ManipulateType } from "dayjs";

export interface TimeRange {
  from: string;
  to: string;
}

interface TimeRangeState extends TimeRange {
  rawFrom: string;
  rawTo: string;
  setTimeRange: (timeRange: TimeRange) => void;
  refreshTimeRange: () => void;
}

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

export const useTimeRangeStore = create<TimeRangeState>((set, get) => ({
  from: parseTime("now-30m").toISOString(),
  to: parseTime("now").toISOString(),
  rawFrom: "now-30m",
  rawTo: "now",
  setTimeRange: ({ from, to }: TimeRange) => {
    set({
      rawFrom: from,
      rawTo: to,
      from: parseTime(from).format("YYYY-MM-DDTHH:mm:ssZ"),
      to: parseTime(to).format("YYYY-MM-DDTHH:mm:ssZ"),
    });
  },
  refreshTimeRange: () => {
    const { rawFrom, rawTo } = get();
    // Only refresh if using relative time
    if (isRelativeTime(rawFrom) || isRelativeTime(rawTo)) {
      set({
        from: parseTime(rawFrom).format("YYYY-MM-DDTHH:mm:ssZ"),
        to: parseTime(rawTo).format("YYYY-MM-DDTHH:mm:ssZ"),
      });
    }
  },
}));

export const useTimeRangeFromUrl = () => {
  const { setTimeRange: setUrlTimeRange } = useTimeRangeParams();
  const { setTimeRange } = useTimeRangeStore();

  return (from: string, to: string) => {
    setTimeRange({ from, to });
    setUrlTimeRange(from, to);
  };
};
