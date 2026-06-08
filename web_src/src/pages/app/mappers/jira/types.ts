export interface JiraProject {
  id?: string;
  key?: string;
  name?: string;
  style?: string;
  simplified?: boolean;
}

export interface JiraStatus {
  name?: string;
  statusCategory?: { name?: string; key?: string };
}

export interface JiraUser {
  accountId?: string;
  displayName?: string;
  emailAddress?: string;
}

export interface JiraIssueFields {
  summary?: string;
  status?: JiraStatus;
  priority?: { name?: string };
  issuetype?: { name?: string };
  project?: JiraProject;
  assignee?: JiraUser;
  reporter?: JiraUser;
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
  issueType?: string;
  status?: string;
}

export interface JiraWorkflowStatus {
  id?: string;
  name?: string;
  category?: string;
  isCurrent?: boolean;
}

export interface JiraWorkflowAvailableTransition {
  id?: string;
  name?: string;
  toStatusId?: string;
  toStatus?: string;
}

export interface JiraWorkflow {
  issueKey?: string;
  issueType?: string;
  projectKey?: string;
  workflowName?: string;
  workflowSchemeId?: string;
  workflowSchemeName?: string;
  currentStatus?: string;
  currentStatusId?: string;
  statuses?: JiraWorkflowStatus[];
  availableTransitions?: JiraWorkflowAvailableTransition[];
}

export interface JiraApproval {
  id?: string;
  name?: string;
  finalDecision?: string;
  approvers?: Array<{
    approver?: JiraUser;
    approverDecision?: string;
  }>;
}

export interface CreateIssueConfiguration {
  project?: string;
  issueType?: string;
  summary?: string;
  description?: string;
  assignee?: string;
  status?: string;
}

export interface GetIssueConfiguration {
  project?: string;
  issueKey?: string;
  expand?: string;
}

export interface UpdateIssueConfiguration {
  project?: string;
  issueKey?: string;
  summary?: string;
  description?: string;
  issueType?: string;
  assignee?: string;
  priority?: string;
  labels?: string[];
  notifyUsers?: boolean;
}

export interface DeleteIssueConfiguration {
  project?: string;
  issueKey?: string;
  deleteSubtasks?: boolean;
}

export interface GetWorkflowConfiguration {
  project?: string;
  issueKey?: string;
}

export interface TransitionIssueConfiguration {
  project?: string;
  issueKey?: string;
  targetStatus?: string;
  comment?: string;
  resolution?: string;
}

export interface ApproveWorkflowConfiguration {
  issueKey?: string;
  decision?: string;
  approvalSelector?: string;
  approvalId?: string;
  comment?: string;
}
