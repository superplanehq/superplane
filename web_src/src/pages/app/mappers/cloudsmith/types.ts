export interface RepositoryNodeMetadata {
  repositoryId?: string;
  repositoryName?: string;
  repositoryNamespace?: string;
  repositorySlug?: string;
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
