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
  dashboardUID?: string;
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
  dashboardUID?: string;
  panel?: string;
  text?: string;
  tags?: string[];
  from?: string;
  to?: string;
  limit?: number;
}

export interface ListAnnotationsOutput {
  annotations: Annotation[];
}

export interface DeleteAnnotationConfiguration {
  annotationId?: string;
}

export interface DeleteAnnotationOutput {
  id: number;
  deleted: boolean;
}

export interface AnnotationNodeMetadata {
  dashboardTitle?: string;
  annotationLabel?: string;
}
