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
  repository?: {
    name?: string;
    url?: string;
    id?: number;
  };
}
