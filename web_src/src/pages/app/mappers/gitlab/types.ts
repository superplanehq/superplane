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
