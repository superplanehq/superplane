export interface RepositoryNodeMetadata {
  repositoryId?: string;
  repositoryName?: string;
  repositoryNamespace?: string;
  repositorySlug?: string;
}

export interface GetRepositoryConfiguration {
  repository: string;
}

export interface OnComplianceCheckCompletedMetadata {
  repository?: {
    namespace?: string;
    slug?: string;
  };
  webhookUrl?: string;
  webhookId?: string;
}

export interface PackageComplianceNodeMetadata {
  repository?: string;
  packageId?: string;
  packageName?: string;
  version?: string;
}

export interface GetPackageComplianceConfiguration {
  repository: string;
  package: string;
}

export interface PackageComplianceData {
  name?: string;
  version?: string;
  slug_perm?: string;
  format?: string;
  license?: string;
  spdx_license?: string;
  osi_approved?: boolean;
  policy_violated?: boolean;
  is_quarantined?: boolean;
  status?: string;
  status_reason?: string | null;
  stage?: string;
  tags?: Record<string, string[]>;
  url?: string;
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
