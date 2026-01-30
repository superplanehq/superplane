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
  title?: string;
  summary?: string;
  status?: string;
  severity?: Severity;
  started_at?: string;
  resolved_at?: string;
  mitigated_at?: string;
  url?: string;
}
