export interface CloudWatchAlarmState {
  value?: string;
  reason?: string;
  timestamp?: string;
}

export interface CloudWatchAlarmDetail {
  alarmName?: string;
  state?: CloudWatchAlarmState;
  previousState?: CloudWatchAlarmState;
}

export interface CloudWatchAlarmEvent {
  account?: string;
  region?: string;
  time?: string;
  "detail-type"?: string;
  detail?: CloudWatchAlarmDetail;
}

export interface CloudWatchMetricDataMessage {
  code?: string;
  value?: string;
}

export interface CloudWatchMetricDataResult {
  id?: string;
  label?: string;
  statusCode?: string;
  timestamps?: string[];
  values?: number[];
  messages?: CloudWatchMetricDataMessage[];
}

export interface CloudWatchMetricsInsightsOutput {
  region?: string;
  query?: string;
  startTime?: string;
  endTime?: string;
  scanBy?: string;
  maxDatapoints?: number;
  requestId?: string;
  results?: CloudWatchMetricDataResult[];
  messages?: CloudWatchMetricDataMessage[];
}
