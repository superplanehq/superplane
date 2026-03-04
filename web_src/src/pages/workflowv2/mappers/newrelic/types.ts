export interface NewRelicIssuePayload {
  issueId?: string;
  issueUrl?: string;
  title?: string;
  priority?: string;
  state?: string;
  policyName?: string;
  conditionName?: string;
  accountId?: number | string;
  createdAt?: number;
  updatedAt?: number;
  sources?: string[];
}

export interface NewRelicMetricPayload {
  metricName?: string;
  metricType?: string;
  value?: number;
  timestamp?: number;
}

export interface NewRelicNRQLResultPayload {
  query?: string;
  results?: any[];
}

export interface OnIssueConfiguration {
  statuses?: string[];
  priorities?: string[];
}

export interface ReportMetricConfiguration {
  metricName?: string;
  metricType?: string;
  value?: number;
  attributes?: Record<string, any>;
  timestamp?: number;
}

export interface RunNRQLQueryConfiguration {
  query?: string;
  timeout?: number;
}
