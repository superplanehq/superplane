export interface BaseNodeMetadata {
  service?: {
    id: string;
    name: string;
    html_url: string;
  };
}

export interface Agent {
  html_url?: string;
  summary?: string;
}

export interface Incident {
  id?: string;
  incident_key?: string;
  incident_number?: string;
  title?: string;
  urgency?: string;
  status?: string;
  html_url?: string;
  created_at?: string;
  updated_at?: string;
  resolved_at?: string;
  last_status_change_at?: string;
  assignments?: IncidentAssignment[];
  service?: ResourceRef;
  escalation_policy?: ResourceRef;
  priority?: ResourceRef;
}

export interface ResourceRef {
  summary?: string;
  html_url?: string;
}

export interface IncidentAssignment {
  assignee: ResourceRef;
}

export interface ListIncidentsConfiguration {
  services?: string[];
}

export interface ListIncidentsResponse {
  incidents: Incident[];
  total: number;
}

export interface Note {
  id?: string;
  content?: string;
  created_at?: string;
  user?: ResourceRef;
  channel?: NoteChannel;
}

export interface NoteChannel {
  type?: string;
}

export interface ListNotesConfiguration {
  incidentId?: string;
}

export interface ListNotesResponse {
  notes: Note[];
  total: number;
}

export interface LogEntry {
  id?: string;
  type?: string;
  summary?: string;
  created_at?: string;
  agent?: ResourceRef;
  channel?: LogChannel;
}

export interface LogChannel {
  type?: string;
}

export interface ListLogEntriesConfiguration {
  incidentId?: string;
  limit?: number;
}

export interface ListLogEntriesResponse {
  log_entries: LogEntry[];
  total: number;
}
