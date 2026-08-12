export interface CloudWatchDimension {
  name?: string;
  value?: string;
}

/** Alarm payload emitted by the create and update alarm components. */
export interface CloudWatchAlarm {
  alarmName?: string;
  alarmArn?: string;
  alarmDescription?: string;
  alarmConfigurationUpdatedTimestamp?: string;
  actionsEnabled?: boolean;
  namespace?: string;
  metricName?: string;
  dimensions?: CloudWatchDimension[];
  statistic?: string;
  extendedStatistic?: string;
  unit?: string;
  period?: number;
  evaluationPeriods?: number;
  datapointsToAlarm?: number;
  threshold?: number;
  thresholdMetricId?: string;
  comparisonOperator?: string;
  treatMissingData?: string;
  evaluateLowSampleCountPercentile?: string;
  stateValue?: string;
  stateReason?: string;
  stateReasonData?: string;
  stateUpdatedTimestamp?: string;
  stateTransitionedTimestamp?: string;
  alarmActions?: string[];
  okActions?: string[];
  insufficientDataActions?: string[];
  region?: string;
  consoleUrl?: string;
}

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
