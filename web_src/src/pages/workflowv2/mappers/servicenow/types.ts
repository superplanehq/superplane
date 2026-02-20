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

export interface BaseNodeMetadata {
  instanceUrl?: string;
  assignmentGroup?: { id: string; name: string };
  assignedTo?: { id: string; name: string };
  caller?: { id: string; name: string };
  incident?: { id: string; name: string };
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
