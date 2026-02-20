export interface Repository {
  name?: string;
  namespace?: string;
  repo_name?: string;
  repo_url?: string;
  description?: string;
  is_private?: boolean;
  star_count?: number;
  pull_count?: number;
  status?: string;
}

export interface RepositoryMetadata {
  name?: string;
  namespace?: string;
}

export interface TagImage {
  architecture?: string;
  os?: string;
  digest?: string;
  size?: number;
  status?: string;
  last_pulled?: string;
  last_pushed?: string;
}

export interface Tag {
  id?: number;
  name?: string;
  full_size?: number;
  last_updated?: string;
  last_updater?: number;
  last_updater_username?: string;
  status?: string;
  tag_last_pulled?: string;
  tag_last_pushed?: string;
  repository?: number;
  images?: TagImage[] | TagImage;
}
