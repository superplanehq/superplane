export interface BaseNodeMetadata {
  service?: {
    id: string;
    name: string;
    slug: string;
    description: string;
  };
}

export interface Severity {
  id?: string;
  name?: string;
  slug?: string;
  color?: string;
}

export interface Incident {
  id?: string;
  sequential_id?: number;
  title?: string;
  summary?: string;
  slug?: string;
  status?: string;
  severity?: Severity;
  started_at?: string;
  resolved_at?: string;
  mitigated_at?: string;
  resolution_message?: string;
  resolved_by?: { id?: string; name?: string } | string;
  url?: string;
}
