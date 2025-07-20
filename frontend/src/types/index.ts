export interface KubeEvent {
  kind: string;
  apiVersion: string;
  metadata: {
    name: string;
    namespace: string;
    uid: string;
    resourceVersion: string;
    creationTimestamp: string;
  };
  involvedObject: {
    kind: string;
    namespace: string;
    name: string;
    uid: string;
    apiVersion: string;
    resourceVersion: string;
    fieldPath?: string;
  };
  reason: string;
  message: string;
  source: {
    component: string;
    host: string;
  };
  firstTimestamp: string;
  lastTimestamp: string;
  count: number;
  type: string;
  eventTime: string;
  series?: {
    count: number;
    lastObservedTime: string;
  };
  action?: string;
  related?: {
    kind: string;
    namespace: string;
    name: string;
    uid: string;
    apiVersion: string;
    resourceVersion: string;
    fieldPath?: string;
  };
  reportingComponent: string;
  reportingInstance: string;
}

export interface QueryRequest {
  query: string;
  start: string;
  end: string;
}

export interface QueryResponse {
  results: Record<string, any>[];
}

export interface MetricData {
  name: string;
  value: number;
  change?: number;
  trend?: 'up' | 'down' | 'neutral';
}

export interface ChartData {
  timestamp: string;
  count: number;
  [key: string]: any;
}

export interface EventSummary {
  reason: string;
  count: number;
  type: string;
  severity: 'critical' | 'warning' | 'info';
}

export interface TimeRange {
  start: Date;
  end: Date;
  label: string;
}