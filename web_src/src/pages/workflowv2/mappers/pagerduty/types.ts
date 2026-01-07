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
  assignments?: IncidentAssignment[];
  service?: ResourceRef;
  escalation_policy?: ResourceRef;
};

export interface ResourceRef {
  summary?: string;
  html_url?: string;
}

export interface IncidentAssignment {
  assignee: ResourceRef
}