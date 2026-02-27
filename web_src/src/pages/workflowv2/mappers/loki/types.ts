export interface LokiLogEntry {
  labels?: Record<string, string>;
  message?: string;
  timestamp?: string;
}

export interface LokiQueryResult {
  status?: string;
  result_type?: string;
  result?: LokiStreamResult[];
}

export interface LokiStreamResult {
  stream?: Record<string, string>;
  values?: string[][];
}
