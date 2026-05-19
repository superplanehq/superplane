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
  workflowName?: string;
  workflowScheme?: JiraWorkflowScheme;
}

export interface JiraWorkflowVersion {
  id?: string;
  versionNumber?: number;
}

export interface JiraWorkflow {
  id?: string;
  name?: string;
  version?: JiraWorkflowVersion;
}

export interface JiraWorkflowScheme {
  id?: string;
  name?: string;
  description?: string;
  self?: string;
}

export interface JiraWorkflowSchemeAssignment {
  projectId?: string;
  workflowSchemeId?: string;
  draftCreated?: boolean;
  dryRun?: boolean;
  taskId?: string;
  taskStatus?: string;
  taskSelf?: string;
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

export interface CreateWorkflowConfiguration {
  name?: string;
  description?: string;
  scope?: string;
  project?: string;
  statuses?: Array<{ name?: string; category?: string }>;
  transitions?: Array<{ name?: string; from?: string[]; to?: string; type?: string }>;
}

export interface AssignWorkflowToProjectConfiguration {
  project?: string;
  workflowScheme?: string;
  dryRun?: boolean;
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
