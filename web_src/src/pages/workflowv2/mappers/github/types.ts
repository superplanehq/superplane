export interface BaseNodeMetadata {
  repository: {
    id: string;
    name: string;
    url: string;
  };
}

export interface Issue {
  id: number;
  number: number;
  html_url: string;
  state: string;
  created_at: string;
  pull_request?: {
    diff_url: string;
    html_url: string;
    patch_url: string;
    url: string;
  };
  title?: string;
  user?: {
    id: number;
    login: string;
    html_url: string;
  };
  labels?: Label[];
  assignees?: Assignee[];
  closed_at?: string;
  closed_by?: {
    id: number;
    login: string;
    html_url: string;
  };
}

export interface Assignee {
  login: string;
  html_url: string;
}

export interface Label {
  name: string;
}

export interface Release {
  id?: number;
  name?: string;
  tag_name?: string;
  html_url?: string;
  prerelease?: boolean;
  draft?: boolean;
  author?: {
    id: number;
    login: string;
  };
  assets?: Array<{
    id: number;
    name: string;
  }>;
}

export interface PullRequest {
  title?: string;
  id?: string;
  number?: number;
  url?: string;
  html_url?: string;
  head?: {
    sha: string;
    ref: string;
  };
  user?: {
    id: string;
    login: string;
  };
  _links?: {
    html?: {
      href: string;
    };
  };
}

export interface Push {
  head_commit?: {
    message?: string;
    id?: string;
    author?: {
      name?: string;
      email?: string;
      username: string;
    };
  };
}

export interface GitRef {
  ref?: string;
  ref_type?: string;
  repository?: {
    name?: string;
    full_name?: string;
  };
  sender?: {
    login?: string;
  };
}

export interface Comment {
  id?: number;
  body?: string;
  html_url?: string;
  path?: string;
  position?: number;
  line?: number;
  user?: {
    id: number;
    login: string;
    html_url: string;
  };
  created_at?: string;
  updated_at?: string;
}
