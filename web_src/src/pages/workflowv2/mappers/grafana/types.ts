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

export interface SilenceMatcher {
  name?: string;
  value?: string;
  /**
   * Used by Create Silence configuration. Mirrors Grafana Alertmanager matcher operators:
   * "=", "!=", "=~", "!~".
   */
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
