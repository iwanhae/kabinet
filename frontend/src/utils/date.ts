import { format, subHours, subDays, subWeeks } from 'date-fns';
import { TimeRange } from '../types';

export const formatDate = (date: Date): string => {
  return format(date, 'yyyy-MM-dd HH:mm:ss');
};

export const formatDateShort = (date: Date): string => {
  return format(date, 'MM/dd HH:mm');
};

export const getTimeRanges = (): TimeRange[] => {
  const now = new Date();
  
  return [
    {
      start: subHours(now, 1),
      end: now,
      label: '지난 1시간'
    },
    {
      start: subHours(now, 6),
      end: now,
      label: '지난 6시간'
    },
    {
      start: subDays(now, 1),
      end: now,
      label: '지난 24시간'
    },
    {
      start: subDays(now, 3),
      end: now,
      label: '지난 3일'
    },
    {
      start: subWeeks(now, 1),
      end: now,
      label: '지난 1주일'
    }
  ];
};