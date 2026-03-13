export interface DropletNodeMetadata {
  dropletId?: string;
  dropletName?: string;
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
