export interface StatuspageIncident {
  id?: string;
  name?: string;
  status?: string;
  impact?: string;
  impact_override?: string;
  shortlink?: string;
  page_id?: string;
  created_at?: string;
  updated_at?: string;
  resolved_at?: string;
  started_at?: string;
  monitoring_at?: string;
  incident_updates?: IncidentUpdate[];
  components?: StatuspageComponent[];
}

export interface IncidentUpdate {
  id?: string;
  status?: string;
  body?: string;
  created_at?: string;
  updated_at?: string;
  display_at?: string;
}

export interface StatuspageComponent {
  id?: string;
  name?: string;
  status?: string;
}
