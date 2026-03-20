export interface DropletNodeMetadata {
  dropletId?: string;
  dropletName?: string;
}

export interface SnapshotNodeMetadata {
  snapshotId?: string;
  snapshotName?: string;
}

export interface GetDropletConfiguration {
  dropletId: string;
}

export interface DeleteDropletConfiguration {
  dropletId: string;
}

export interface ManageDropletPowerConfiguration {
  dropletId: string;
  operation: string;
}

export interface DeleteSnapshotConfiguration {
  snapshot: string;
}

export interface DNSRecordConfiguration {
  domain?: string;
  type?: string;
  name?: string;
  data?: string;
  recordId?: string;
}

export interface DNSRecordNodeMetadata {
  recordId?: number;
  recordName?: string;
}

export interface LBNodeMetadata {
  lbId?: string;
  lbName?: string;
}

export interface DeleteLoadBalancerConfiguration {
  loadBalancerID: string;
}

export interface AssignReservedIPConfiguration {
  reservedIP: string;
  action: string;
  droplet?: string;
}

export interface CreateLoadBalancerConfiguration {
  name: string;
  region: string;
}

export interface AlertPolicyNodeMetadata {
  policyUuid?: string;
  policyDesc?: string;
  scopedDroplets?: { dropletId: string; dropletName: string }[];
}

export interface CreateAlertPolicyConfiguration {
  description?: string;
  type?: string;
  compare?: string;
  value?: number;
  window?: string;
}

export interface GetAlertPolicyConfiguration {
  alertPolicy: string;
}

export interface DeleteAlertPolicyConfiguration {
  alertPolicy: string;
}

export interface UpdateAlertPolicyConfiguration {
  alertPolicy: string;
  description?: string;
  type?: string;
  compare?: string;
  value?: number;
  window?: string;
}

export interface GetDropletMetricsConfiguration {
  droplet: string;
  lookbackPeriod: string;
}

interface AlertPolicySlackDetails {
  channel?: string;
  url?: string;
}

interface AlertPolicyNotifications {
  email?: string[];
  slack?: AlertPolicySlackDetails[];
}

export interface AlertPolicyOutput {
  uuid?: string;
  description?: string;
  type?: string;
  compare?: string;
  value?: number;
  window?: string;
  enabled?: boolean;
  alerts?: AlertPolicyNotifications;
}

export interface DeleteAlertPolicyOutput {
  alertPolicyUuid?: string;
}

export interface GetDropletMetricsOutput {
  dropletId?: string;
  lookbackPeriod?: string;
  start?: string;
  end?: string;
  avgCpuUsagePercent?: number;
  avgMemoryUsagePercent?: number;
  avgPublicOutboundBandwidthMbps?: number;
  avgPublicInboundBandwidthMbps?: number;
}
