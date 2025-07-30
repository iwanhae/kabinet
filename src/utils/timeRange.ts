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
    const match = timeStr.match(/^now-(\d+)(m|h|d)$/);
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

/**
 * 상대 시간 범위를 현재 시간 기준으로 새로고침
 */
export const refreshRelativeTimeRange = (
  from: string,
  to: string,
): ParsedTimeRange => {
  // 상대 시간인 경우에만 새로고침
  if (isRelativeTime(from) || isRelativeTime(to)) {
    return formatTimeRange(from, to);
  }
  // 절대 시간인 경우 그대로 반환
  return {
    rawFrom: from,
    rawTo: to,
    from: from,
    to: to,
  };
};
