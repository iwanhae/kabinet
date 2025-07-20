// import { EventSummary } from '../types';

export const getSeverity = (reason: string, type: string): 'critical' | 'warning' | 'info' => {
  const criticalReasons = [
    'FailedScheduling',
    'FailedMount',
    'FailedAttachVolume',
    'FailedDetachVolume',
    'NodeNotReady',
    'NodeNotSchedulable',
    'PodEvicted',
    'BackOff',
    'Failed'
  ];

  const warningReasons = [
    'Unhealthy',
    'ProbeWarning',
    'NodeSysctlChange',
    'ContainerGCFailed',
    'ImageGCFailed',
    'FailedCreatePodSandBox',
    'NetworkNotReady'
  ];

  if (type === 'Warning' && criticalReasons.some(cr => reason.includes(cr))) {
    return 'critical';
  }

  if (type === 'Warning' || warningReasons.some(wr => reason.includes(wr))) {
    return 'warning';
  }

  return 'info';
};

export const getSeverityColor = (severity: 'critical' | 'warning' | 'info'): string => {
  switch (severity) {
    case 'critical':
      return 'text-red-600 bg-red-50 border-red-200';
    case 'warning':
      return 'text-yellow-600 bg-yellow-50 border-yellow-200';
    case 'info':
      return 'text-blue-600 bg-blue-50 border-blue-200';
    default:
      return 'text-gray-600 bg-gray-50 border-gray-200';
  }
};