import { Predicate } from "../../utils";

export interface CodeArtifactRepository {
  name?: string;
  arn?: string;
  domainName?: string;
}

export interface CodeArtifactTriggerConfiguration {
  region?: string;
  repository?: string;
  packages?: Predicate[];
  versions?: Predicate[];
}

export interface CodeArtifactTriggerMetadata {
  region?: string;
  subscriptionId?: string;
  repository?: CodeArtifactRepository;
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

export interface CodeArtifactPackageVersionDescription {
  displayName?: string;
  format?: string;
  homePage?: string;
  licenses?: CodeArtifactPackageLicense[];
  namespace?: string;
  packageName?: string;
  publishedTime?: string;
  revision?: string;
  sourceCodeRepository?: string;
  status?: string;
  summary?: string;
  version?: string;
}
