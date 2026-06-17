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
  slug?: string;
  slug_perm?: string;
  name?: string;
  version?: string;
  format?: string;
  status?: number;
  status_str?: string;
  repository?: string;
  namespace?: string;
  uploaded_at?: string;
  checksum_md5?: string;
  checksum_sha1?: string;
  checksum_sha256?: string;
  checksum_sha512?: string;
  self_url?: string;
  self_html_url?: string;
  cdn_url?: string;
  size?: number;
  size_str?: string;
  description?: string;
  summary?: string;
}

export interface PackageStatusData {
  self_url?: string;
  stage?: number;
  stage_str?: string;
  stage_updated_at?: string;
  status?: number;
  status_reason?: string;
  status_str?: string;
  status_updated_at?: string;
  is_sync_awaiting?: boolean;
  is_sync_completed?: boolean;
  is_sync_failed?: boolean;
  is_sync_in_flight?: boolean;
  is_sync_in_progress?: boolean;
  is_quarantined?: boolean;
  sync_finished_at?: string;
  sync_progress?: number;
}

// Get Package Status

export interface GetPackageStatusConfiguration {
  repository?: string;
  package?: string;
}

// Get Package

export interface GetPackageConfiguration {
  repository?: string;
  package?: string;
}
