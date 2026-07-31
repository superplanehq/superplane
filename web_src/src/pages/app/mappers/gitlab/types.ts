export interface Issue {
  id: number;
  iid: number;
  project_id: number;
  title: string;
  description: string;
  state: string;
  created_at: string;
  updated_at: string;
  closed_at?: string;
  closed_by?: User;
  labels: string[];
  assignees?: User[];
  author: User;
  type: string;
  web_url: string;
  milestone?: Milestone;
  due_date?: string;
}

export interface Milestone {
  id: number;
  iid: number;
  title: string;
  state: string;
}

export interface User {
  id: number;
  name: string;
  username: string;
  state: string;
  avatar_url: string;
  web_url: string;
}

export interface GitLabNodeMetadata {
  project?: {
    name?: string;
    url?: string;
    id?: number;
  };
}

export interface CommitStatus {
  id: number;
  sha: string;
  ref?: string;
  status: string;
  name?: string;
  target_url?: string;
  description?: string;
  created_at?: string;
  coverage?: number | null;
  pipeline_id?: number;
  author?: User;
}

export interface CommitPipeline {
  id?: number;
  ref?: string;
  sha?: string;
  status?: string;
  web_url?: string;
}

export interface Commit {
  id?: string;
  short_id?: string;
  title?: string;
  status?: string;
  web_url?: string;
  last_pipeline?: CommitPipeline;
}

export interface Note {
  id: number;
  body: string;
  author: User;
  created_at: string;
  updated_at: string;
  system: boolean;
  noteable_id?: number;
  noteable_iid?: number;
  noteable_type?: string;
}

export interface MergeRequest {
  id: number;
  iid: number;
  project_id: number;
  title: string;
  description?: string;
  state: string;
  created_at?: string;
  updated_at?: string;
  merged_at?: string;
  merge_user?: User;
  source_branch?: string;
  target_branch?: string;
  sha?: string;
  merge_commit_sha?: string;
  squash_commit_sha?: string;
  detailed_merge_status?: string;
  draft?: boolean;
  labels?: string[];
  author?: User;
  assignees?: User[];
  reviewers?: User[];
  web_url?: string;
}

export interface MergeRequestApprover {
  user?: User;
  approved_at?: string;
}

export interface MergeRequestApproval {
  id: number;
  iid: number;
  project_id: number;
  title?: string;
  state?: string;
  approvals_required?: number;
  approvals_left?: number;
  approved_by?: MergeRequestApprover[];
}

export interface AwardEmoji {
  id: number;
  name: string;
  user: User;
  created_at: string;
  updated_at: string;
  awardable_id?: number;
}

export interface DeploymentEnvironment {
  id: number;
  name: string;
  external_url?: string;
}

export interface Deployment {
  id: number;
  iid: number;
  ref: string;
  sha: string;
  status: string;
  created_at: string;
  updated_at?: string;
  user?: User;
  environment?: DeploymentEnvironment;
}

export interface ReleaseCommit {
  id: string;
  short_id: string;
  title: string;
  created_at: string;
  parent_ids?: string[];
  message: string;
  author_name: string;
  author_email: string;
  authored_date: string;
  committer_name: string;
  committer_email: string;
  committed_date: string;
}

export interface ReleaseMilestoneIssueStats {
  total: number;
  closed: number;
}

export interface ReleaseMilestone {
  id: number;
  iid: number;
  project_id: number;
  title: string;
  description: string;
  state: string;
  created_at: string;
  updated_at: string;
  due_date?: string;
  start_date?: string;
  web_url: string;
  issue_stats?: ReleaseMilestoneIssueStats;
}

export interface ReleaseAssetSource {
  format: string;
  url: string;
}

export interface ReleaseAssetLink {
  id: number;
  name: string;
  url: string;
  link_type: string;
}

export interface ReleaseAssets {
  count: number;
  sources?: ReleaseAssetSource[];
  links?: ReleaseAssetLink[];
}

export interface ReleaseEvidence {
  sha: string;
  filepath: string;
  collected_at: string;
}

export interface ReleaseLinks {
  closed_issues_url?: string;
  closed_merge_requests_url?: string;
  edit_url?: string;
  merged_merge_requests_url?: string;
  opened_issues_url?: string;
  opened_merge_requests_url?: string;
  self?: string;
}

export interface Release {
  tag_name: string;
  name: string;
  description: string;
  created_at: string;
  released_at: string;
  author?: User;
  commit?: ReleaseCommit;
  milestones?: ReleaseMilestone[];
  commit_path?: string;
  tag_path?: string;
  assets?: ReleaseAssets;
  evidences?: ReleaseEvidence[];
  evidence_sha?: string;
  _links?: ReleaseLinks;
}

export interface PipelineMinutesUsageProjectRef {
  id: string;
  name: string;
  fullPath: string;
}

export interface PipelineMinutesUsageProject {
  minutes: number;
  sharedRunnersDuration: number;
  project?: PipelineMinutesUsageProjectRef;
}

export interface PipelineMinutesUsage {
  month: string;
  monthIso8601: string;
  minutes: number;
  sharedRunnersDuration: number;
  projects: PipelineMinutesUsageProject[];
}
