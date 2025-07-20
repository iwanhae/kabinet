import React from 'react';
import { TrendingUp, TrendingDown, Minus } from 'lucide-react';
import { MetricData } from '../types';
import clsx from 'clsx';

interface MetricCardProps {
  metric: MetricData;
  icon?: React.ReactNode;
  color?: 'blue' | 'green' | 'red' | 'yellow';
}

export const MetricCard: React.FC<MetricCardProps> = ({ 
  metric, 
  icon, 
  color = 'blue' 
}) => {
  const getTrendIcon = () => {
    if (!metric.trend) return null;
    
    switch (metric.trend) {
      case 'up':
        return <TrendingUp className="w-4 h-4 text-red-500" />;
      case 'down':
        return <TrendingDown className="w-4 h-4 text-green-500" />;
      default:
        return <Minus className="w-4 h-4 text-gray-500" />;
    }
  };

  const getColorClasses = () => {
    switch (color) {
      case 'green':
        return 'text-green-600 bg-green-50 border-green-200';
      case 'red':
        return 'text-red-600 bg-red-50 border-red-200';
      case 'yellow':
        return 'text-yellow-600 bg-yellow-50 border-yellow-200';
      default:
        return 'text-blue-600 bg-blue-50 border-blue-200';
    }
  };

  return (
    <div className="metric-card">
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-3">
          {icon && (
            <div className={clsx('p-2 rounded-lg', getColorClasses())}>
              {icon}
            </div>
          )}
          <div>
            <p className="text-sm font-medium text-gray-600">{metric.name}</p>
            <p className="text-2xl font-bold text-gray-900">
              {metric.value.toLocaleString()}
            </p>
          </div>
        </div>
        
        {(metric.change !== undefined || metric.trend) && (
          <div className="flex items-center space-x-1">
            {getTrendIcon()}
            {metric.change !== undefined && (
              <span className={clsx(
                'text-sm font-medium',
                metric.change > 0 ? 'text-red-600' : 
                metric.change < 0 ? 'text-green-600' : 'text-gray-600'
              )}>
                {metric.change > 0 ? '+' : ''}{metric.change}%
              </span>
            )}
          </div>
        )}
      </div>
    </div>
  );
};