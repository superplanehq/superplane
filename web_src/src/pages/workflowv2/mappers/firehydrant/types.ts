export interface Incident {
  id?: string;
  name?: string;
  number?: number;
  summary?: string;
  description?: string;
  severity?: string;
  priority?: string;
  current_milestone?: string;
  created_at?: string;
  started_at?: string;
  customer_impact_summary?: string;
}
