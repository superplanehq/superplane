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
  dataSource: string;
  query: string;
  timeFrom?: string;
  timeTo?: string;
  format?: string;
}

export interface GrafanaAlertRule {
  uid?: string;
  title?: string;
  folderUID?: string;
  folderTitle?: string;
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

export interface GrafanaAlertRuleSummary {
  uid?: string;
  title?: string;
}

export interface AlertRuleKeyValuePair {
  key?: string;
  value?: string;
}

export interface AlertRuleNodeMetadata {
  alertRuleTitle?: string;
  folderTitle?: string;
}

export interface CreateAlertRuleConfiguration {
  title?: string;
  folder?: string;
  ruleGroup?: string;
  dataSource?: string;
  query?: string;
  lookbackSeconds?: number;
  reducer?: string;
  conditionType?: string;
  threshold?: number;
  threshold2?: number;
  notificationReceiver?: string;
  for?: string;
  noDataState?: string;
  execErrState?: string;
  labels?: AlertRuleKeyValuePair[];
  annotations?: AlertRuleKeyValuePair[];
  isPaused?: boolean;
}

export interface GetAlertRuleConfiguration {
  alertRule: string;
}

export interface UpdateAlertRuleConfiguration extends CreateAlertRuleConfiguration {
  alertRule: string;
}

export interface DeleteAlertRuleConfiguration {
  alertRule: string;
}

export interface DeleteAlertRuleOutput {
  uid?: string;
  title?: string;
  deleted?: boolean;
}

export interface ListAlertRulesConfiguration {
  folder?: string;
  group?: string;
}

export interface ListAlertRulesNodeMetadata {
  folderTitle?: string;
}

export interface ListAlertRulesOutput {
  alertRules?: GrafanaAlertRuleSummary[];
}

export interface Annotation {
  id: number;
  dashboardUID?: string;
  panelId?: number;
  time?: number;
  timeEnd?: number;
  text: string;
  tags?: string[];
  type?: string;
}

export interface CreateAnnotationConfiguration {
  dashboard?: string;
  panel?: string;
  text: string;
  tags?: string[];
  time?: string;
  timeEnd?: string;
}

export interface CreateAnnotationOutput {
  id: number;
  url?: string;
}

export interface ListAnnotationsConfiguration {
  dashboard?: string;
  panel?: string;
  text?: string;
  tags?: string[];
  from?: string;
  to?: string;
  limit?: number;
}

export interface ListAnnotationsOutput {
  annotations: Annotation[];
  from?: string;
  to?: string;
}

export interface DeleteAnnotationConfiguration {
  annotation?: string;
}

export interface DeleteAnnotationOutput {
  id: number;
  deleted: boolean;
}

export interface AnnotationNodeMetadata {
  dashboardTitle?: string;
  annotationLabel?: string;
}
