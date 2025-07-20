import { QueryRequest, QueryResponse } from '../types';

export class ApiClient {
  private baseUrl: string;

  constructor(baseUrl: string = '') {
    this.baseUrl = baseUrl;
  }

  async query(request: QueryRequest): Promise<QueryResponse> {
    const response = await fetch(`${this.baseUrl}/query`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(request),
    });

    if (!response.ok) {
      throw new Error(`Query failed: ${response.statusText}`);
    }

    return response.json();
  }

  async getEventsByReason(start: Date, end: Date): Promise<QueryResponse> {
    return this.query({
      query: "SELECT reason, COUNT(*) as count FROM $events GROUP BY reason ORDER BY count DESC",
      start: start.toISOString(),
      end: end.toISOString(),
    });
  }

  async getEventsOverTime(start: Date, end: Date, interval: string = '1h'): Promise<QueryResponse> {
    return this.query({
      query: `
        SELECT 
          DATE_TRUNC('${interval}', firstTimestamp) as timestamp,
          COUNT(*) as count,
          type
        FROM $events 
        GROUP BY timestamp, type 
        ORDER BY timestamp
      `,
      start: start.toISOString(),
      end: end.toISOString(),
    });
  }

  async getEventsByNamespace(start: Date, end: Date): Promise<QueryResponse> {
    return this.query({
      query: `
        SELECT 
          involvedObject.namespace as namespace,
          COUNT(*) as count,
          type
        FROM $events 
        WHERE involvedObject.namespace IS NOT NULL
        GROUP BY involvedObject.namespace, type 
        ORDER BY count DESC
        LIMIT 20
      `,
      start: start.toISOString(),
      end: end.toISOString(),
    });
  }

  async getTopErrorEvents(start: Date, end: Date): Promise<QueryResponse> {
    return this.query({
      query: `
        SELECT 
          reason,
          message,
          involvedObject.kind as objectKind,
          involvedObject.name as objectName,
          involvedObject.namespace as namespace,
          COUNT(*) as count
        FROM $events 
        WHERE type = 'Warning'
        GROUP BY reason, message, involvedObject.kind, involvedObject.name, involvedObject.namespace
        ORDER BY count DESC
        LIMIT 50
      `,
      start: start.toISOString(),
      end: end.toISOString(),
    });
  }

  async getClusterHealth(start: Date, end: Date): Promise<QueryResponse> {
    return this.query({
      query: `
        SELECT 
          type,
          COUNT(*) as count
        FROM $events 
        GROUP BY type
      `,
      start: start.toISOString(),
      end: end.toISOString(),
    });
  }
}

export const apiClient = new ApiClient();