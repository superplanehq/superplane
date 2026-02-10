import { Predicate } from "../utils";

export interface DockerHubRepository {
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

export interface DockerHubRepositoryMetadata {
  namespace?: string;
  repository?: DockerHubRepository;
}

export interface DockerHubRepositoryConfiguration {
  namespace?: string;
  repository?: string;
  tag?: string;
  tags?: Predicate[];
}

export type DockerHubTriggerMetadata = DockerHubRepositoryMetadata;
export type DockerHubTriggerConfiguration = DockerHubRepositoryConfiguration;

export interface DockerHubPushData {
  tag?: string;
  pushed_at?: number;
  pusher?: string;
}

export interface DockerHubImagePushEvent {
  callback_url?: string;
  push_data?: DockerHubPushData;
  repository?: DockerHubRepository;
}

export interface DockerHubTagImage {
  architecture?: string;
  os?: string;
  digest?: string;
  size?: number;
  status?: string;
  last_pulled?: string;
  last_pushed?: string;
}

export interface DockerHubTag {
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
  v2?: string;
  images?: DockerHubTagImage[] | DockerHubTagImage;
}
