import { Predicate } from "../../utils";

export interface CloudWatchAlarmTriggerConfiguration {
  region?: string;
  alarms?: Predicate[];
}

export interface CloudWatchAlarmTriggerMetadata {
  region?: string;
  subscriptionId?: string;
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
