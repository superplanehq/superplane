export interface PrometheusAlertPayload {
  status?: string;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  startsAt?: string;
  endsAt?: string;
  value?: string;
  generatorURL?: string;
  fingerprint?: string;
  receiver?: string;
  groupKey?: string;
  groupLabels?: Record<string, string>;
  commonLabels?: Record<string, string>;
  commonAnnotations?: Record<string, string>;
  externalURL?: string;
}

export interface OnAlertConfiguration {
  statuses?: string[];
  alertNames?: string[];
}

export interface OnAlertMetadata {
  webhookUrl?: string;
  webhookAuthEnabled?: boolean;
}

export interface GetAlertConfiguration {
  alertName?: string;
  state?: string;
}

export interface CreateSilenceConfiguration {
  matchers?: SilenceMatcherConfiguration[];
  duration?: string;
  createdBy?: string;
  comment?: string;
}

export interface SilenceMatcherConfiguration {
  name?: string;
  value?: string;
  isRegex?: boolean;
  isEqual?: boolean;
}

export interface ExpireSilenceConfiguration {
  silenceID?: string;
}

export interface GetSilenceConfiguration {
  silenceID?: string;
}

export interface QueryConfiguration {
  query?: string;
}

export interface QueryRangeConfiguration {
  query?: string;
  start?: string;
  end?: string;
  step?: string;
}

export interface PrometheusSilencePayload {
  silenceID?: string;
  matchers?: SilenceMatcherPayload[];
  startsAt?: string;
  endsAt?: string;
  createdBy?: string;
  comment?: string;
  state?: string;
}

export interface SilenceMatcherPayload {
  name?: string;
  value?: string;
  isRegex?: boolean;
  isEqual?: boolean;
}

export interface PrometheusQueryResultPayload {
  query?: string;
  resultType?: string;
  result?: any[];
  start?: string;
  end?: string;
  step?: string;
}
