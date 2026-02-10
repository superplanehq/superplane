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
