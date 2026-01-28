export interface MonitorAlert {
  id?: string;
  event_type?: string;
  alert_type?: string;
  alert_transition?: string;
  hostname?: string;
  monitor_id?: number;
  monitor_name?: string;
  priority?: string;
  tags?: string[];
  title?: string;
  date?: number;
  body?: string;
  org?: OrgInfo;
}

export interface OrgInfo {
  id?: number;
  name?: string;
}

export interface DatadogEvent {
  id?: number;
  title?: string;
  text?: string;
  date_happened?: number;
  alert_type?: string;
  priority?: string;
  tags?: string[];
  url?: string;
}
