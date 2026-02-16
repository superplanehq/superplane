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

export interface AssertionItem {
  kind: string;
  severity: string;
  statusCodeOperator?: string;
  statusCodeValue?: string;
  timingType?: string;
  timingOperator?: string;
  timingValue?: string;
  errorTypeValue?: string;
  sslOperator?: string;
  sslDays?: string;
  headerName?: string;
  headerOperator?: string;
  headerValue?: string;
  jsonPath?: string;
  jsonOperator?: string;
  jsonValue?: string;
  textOperator?: string;
  textValue?: string;
}

export interface CreateSyntheticCheckConfiguration {
  name: string;
  url: string;
  method: string;
  dataset: string;
  locations: string[];
  interval: string;
  assertions?: AssertionItem[];
  headers?: Array<{ name: string; value: string }>;
  body?: string;
  strategy?: string;
  retries?: { attempts: number; delay: string };
  followRedirects?: string;
  allowInsecure?: string;
}
