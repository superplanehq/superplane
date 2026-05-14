export interface JiraProject {
  id?: string;
  key?: string;
  name?: string;
}

export interface JiraIssueFields {
  summary?: string;
  status?: { name?: string };
  priority?: { name?: string };
  issuetype?: { name?: string };
  project?: JiraProject;
  assignee?: {
    accountId?: string;
    displayName?: string;
    emailAddress?: string;
  };
  reporter?: {
    accountId?: string;
    displayName?: string;
  };
  labels?: string[];
  created?: string;
  updated?: string;
}

export interface JiraIssue {
  id?: string;
  key?: string;
  self?: string;
  fields?: JiraIssueFields;
}

export interface JiraDeletedIssue {
  id?: string;
  key?: string;
  deleted?: boolean;
}

export interface JiraNodeMetadata {
  project?: JiraProject;
}
