// 이벤트의 전체 구조를 정의하는 타입
export interface EventResult {
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
  eventTime?: string;
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
  reportingComponent?: string;
  reportingInstance?: string;
}
