export interface IncidentRecord {
  sys_id?: string;
  number?: string;
  short_description?: string;
  state?: string;
  urgency?: string;
  impact?: string;
  priority?: string;
  category?: string;
  subcategory?: string;
  sys_created_on?: string;
  sys_updated_on?: string;
}

export interface ServiceNowIncident {
  sys_id?: string;
  number?: string;
  short_description?: string;
  description?: string;
  state?: string;
  urgency?: string;
  impact?: string;
  priority?: string;
  category?: string;
  subcategory?: string;
  assignment_group?: ServiceNowReference;
  assigned_to?: ServiceNowReference;
  caller_id?: ServiceNowReference;
  opened_by?: ServiceNowReference;
  sys_created_on?: string;
  sys_updated_on?: string;
  opened_at?: string;
  closed_at?: string;
  resolved_at?: string;
}

export interface ServiceNowReference {
  display_value?: string;
  value?: string;
  link?: string;
}

export interface BaseNodeMetadata {
  instanceUrl?: string;
  assignmentGroup?: { id: string; name: string };
  assignedTo?: { id: string; name: string };
  caller?: { id: string; name: string };
}

export interface CreateIncidentConfiguration {
  shortDescription?: string;
  urgency?: string;
  impact?: string;
  description?: string;
  category?: string;
  subcategory?: string;
  assignmentGroup?: string;
  assignedTo?: string;
  caller?: string;
  state?: string;
  onHoldReason?: string;
  resolutionCode?: string;
  resolutionNotes?: string;
}

export interface GetIncidentsConfiguration {
  assignmentGroup?: string;
  assignedTo?: string;
  caller?: string;
  category?: string;
  subcategory?: string;
  service?: string;
  state?: string;
  urgency?: string;
  impact?: string;
  priority?: string;
  limit?: number;
}

export const STATE_LABELS: Record<string, string> = {
  "1": "New",
  "2": "In Progress",
  "3": "On Hold",
  "6": "Resolved",
  "7": "Closed",
  "8": "Canceled",
};

export const URGENCY_LABELS: Record<string, string> = {
  "1": "High",
  "2": "Medium",
  "3": "Low",
};

export const IMPACT_LABELS: Record<string, string> = {
  "1": "High",
  "2": "Medium",
  "3": "Low",
};
