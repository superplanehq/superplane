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
  timezone?: string;
  format?: string;
}

export interface QueryLogsConfiguration {
  dataSource: string;
  query: string;
  timeFrom?: string;
  timeTo?: string;
  limit?: number;
}

export interface QueryTracesConfiguration {
  dataSource: string;
  query: string;
  timeFrom?: string;
  timeTo?: string;
}

export interface PanelSummary {
  id?: number;
  title?: string;
  type?: string;
}

export interface DashboardDetails {
  uid?: string;
  title?: string;
  slug?: string;
  url?: string;
  folderTitle?: string;
  folder?: string;
  tags?: string[];
  panels?: PanelSummary[];
}

export interface DashboardNodeMetadata {
  dashboardTitle?: string;
  panelTitle?: string;
  panelLabel?: string;
}

export interface GetDashboardConfiguration {
  dashboard?: string;
}

export interface RenderPanelConfiguration {
  dashboard?: string;
  panel?: string | number;
  width?: number;
  height?: number;
  from?: string;
  to?: string;
}

export interface RenderPanelOutput {
  url?: string;
  dashboard?: string;
  panel?: number;
}

export interface SilenceMatcher {
  name?: string;
  value?: string;
  operator?: string;
  isRegex?: boolean;
  isEqual?: boolean;
}

export interface SilenceStatus {
  state?: string;
}

export interface Silence {
  id?: string;
  status?: SilenceStatus;
  comment?: string;
  createdBy?: string;
  startsAt?: string;
  endsAt?: string;
  updatedAt?: string;
  matchers?: SilenceMatcher[];
  url?: string;
}

export interface ListSilencesOutput {
  silences?: Silence[];
}

export interface CreateSilenceOutput {
  silenceId?: string;
  silenceUrl?: string;
  startsAt?: string;
  endsAt?: string;
}

export interface CreateSilenceConfiguration {
  matchers?: SilenceMatcher[];
  startsAt?: string;
  endsAt?: string;
  comment?: string;
}

export interface DeleteSilenceOutput {
  silenceId?: string;
  deleted?: boolean;
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

export interface GrafanaIncidentLabel {
  key?: string;
  label?: string;
}

export interface GrafanaIncident {
  incidentID?: string;
  incidentId?: string;
  title?: string;
  summary?: string;
  severity?: string;
  status?: string;
  labels?: GrafanaIncidentLabel[];
  isDrill?: boolean;
  createdTime?: string;
  modifiedTime?: string;
  closedTime?: string;
  incidentUrl?: string;
  overviewURL?: string;
}

export interface GrafanaIncidentActivity {
  activityItemID?: string;
  activityId?: string;
  incidentID?: string;
  incidentId?: string;
  activityKind?: string;
  body?: string;
  createdTime?: string;
  eventTime?: string;
  url?: string;
}

export interface GrafanaIncidentNodeMetadata {
  title?: string;
  status?: string;
  severity?: string;
  label?: string;
}

export interface DeclareIncidentConfiguration {
  title?: string;
  severity?: string;
  description?: string;
  labels?: string[];
  status?: string;
  startTime?: string;
  isDrill?: boolean;
}

export interface IncidentSelectionConfiguration {
  incident?: string;
}

export interface UpdateIncidentConfiguration extends IncidentSelectionConfiguration {
  title?: string;
  severity?: string;
  labels?: string[];
  isDrill?: boolean;
}

export interface ResolveIncidentConfiguration extends IncidentSelectionConfiguration {
  summary?: string;
}

export interface AddIncidentActivityConfiguration extends IncidentSelectionConfiguration {
  body?: string;
}
