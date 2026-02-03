export interface CodeArtifactTriggerConfiguration {
  region?: string;
  domainName?: string;
  domainOwner?: string;
  repositoryName?: string;
  packageFormat?: string;
  packageNamespace?: string | null;
  packageName?: string;
  packageVersion?: string;
  packageVersionState?: string;
  operationType?: string;
}

export interface CodeArtifactTriggerMetadata {
  region?: string;
  subscriptionId?: string;
  filters?: CodeArtifactTriggerConfiguration;
}

export interface CodeArtifactPackageVersionChanges {
  assetsAdded?: number;
  assetsRemoved?: number;
  assetsUpdated?: number;
  metadataUpdated?: boolean;
  statusChanged?: boolean;
}

export interface CodeArtifactPackageVersionDetail {
  domainName?: string;
  domainOwner?: string;
  repositoryName?: string;
  repositoryAdministrator?: string;
  packageFormat?: string;
  packageNamespace?: string | null;
  packageName?: string;
  packageVersion?: string;
  packageVersionState?: string;
  packageVersionRevision?: string;
  changes?: CodeArtifactPackageVersionChanges;
  operationType?: string;
  sequenceNumber?: number;
  eventDeduplicationId?: string;
}

export interface CodeArtifactPackageVersionEvent {
  account?: string;
  region?: string;
  time?: string;
  "detail-type"?: string;
  detail?: CodeArtifactPackageVersionDetail;
}

export interface CodeArtifactPackageVersionConfiguration {
  region?: string;
  domain?: string;
  repository?: string;
  package?: string;
  format?: string;
  namespace?: string;
  version?: string;
}

export interface CodeArtifactPackageLicense {
  name?: string;
  url?: string;
}

export interface CodeArtifactPackageVersionDomainEntryPoint {
  externalConnectionName?: string;
  repositoryName?: string;
}

export interface CodeArtifactPackageVersionOrigin {
  domainEntryPoint?: CodeArtifactPackageVersionDomainEntryPoint;
  originType?: string;
}

export interface CodeArtifactPackageVersionDescription {
  displayName?: string;
  format?: string;
  homePage?: string;
  licenses?: CodeArtifactPackageLicense[];
  namespace?: string;
  origin?: CodeArtifactPackageVersionOrigin;
  packageName?: string;
  publishedTime?: number;
  revision?: string;
  sourceCodeRepository?: string;
  status?: string;
  summary?: string;
  version?: string;
}
