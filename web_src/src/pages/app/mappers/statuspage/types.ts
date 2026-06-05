export interface StatuspageNodeMetadata {
  pageName?: string;
  componentNames?: string[];
  incidentName?: string;
}

export interface StatuspageIncidentUpdate {
  id?: string;
  status?: string;
  body?: string;
  created_at?: string;
}

export interface StatuspageIncident {
  id?: string;
  name?: string;
  status?: string;
  impact?: string;
  shortlink?: string;
  created_at?: string;
  updated_at?: string;
  page_id?: string;
  incident_updates?: StatuspageIncidentUpdate[];
}
