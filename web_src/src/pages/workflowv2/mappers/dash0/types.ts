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
  operator?: string;
  value?: string;
  type?: string;
  name?: string;
  expression?: string;
}

export interface DeleteHttpSyntheticCheckConfiguration {
  checkId: string;
  dataset: string;
}

export interface UpdateHttpSyntheticCheckConfiguration {
  checkId: string;
  name: string;
  dataset: string;
  request: {
    url: string;
    method: string;
    redirects?: string;
    allowInsecure?: string;
    headers?: Array<{ name: string; value: string }>;
    body?: string;
  };
  schedule: {
    interval: string;
    locations: string[];
    strategy?: string;
  };
  assertions?: AssertionItem[];
  retries?: { attempts: number; delay: string };
}

export interface CreateHttpSyntheticCheckConfiguration {
  name: string;
  dataset: string;
  request: {
    url: string;
    method: string;
    redirects?: string;
    allowInsecure?: string;
    headers?: Array<{ name: string; value: string }>;
    body?: string;
  };
  schedule: {
    interval: string;
    locations: string[];
    strategy?: string;
  };
  assertions?: AssertionItem[];
  retries?: { attempts: number; delay: string };
}
