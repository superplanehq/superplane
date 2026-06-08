export interface Incident {
  id?: string;
  name?: string;
  summary?: string;
  reference?: string;
  permalink?: string;
  visibility?: string;
  created_at?: string;
  updated_at?: string;
  severity?: { id?: string; name?: string };
  incident_status?: { id?: string; name?: string };
}
