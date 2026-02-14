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

export interface CheckRuleKeyValueConfiguration {
  key: string;
  value: string;
}

export interface UpsertCheckRuleConfiguration {
  originOrId?: string;
  name?: string;
  expression?: string;
  for?: string;
  interval?: string;
  keepFiringFor?: string;
  labels?: CheckRuleKeyValueConfiguration[];
  annotations?: CheckRuleKeyValueConfiguration[];
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
