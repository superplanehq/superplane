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

export interface GetObjectConfiguration {
  bucket?: string;
  filePath?: string;
  includeBody?: boolean;
}

export interface GetObjectOutput {
  bucket?: string;
  filePath?: string;
  endpoint?: string;
  contentType?: string;
  size?: string;
  lastModified?: string;
  eTag?: string;
  metadata?: Record<string, string>;
  tags?: Record<string, string>;
  body?: string;
}

export interface CopyObjectConfiguration {
  sourceBucket?: string;
  sourceFilePath?: string;
  destinationBucket?: string;
  destinationFilePath?: string;
  deleteSource?: boolean;
}

export interface CopyObjectOutput {
  sourceBucket?: string;
  sourceFilePath?: string;
  destinationBucket?: string;
  destinationFilePath?: string;
  endpoint?: string;
  eTag?: string;
  moved?: boolean;
}

export interface DeleteObjectConfiguration {
  bucket?: string;
  filePath?: string;
}

export interface DeleteObjectOutput {
  bucket?: string;
  filePath?: string;
  deleted?: boolean;
}

export interface PutObjectConfiguration {
  bucket?: string;
  filePath?: string;
  body?: string;
  acl?: string;
  metadata?: Record<string, string>;
  tags?: Record<string, string>;
}

export interface PutObjectOutput {
  bucket?: string;
  filePath?: string;
  endpoint?: string;
  eTag?: string;
  contentType?: string;
  size?: string;
  metadata?: Record<string, string>;
  tags?: Record<string, string>;
}

export interface AppNodeMetadata {
  appId?: string;
  appName?: string;
}

export interface CreateAppConfiguration {
  name: string;
  region: string;
  sourceProvider?: string;
  gitHubRepo?: string;
  gitHubBranch?: string;
  gitLabRepo?: string;
  gitLabBranch?: string;
  bitbucketRepo?: string;
  bitbucketBranch?: string;
  envVars?: string[];
}

export interface GetAppConfiguration {
  app: string;
}

export interface DeleteAppConfiguration {
  app: string;
}

export interface UpdateAppConfiguration {
  app: string;
  envVars?: string[];
  gitHubBranch?: string;
}

export interface GetGPUDropletConfiguration {
  droplet: string;
}

export interface UpdateGPUDropletConfiguration {
  droplet: string;
  name?: string;
  size?: string;
}

export interface DeleteGPUDropletConfiguration {
  droplet: string;
}

export interface CreateGPUDropletConfiguration {
  name?: string;
  region?: string;
  size?: string;
  imageType?: string;
  oneClickImage?: string;
  baseImage?: string;
}

export interface NetworkV4 {
  ip_address?: string;
  netmask?: string;
  gateway?: string;
  type?: string;
}

export interface DropletRegion {
  name?: string;
  slug?: string;
}

export interface DropletImage {
  name?: string;
  slug?: string;
}

export interface DropletData {
  id?: number;
  name?: string;
  status?: string;
  region?: DropletRegion;
  size_slug?: string;
  image?: DropletImage;
  memory?: number;
  vcpus?: number;
  disk?: number;
  networks?: {
    v4?: NetworkV4[];
  };
  tags?: string[];
}

export interface DeleteGPUDropletResult {
  dropletId?: number;
}
