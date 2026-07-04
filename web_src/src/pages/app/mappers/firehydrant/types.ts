export interface Incident {
  id?: string;
  name?: string;
  number?: number;
  summary?: string;
  description?: string;
  severity?: string;
  priority?: string;
  current_milestone?: string;
  incident_url?: string;
  customer_impact_summary?: string;
}
