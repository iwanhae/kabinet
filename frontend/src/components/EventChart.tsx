import React from 'react';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts';
import { ChartData } from '../types';
import { formatDateShort } from '../utils/date';

interface EventChartProps {
  data: ChartData[];
  title: string;
}

export const EventChart: React.FC<EventChartProps> = ({ data, title }) => {
  const formatTooltipLabel = (label: string) => {
    return formatDateShort(new Date(label));
  };

  const formatXAxisLabel = (tickItem: string) => {
    return formatDateShort(new Date(tickItem));
  };

  return (
    <div className="card">
      <h3 className="text-lg font-semibold text-gray-900 mb-4">{title}</h3>
      <div style={{ width: '100%', height: '300px' }}>
        <ResponsiveContainer>
          <LineChart data={data}>
            <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
            <XAxis 
              dataKey="timestamp" 
              tickFormatter={formatXAxisLabel}
              stroke="#6b7280"
              fontSize={12}
            />
            <YAxis stroke="#6b7280" fontSize={12} />
            <Tooltip
              labelFormatter={formatTooltipLabel}
              contentStyle={{
                backgroundColor: 'white',
                border: '1px solid #e5e7eb',
                borderRadius: '8px',
                boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.1)',
              }}
            />
            <Legend />
            <Line
              type="monotone"
              dataKey="count"
              stroke="#1DA1F2"
              strokeWidth={2}
              dot={{ fill: '#1DA1F2', strokeWidth: 2, r: 4 }}
              activeDot={{ r: 6, stroke: '#1DA1F2', strokeWidth: 2 }}
            />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
};