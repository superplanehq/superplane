export interface BaseNodeMetadata {
  service?: {
    id: string;
    name: string;
    slug: string;
    description: string;
  };
}

export interface Incident {
  id?: string;
  title?: string;
  summary?: string;
  status?: string;
  severity?: string;
  started_at?: string;
  resolved_at?: string;
  mitigated_at?: string;
  url?: string;
}
