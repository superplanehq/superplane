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
