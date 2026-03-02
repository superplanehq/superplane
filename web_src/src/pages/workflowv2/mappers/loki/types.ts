export interface PushLogsConfiguration {
  labels?: string;
  message?: string;
}

export interface PushLogsPayload {
  labels?: Record<string, string>;
  message?: string;
}

export interface PushLogsNodeMetadata {
  labels?: string;
}

export interface QueryLogsConfiguration {
  query?: string;
  start?: string;
  end?: string;
  limit?: string;
}

export interface QueryLogsNodeMetadata {
  query?: string;
}

export interface QueryLogsPayload {
  resultType?: string;
  result?: any[];
}
