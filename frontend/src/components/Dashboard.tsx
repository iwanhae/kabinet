import React from 'react';
import { 
  Activity, 
  AlertTriangle, 
  CheckCircle, 
  XCircle,
  Server
} from 'lucide-react';
import { useEventData } from '../hooks/useEventData';
import { TimeRange, MetricData } from '../types';
import { MetricCard } from './MetricCard';
import { EventChart } from './EventChart';
import { EventTable } from './EventTable';
import { LoadingSpinner } from './LoadingSpinner';
import { ErrorMessage } from './ErrorMessage';

interface DashboardProps {
  timeRange: TimeRange;
}

export const Dashboard: React.FC<DashboardProps> = ({ timeRange }) => {
  const { data, loading, error } = useEventData(timeRange);

  if (loading) {
    return <LoadingSpinner message="클러스터 이벤트 데이터를 분석하는 중..." />;
  }

  if (error) {
    return <ErrorMessage message={error} />;
  }

  // 메트릭 계산
  const totalEvents = data.clusterHealth?.results.reduce((sum, item) => sum + (item.count || 0), 0) || 0;
  const warningEvents = data.clusterHealth?.results.find(item => item.type === 'Warning')?.count || 0;
  const normalEvents = data.clusterHealth?.results.find(item => item.type === 'Normal')?.count || 0;
  
  const criticalEventReasons = ['FailedScheduling', 'FailedMount', 'BackOff', 'Failed'];
  const criticalEvents = data.eventsByReason?.results.filter(event => 
    criticalEventReasons.some(reason => event.reason?.includes(reason))
  ).reduce((sum, event) => sum + (event.count || 0), 0) || 0;

  const metrics: MetricData[] = [
    {
      name: '총 이벤트',
      value: totalEvents,
      trend: 'neutral'
    },
    {
      name: '정상 이벤트',
      value: normalEvents,
      trend: 'neutral'
    },
    {
      name: '경고 이벤트',
      value: warningEvents,
      trend: warningEvents > 0 ? 'up' : 'neutral'
    },
    {
      name: '심각한 이벤트',
      value: criticalEvents,
      trend: criticalEvents > 0 ? 'up' : 'neutral'
    }
  ];

  // 차트 데이터 변환
  const chartData = data.eventsOverTime?.results.map(item => ({
    timestamp: item.timestamp,
    count: item.count || 0,
    type: item.type
  })) || [];

  // 시간별 집계 데이터 (타입별로 분리하지 않고 총합)
  const aggregatedChartData = chartData.reduce((acc, curr) => {
    const existing = acc.find(item => item.timestamp === curr.timestamp);
    if (existing) {
      existing.count += curr.count;
    } else {
      acc.push({ timestamp: curr.timestamp, count: curr.count });
    }
    return acc;
  }, [] as Array<{ timestamp: string; count: number }>);

  return (
    <div className="space-y-6">
      {/* 메트릭 카드들 */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <MetricCard 
          metric={metrics[0]} 
          icon={<Activity className="w-5 h-5" />}
          color="blue"
        />
        <MetricCard 
          metric={metrics[1]} 
          icon={<CheckCircle className="w-5 h-5" />}
          color="green"
        />
        <MetricCard 
          metric={metrics[2]} 
          icon={<AlertTriangle className="w-5 h-5" />}
          color="yellow"
        />
        <MetricCard 
          metric={metrics[3]} 
          icon={<XCircle className="w-5 h-5" />}
          color="red"
        />
      </div>

      {/* 차트 섹션 */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <EventChart 
          data={aggregatedChartData} 
          title="시간대별 이벤트 발생 추이" 
        />
        
        <div className="card">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">이벤트 유형별 분포</h3>
          <div className="space-y-4">
            {data.eventsByReason?.results.slice(0, 8).map((event, index) => (
              <div key={index} className="flex items-center justify-between">
                <div className="flex items-center space-x-3">
                  <div className="w-3 h-3 rounded-full bg-twitter-blue"></div>
                  <span className="text-sm font-medium text-gray-900">{event.reason}</span>
                </div>
                <span className="text-sm text-gray-600 font-medium">
                  {(event.count || 0).toLocaleString()}
                </span>
              </div>
            )) || []}
          </div>
        </div>
      </div>

      {/* 네임스페이스별 이벤트 */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="card">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">네임스페이스별 이벤트</h3>
          <div className="space-y-3">
            {data.eventsByNamespace?.results.slice(0, 10).map((ns, index) => (
              <div key={index} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                <div className="flex items-center space-x-3">
                  <Server className="w-4 h-4 text-twitter-blue" />
                  <span className="text-sm font-medium text-gray-900">{ns.namespace || 'default'}</span>
                  <span className={`text-xs px-2 py-1 rounded-full ${
                    ns.type === 'Warning' ? 'bg-red-100 text-red-800' : 'bg-green-100 text-green-800'
                  }`}>
                    {ns.type}
                  </span>
                </div>
                <span className="text-sm text-gray-600 font-medium">
                  {(ns.count || 0).toLocaleString()}
                </span>
              </div>
            )) || []}
          </div>
        </div>

        <div className="card">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">클러스터 상태 요약</h3>
          <div className="space-y-4">
            <div className="flex items-center justify-between p-4 bg-green-50 border border-green-200 rounded-lg">
              <div className="flex items-center space-x-3">
                <CheckCircle className="w-5 h-5 text-green-600" />
                <span className="text-sm font-medium text-green-900">정상 이벤트</span>
              </div>
              <span className="text-lg font-bold text-green-900">{normalEvents.toLocaleString()}</span>
            </div>
            
            <div className="flex items-center justify-between p-4 bg-yellow-50 border border-yellow-200 rounded-lg">
              <div className="flex items-center space-x-3">
                <AlertTriangle className="w-5 h-5 text-yellow-600" />
                <span className="text-sm font-medium text-yellow-900">경고 이벤트</span>
              </div>
              <span className="text-lg font-bold text-yellow-900">{warningEvents.toLocaleString()}</span>
            </div>

            <div className="flex items-center justify-between p-4 bg-red-50 border border-red-200 rounded-lg">
              <div className="flex items-center space-x-3">
                <XCircle className="w-5 h-5 text-red-600" />
                <span className="text-sm font-medium text-red-900">심각한 이벤트</span>
              </div>
              <span className="text-lg font-bold text-red-900">{criticalEvents.toLocaleString()}</span>
            </div>
          </div>
        </div>
      </div>

      {/* 상위 오류 이벤트 테이블 */}
      <EventTable 
        events={data.topErrorEvents?.results.map(event => ({
          reason: event.reason || '',
          message: event.message || '',
          objectKind: event.objectKind,
          objectName: event.objectName,
          namespace: event.namespace,
          count: event.count || 0,
          type: 'Warning'
        })) || []}
        title="주요 오류 이벤트"
      />
    </div>
  );
};