export interface EcrRepository {
  repositoryName?: string;
  repositoryArn?: string;
}

export interface EcrTriggerMetadata {
  repository?: EcrRepository;
}

export interface EcrTriggerConfiguration {
  repository?: string;
}

export interface EcrImageScanDetail {
  "scan-status"?: string;
  "repository-name"?: string;
  "image-digest"?: string;
  "image-tags"?: string[];
  "finding-severity-counts"?: Record<string, number>;
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
  detail?: Record<string, unknown>;
}

export interface EcrImageScanEvent extends EcrEventBase {
  detail?: EcrImageScanDetail;
}

export interface EcrImagePushEvent extends EcrEventBase {
  detail?: EcrImagePushDetail;
}
