export interface PushData {
  images?: string[];
  pushed_at?: number;
  pusher?: string;
  tag?: string;
}

export interface RepositoryInfo {
  comment_count?: number;
  date_created?: number;
  description?: string;
  dockerfile?: string;
  full_description?: string;
  is_official?: boolean;
  is_private?: boolean;
  is_trusted?: boolean;
  name?: string;
  namespace?: string;
  owner?: string;
  repo_name?: string;
  repo_url?: string;
  star_count?: number;
  status?: string;
}

export interface WebhookPayload {
  callback_url?: string;
  push_data?: PushData;
  repository?: RepositoryInfo;
}

export interface Tag {
  creator?: number;
  id?: number;
  last_updated?: string;
  last_updater?: number;
  last_updater_username?: string;
  name?: string;
  repository?: number;
  full_size?: number;
  v2?: boolean;
  tag_status?: string;
  tag_last_pulled?: string;
  tag_last_pushed?: string;
  media_type?: string;
  content_type?: string;
  digest?: string;
  images?: ImageInfo[];
}

export interface ImageInfo {
  architecture?: string;
  features?: string;
  variant?: string;
  digest?: string;
  os?: string;
  os_features?: string;
  os_version?: string;
  size?: number;
  status?: string;
  last_pulled?: string;
  last_pushed?: string;
}

export interface ListTagsResponse {
  count?: number;
  next?: string;
  previous?: string;
  results?: Tag[];
}

export interface RepositoryMetadata {
  namespace?: string;
  name?: string;
  fullName?: string;
  url?: string;
  description?: string;
}

export interface OnImagePushMetadata {
  repository?: RepositoryMetadata;
  webhookUrl?: string;
}

export interface OnImagePushConfiguration {
  namespace?: string;
  repository?: string;
  tags?: { type?: string; value?: string }[];
}

// Legacy types for backwards compatibility
export interface OnImagePushedMetadata {
  repository?: string;
}

export interface OnImagePushedConfiguration {
  repository?: string;
  tagFilter?: string;
}
