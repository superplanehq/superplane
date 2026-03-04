export interface FluxReconciliationEvent {
  involvedObject?: {
    kind?: string;
    namespace?: string;
    name?: string;
    uid?: string;
    apiVersion?: string;
    resourceVersion?: string;
  };
  severity?: string;
  timestamp?: string;
  message?: string;
  reason?: string;
  metadata?: Record<string, string>;
  reportingController?: string;
  reportingInstance?: string;
}

export interface ReconcileSourceOutput {
  kind?: string;
  namespace?: string;
  name?: string;
  annotations?: Record<string, string>;
  resourceVersion?: string;
  lastAppliedRevision?: string;
  lastAttemptedRevision?: string;
}
