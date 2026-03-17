export interface QueryPrometheusConfiguration {
  query: string;
  dataset: string;
  type: "instant" | "range";
  start?: string;
  end?: string;
  step?: string;
}

export interface SendLogEventConfiguration {
  body: string;
  severityText?: string;
  dataset?: string;
  attributes?: Record<string, string>;
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

export interface GetHttpSyntheticCheckConfiguration {
  checkId: string;
  dataset: string;
}

export interface GetHttpSyntheticCheckNodeMetadata {
  checkName?: string;
}

export interface CheckRuleNodeMetadata {
  checkRule?: string;
  checkRuleName?: string;
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

export interface CheckRuleConfiguration {
  name: string;
  expression: string;
  dataset: string;
  thresholds?: { degraded?: number; critical?: number };
  summary?: string;
  description?: string;
  interval?: string;
  for?: string;
  keepFiringFor?: string;
  labels?: Array<{ key: string; value: string }>;
  annotations?: Array<{ key: string; value: string }>;
  enabled?: boolean;
}

export interface GetCheckRuleConfiguration {
  checkRule: string;
  dataset: string;
}

export interface UpdateCheckRuleConfiguration extends CheckRuleConfiguration {
  checkRule: string;
}

export interface DeleteCheckRuleConfiguration {
  checkRule: string;
  dataset: string;
}
