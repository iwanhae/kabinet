import { useState, useEffect } from 'react';
import { apiClient } from '../utils/api';
import { QueryResponse, TimeRange } from '../types';

export const useEventData = (timeRange: TimeRange) => {
  const [data, setData] = useState<{
    eventsByReason: QueryResponse | null;
    eventsOverTime: QueryResponse | null;
    eventsByNamespace: QueryResponse | null;
    topErrorEvents: QueryResponse | null;
    clusterHealth: QueryResponse | null;
  }>({
    eventsByReason: null,
    eventsOverTime: null,
    eventsByNamespace: null,
    topErrorEvents: null,
    clusterHealth: null,
  });
  
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      setLoading(true);
      setError(null);
      
      try {
        const [
          eventsByReason,
          eventsOverTime,
          eventsByNamespace,
          topErrorEvents,
          clusterHealth
        ] = await Promise.all([
          apiClient.getEventsByReason(timeRange.start, timeRange.end),
          apiClient.getEventsOverTime(timeRange.start, timeRange.end),
          apiClient.getEventsByNamespace(timeRange.start, timeRange.end),
          apiClient.getTopErrorEvents(timeRange.start, timeRange.end),
          apiClient.getClusterHealth(timeRange.start, timeRange.end),
        ]);

        setData({
          eventsByReason,
          eventsOverTime,
          eventsByNamespace,
          topErrorEvents,
          clusterHealth,
        });
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch data');
        console.error('Error fetching event data:', err);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [timeRange]);

  return { data, loading, error };
};