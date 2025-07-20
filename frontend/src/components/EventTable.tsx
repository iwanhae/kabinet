import React from 'react';
import { AlertTriangle, Info, XCircle } from 'lucide-react';
import { getSeverity, getSeverityColor } from '../utils/severity';
import clsx from 'clsx';

interface EventTableProps {
  events: Array<{
    reason: string;
    message: string;
    objectKind?: string;
    objectName?: string;
    namespace?: string;
    count: number;
    type?: string;
  }>;
  title: string;
}

export const EventTable: React.FC<EventTableProps> = ({ events, title }) => {
  const getSeverityIcon = (severity: 'critical' | 'warning' | 'info') => {
    switch (severity) {
      case 'critical':
        return <XCircle className="w-4 h-4" />;
      case 'warning':
        return <AlertTriangle className="w-4 h-4" />;
      default:
        return <Info className="w-4 h-4" />;
    }
  };

  return (
    <div className="card">
      <h3 className="text-lg font-semibold text-gray-900 mb-4">{title}</h3>
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                심각도
              </th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                이유
              </th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                오브젝트
              </th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                네임스페이스
              </th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                발생 횟수
              </th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                메시지
              </th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {events.map((event, index) => {
              const severity = getSeverity(event.reason, event.type || 'Normal');
              const severityColor = getSeverityColor(severity);
              
              return (
                <tr key={index} className="hover:bg-gray-50">
                  <td className="px-4 py-3 whitespace-nowrap">
                    <div className={clsx('inline-flex items-center px-2 py-1 rounded-full text-xs font-medium border', severityColor)}>
                      {getSeverityIcon(severity)}
                      <span className="ml-1 capitalize">{severity}</span>
                    </div>
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap text-sm font-medium text-gray-900">
                    {event.reason}
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
                    {event.objectKind && event.objectName ? (
                      <span className="font-mono text-xs bg-gray-100 px-2 py-1 rounded">
                        {event.objectKind}/{event.objectName}
                      </span>
                    ) : '-'}
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
                    {event.namespace ? (
                      <span className="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-blue-100 text-blue-800">
                        {event.namespace}
                      </span>
                    ) : '-'}
                  </td>
                  <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-900 font-medium">
                    {event.count.toLocaleString()}
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-500 max-w-md truncate">
                    {event.message}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
      
      {events.length === 0 && (
        <div className="text-center py-8">
          <Info className="w-12 h-12 text-gray-400 mx-auto mb-4" />
          <p className="text-gray-500">표시할 이벤트가 없습니다.</p>
        </div>
      )}
    </div>
  );
};