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
  sequential_id?: number;
  title?: string;
  slug?: string;
  summary?: string;
  status?: string;
  severity?: string;
  started_at?: string;
  resolved_at?: string;
  mitigated_at?: string;
  updated_at?: string;
  url?: string;
  user?: { id?: string; full_name?: string; email?: string };
  started_by?: { id?: string; full_name?: string; email?: string };
  services?: Array<{ id?: string; name?: string; slug?: string }>;
  groups?: Array<{ id?: string; name?: string; slug?: string }>;
  events?: Array<Record<string, unknown>>;
  action_items?: Array<{ id?: string; summary?: string; status?: string }>;
}

export interface IncidentEvent {
  id?: string;
  event?: string;
  visibility?: string;
  occurred_at?: string;
  created_at?: string;
}
