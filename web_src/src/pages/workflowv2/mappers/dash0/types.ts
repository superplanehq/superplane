export interface QueryPrometheusConfiguration {
  query: string;
  dataset: string;
  type: "instant" | "range";
  start?: string;
  end?: string;
  step?: string;
}

export interface QueryPrometheusMetadata {
  // No metadata needed initially
}

export interface ListIssuesConfiguration {
  checkRules?: string[];
}

export interface OnAlertEventData {
  eventType?: string;
  checkId?: string;
  checkName?: string;
  severity?: string;
  labels?: Record<string, unknown>;
  summary?: string;
  description?: string;
  timestamp?: string;
  event?: Record<string, unknown>;
}

export interface SendLogEventRecordConfiguration {
  message: string;
  severity?: string;
  timestamp?: string;
  attributes?: Record<string, unknown>;
}

export interface SendLogEventConfiguration {
  serviceName?: string;
  records?: SendLogEventRecordConfiguration[];
}

export interface GetCheckDetailsConfiguration {
  checkId?: string;
  includeHistory?: boolean;
}

export interface UpsertSyntheticCheckConfiguration {
  originOrId?: string;
  spec?: string;
}

export interface UpsertCheckRuleConfiguration {
  originOrId?: string;
  spec?: string;
}

export interface PrometheusResponse {
  status: string;
  data: {
    resultType: string;
    result: Array<{
      metric: Record<string, string>;
      value?: [number, string];
      values?: Array<[number, string]>;
    }>;
  };
}
