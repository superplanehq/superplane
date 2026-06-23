export interface RepositoryNodeMetadata {
  repositoryId?: string;
  repositoryName?: string;
  repositoryNamespace?: string;
  repositorySlug?: string;
}

export interface PackageNodeMetadata {
  repositoryId?: string;
  repositoryName?: string;
  repositoryNamespace?: string;
  repositorySlug?: string;
  packageId?: string;
  packageName?: string;
}

export interface GetRepositoryConfiguration {
  repository: string;
}

export interface WebhookTriggerNodeMetadata {
  repository?: {
    namespace?: string;
    slug?: string;
  };
  webhookUrl?: string;
  webhookId?: string;
}

export interface GetPackageConfiguration {
  repository?: string;
  package?: string;
}

export interface PackageOperationConfiguration {
  repository?: string;
  package?: string;
  action?: string;
  tags?: string[];
}

export interface RepositoryData {
  name?: string;
  slug?: string;
  slug_perm?: string;
  namespace?: string;
  namespace_url?: string;
  description?: string;
  repository_type_str?: string;
  content_kind?: string;
  storage_region?: string;
  cdn_url?: string;
  self_url?: string;
  self_html_url?: string;
  self_webapp_url?: string;
  is_private?: boolean;
  is_public?: boolean;
  is_open_source?: boolean;
  size?: number;
  size_str?: string;
  package_count?: number;
  package_group_count?: number;
  num_downloads?: number;
  num_quarantined_packages?: number;
  num_policy_violated_packages?: number;
  created_at?: string;
}

// Package types

export interface PackageData {
  // Identity
  slug?: string;
  slug_perm?: string;
  name?: string;
  display_name?: string;
  version?: string;
  filename?: string;
  format?: string;
  repository?: string;
  namespace?: string;
  uploaded_at?: string;
  uploader?: string;

  // Status
  status?: number;
  status_str?: string;
  status_reason?: string | null;
  status_updated_at?: string;

  // Stage / sync
  stage?: number;
  stage_str?: string;
  stage_updated_at?: string;
  is_sync_awaiting?: boolean;
  is_sync_completed?: boolean;
  is_sync_failed?: boolean;
  is_sync_in_flight?: boolean;
  is_sync_in_progress?: boolean;
  sync_finished_at?: string;
  sync_progress?: number;

  // Quarantine / policy
  is_quarantined?: boolean;
  policy_violated?: boolean;

  // Security scanning
  security_scan_status?: string;
  security_scan_started_at?: string;
  security_scan_completed_at?: string;
  vulnerability_scan_results_url?: string;

  // Checksums
  checksum_md5?: string;
  checksum_sha1?: string;
  checksum_sha256?: string;
  checksum_sha512?: string;

  // URLs
  self_url?: string;
  self_html_url?: string;
  self_webapp_url?: string;
  cdn_url?: string;

  // Size / metadata
  size?: number;
  size_str?: string;
  description?: string;
  summary?: string;

  // Tags
  tags?: Record<string, unknown>;
  tags_immutable?: Record<string, unknown>;
}

export interface PackageOperationResult {
  repository?: string;
  package?: string;
  data?: PackageData;
}

// List Packages

export interface ListPackagesConfiguration {
  repository?: string;
  syncStatus?: string;
  quarantineStatus?: string;
  vulnerabilityStatus?: string;
}

export interface TrimmedPackageData {
  description?: string;
  display_name?: string;
  format?: string;
  is_quarantined?: boolean;
  license?: string;
  policy_violated?: boolean;
  repository?: string;
  slug_perm?: string;
  stage_str?: string;
  status_str?: string;
  tags?: Record<string, unknown>;
}

export interface ListPackagesData {
  packages?: TrimmedPackageData[];
}

// Promote Package

export interface PromotePackageConfiguration {
  sourceRepository?: string;
  package?: string;
  destinationRepository?: string;
  mode?: string;
}

export interface PromotePackageResult {
  name?: string;
  version?: string;
  format?: string;
  repository?: string;
  namespace?: string;
  status_str?: string;
  stage_str?: string;
  self_webapp_url?: string;
  slug_perm?: string;
}
