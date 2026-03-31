export interface OnAlertFiringEventData {
  status?: string;
  title?: string;
  ruleUid?: string;
  ruleId?: number;
  orgId?: number;
  externalURL?: string;
  alerts?: Array<{
    status?: string;
    labels?: Record<string, string>;
    annotations?: Record<string, string>;
    startsAt?: string;
    endsAt?: string;
  }>;
  groupLabels?: Record<string, string>;
  commonLabels?: Record<string, string>;
  commonAnnotations?: Record<string, string>;
}

export interface AlertNamePredicate {
  type: "equals" | "notEquals" | "matches";
  value: string;
}

export interface OnAlertFiringConfiguration {
  alertNames?: AlertNamePredicate[];
}

export interface QueryDataSourceConfiguration {
  dataSourceUid: string;
  query: string;
  timeFrom?: string;
  timeTo?: string;
  timezone?: string;
  format?: string;
}

export interface GrafanaAlertRule {
  uid?: string;
  title?: string;
  folderUID?: string;
  ruleGroup?: string;
  condition?: string;
  noDataState?: string;
  execErrState?: string;
  for?: string;
  isPaused?: boolean;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  data?: Array<Record<string, unknown>>;
}

export interface AlertRuleKeyValuePair {
  key?: string;
  value?: string;
}

export interface CreateAlertRuleConfiguration {
  title?: string;
  folderUID?: string;
  ruleGroup?: string;
  dataSourceUid?: string;
  query?: string;
  lookbackSeconds?: number;
  for?: string;
  noDataState?: string;
  execErrState?: string;
  labels?: AlertRuleKeyValuePair[];
  annotations?: AlertRuleKeyValuePair[];
  isPaused?: boolean;
}

export interface GetAlertRuleConfiguration {
  alertRuleUid: string;
}

export interface UpdateAlertRuleConfiguration extends CreateAlertRuleConfiguration {
  alertRuleUid: string;
}
