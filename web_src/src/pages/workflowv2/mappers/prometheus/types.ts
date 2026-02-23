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

export interface PrometheusSilencePayload {
  silenceID?: string;
  status?: string;
  matchers?: PrometheusMatcher[];
  startsAt?: string;
  endsAt?: string;
  createdBy?: string;
  comment?: string;
}

export interface PrometheusMatcher {
  name?: string;
  value?: string;
  isRegex?: boolean;
  isEqual?: boolean;
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
  matchers?: PrometheusMatcher[];
  duration?: string;
  createdBy?: string;
  comment?: string;
}

export interface CreateSilenceNodeMetadata {
  silenceID?: string;
}

export interface ExpireSilenceConfiguration {
  silence?: string;
  silenceID?: string;
}

export interface ExpireSilenceNodeMetadata {
  silenceID?: string;
}
