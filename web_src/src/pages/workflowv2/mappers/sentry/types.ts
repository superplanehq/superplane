export interface BaseNodeMetadata {}

export interface Issue {
  id?: string;
  shortId?: string;
  title?: string;
  level?: string;
  status?: string;
  project?: ProjectRef;
  assigned?: {
    id?: string;
    username?: string;
    email?: string;
  };
}

export interface ProjectRef {
  id?: string;
  slug?: string;
  name?: string;
}

export interface ActionUser {
  id?: string;
  username?: string;
  email?: string;
}

export interface OnIssueEventData {
  event?: string;
  issue?: Issue;
  actionUser?: ActionUser;
}

export interface UpdateIssueResponse {
  issue?: Issue;
}
