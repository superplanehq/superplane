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
