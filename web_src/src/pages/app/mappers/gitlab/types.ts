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
