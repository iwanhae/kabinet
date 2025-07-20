import React from 'react';
import { Activity, Clock } from 'lucide-react';
import { TimeRange } from '../types';
import { getTimeRanges } from '../utils/date';

interface HeaderProps {
  selectedTimeRange: TimeRange;
  onTimeRangeChange: (timeRange: TimeRange) => void;
}

export const Header: React.FC<HeaderProps> = ({ selectedTimeRange, onTimeRangeChange }) => {
  const timeRanges = getTimeRanges();

  return (
    <header className="bg-white border-b border-gray-200 px-6 py-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-3">
          <div className="flex items-center space-x-2">
            <Activity className="w-8 h-8 text-twitter-blue" />
            <h1 className="text-2xl font-bold text-gray-900">Kube Event Analyzer</h1>
          </div>
          <div className="hidden md:block">
            <span className="px-3 py-1 bg-green-100 text-green-800 text-sm font-medium rounded-full">
              실시간 모니터링
            </span>
          </div>
        </div>
        
        <div className="flex items-center space-x-4">
          <div className="flex items-center space-x-2">
            <Clock className="w-4 h-4 text-gray-500" />
            <select
              value={selectedTimeRange.label}
              onChange={(e) => {
                const selected = timeRanges.find(tr => tr.label === e.target.value);
                if (selected) onTimeRangeChange(selected);
              }}
              className="bg-white border border-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-twitter-blue focus:border-transparent"
            >
              {timeRanges.map((range) => (
                <option key={range.label} value={range.label}>
                  {range.label}
                </option>
              ))}
            </select>
          </div>
        </div>
      </div>
    </header>
  );
};