export interface Repository {
  name?: string;
  arn?: string;
  domainName?: string;
}

export interface PackageVersionChanges {
  assetsAdded?: number;
  assetsRemoved?: number;
  assetsUpdated?: number;
  metadataUpdated?: boolean;
  statusChanged?: boolean;
}

export interface PackageVersionDetail {
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
  changes?: PackageVersionChanges;
  operationType?: string;
  sequenceNumber?: number;
  eventDeduplicationId?: string;
}

export interface PackageVersionEvent {
  account?: string;
  region?: string;
  time?: string;
  "detail-type"?: string;
  detail?: PackageVersionDetail;
}

export interface PackageLicense {
  name?: string;
  url?: string;
}

export interface PackageVersionAsset {
  hashes?: Record<string, string>;
  name?: string;
  size?: number;
}

export interface PackageVersionDescription {
  displayName?: string;
  format?: string;
  homePage?: string;
  licenses?: PackageLicense[];
  namespace?: string;
  packageName?: string;
  publishedTime?: string;
  revision?: string;
  sourceCodeRepository?: string;
  status?: string;
  summary?: string;
  version?: string;
}

export interface PackageVersionPayload {
  package?: PackageVersionDescription;
  assets?: PackageVersionAsset[];
}
