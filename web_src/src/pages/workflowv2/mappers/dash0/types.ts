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

export interface UpsertSyntheticCheckConfiguration {
  originOrId?: string;
  name?: string;
  enabled?: boolean;
  pluginKind?: string;
  method?: string;
  url?: string;
  headers?: Array<{ key: string; value: string }>;
  requestBody?: string;
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
