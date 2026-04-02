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

export interface DashboardSummary {
  uid?: string;
  title?: string;
  url?: string;
  folderTitle?: string;
  folderUid?: string;
  tags?: string[];
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
  folderUid?: string;
  tags?: string[];
  panels?: PanelSummary[];
}

export interface DashboardNodeMetadata {
  dashboardTitle?: string;
}

export interface SearchDashboardsOutput {
  dashboards?: DashboardSummary[];
}

export interface CreateDashboardShareLinkOutput {
  url?: string;
  dashboardTitle?: string;
  dashboardUid?: string;
}

export interface RenderPanelOutput {
  url?: string;
  dashboardUid?: string;
  panelId?: number;
}
