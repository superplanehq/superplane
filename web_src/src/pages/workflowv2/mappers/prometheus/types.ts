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
