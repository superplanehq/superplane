// Invoke Handler response
export interface InvocationResponse {
  service?: string;
  handler?: string;
  status_code?: number;
  response?: any;
  idempotency_key?: string;
}

// Send Handler / Send Delayed Handler response
export interface InvocationSent {
  invocation_id?: string;
  status?: string;
  service?: string;
  handler?: string;
  delay?: string;
}

// Register Deployment response
export interface DeploymentRegistered {
  id?: string;
  uri?: string;
  protocol_type?: string;
  services?: ServiceSummary[];
}

export interface ServiceSummary {
  name?: string;
  revision?: number;
  ty?: string;
  public?: boolean;
  handlers?: HandlerSummary[];
}

export interface HandlerSummary {
  name?: string;
  ty?: string;
}

// Remove Deployment response
export interface DeploymentRemoved {
  deployment_id?: string;
  force?: boolean;
  status?: string;
}

// Get Service response
export interface ServiceDetails {
  name?: string;
  revision?: number;
  ty?: string;
  deployment_id?: string;
  public?: boolean;
  idempotency_retention?: string;
  handlers?: HandlerSummary[];
}

// List Services response
export interface ServicesList {
  services?: ServiceDetails[];
  count?: number;
}

// Cancel/Kill/Purge Invocation response
export interface InvocationAction {
  invocation_id?: string;
  status?: string;
}

// Health Check response
export interface HealthCheckResult {
  healthy?: boolean;
  cluster_health?: any;
  version?: any;
  error?: string;
}
