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

export interface DataSource {
  id?: number;
  uid: string;
  name: string;
  type: string;
  url?: string;
  isDefault?: boolean;
}

export interface GetDataSourceNodeMetadata {
  dataSource?: string;
  dataSourceName?: string;
  dataSourceType?: string;
}

export interface ListDataSourcesOutput {
  dataSources: DataSource[];
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
