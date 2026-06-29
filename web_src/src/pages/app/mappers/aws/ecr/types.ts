export interface EcrRepository {
  repositoryName?: string;
  repositoryArn?: string;
}

export interface EcrRepositoryMetadata {
  repository?: EcrRepository;
  region?: string;
}

export interface EcrRepositoryConfiguration {
  repository?: string;
  region?: string;
  imageDigest?: string;
  imageTag?: string;
}

export type EcrTriggerMetadata = EcrRepositoryMetadata;
export type EcrTriggerConfiguration = EcrRepositoryConfiguration;

export interface EcrImageDetail {
  registryId?: string;
  repositoryName?: string;
  imageDigest?: string;
  imageTags?: string[];
  imageSizeInBytes?: number;
  imagePushedAt?: string;
  imageManifestMediaType?: string;
  artifactMediaType?: string;
}

export interface EcrImageScanStatus {
  status?: string;
  description?: string;
}

export interface EcrImageScanFindingAttribute {
  key?: string;
  value?: string;
}

export interface EcrImageScanFinding {
  name?: string;
  description?: string;
  uri?: string;
  severity?: string;
  attributes?: EcrImageScanFindingAttribute[];
}

export interface EcrImageScanFindings {
  findings?: EcrImageScanFinding[];
  imageScanCompletedAt?: string;
  vulnerabilitySourceUpdatedAt?: string;
  findingSeverityCounts?: Record<string, number>;
}

export interface EcrImageScanFindingsResponse {
  imageScanFindings?: EcrImageScanFindings;
  registryId?: string;
  repositoryName?: string;
  imageId?: Record<string, string>;
  imageScanStatus?: EcrImageScanStatus;
}

export interface EcrImageScanDetail {
  "scan-status"?: string;
  "repository-name"?: string;
  "image-digest"?: string;
  "image-tags"?: string[];
  "finding-severity-counts"?: {
    CRITICAL?: number;
    HIGH?: number;
    MEDIUM?: number;
    LOW?: number;
    UNDEFINED?: number;
  };
}

export interface EcrImagePushDetail {
  "repository-name"?: string;
  "image-tag"?: string;
  "image-digest"?: string;
  "action-type"?: string;
  result?: string;
  "repository-arn"?: string;
}

export interface EcrEventBase {
  account?: string;
  region?: string;
  time?: string;
  "detail-type"?: string;
  detail?: EcrImageScanDetail | EcrImagePushDetail;
}

export interface EcrImageScanEvent extends EcrEventBase {
  detail?: EcrImageScanDetail;
}

export interface EcrImagePushEvent extends EcrEventBase {
  detail?: EcrImagePushDetail;
}
