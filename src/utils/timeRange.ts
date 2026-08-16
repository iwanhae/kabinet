import dayjs, { type ManipulateType } from "dayjs";

export interface TimeRange {
  from: string;
  to: string;
}

export interface ParsedTimeRange extends TimeRange {
  rawFrom: string;
  rawTo: string;
}

export const parseTime = (timeStr: string): dayjs.Dayjs => {
  if (timeStr === "now") {
    return dayjs();
  }
  if (timeStr.startsWith("now-")) {
    const match = timeStr.match(/^now-(\d+)(s|m|h|d|w)$/);
    if (match) {
      const amount = parseInt(match[1], 10);
      const unit = match[2] as ManipulateType;
      return dayjs().subtract(amount, unit);
    }
  }
  return dayjs(timeStr);
};

export const isRelativeTime = (timeStr: string): boolean => {
  return timeStr.startsWith("now");
};

export const formatTimeRange = (from: string, to: string): ParsedTimeRange => {
  return {
    rawFrom: from,
    rawTo: to,
    from: parseTime(from).format("YYYY-MM-DDTHH:mm:ssZ"),
    to: parseTime(to).format("YYYY-MM-DDTHH:mm:ssZ"),
  };
};
