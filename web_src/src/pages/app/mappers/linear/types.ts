/** Shapes returned by the Linear GraphQL API and delivered by Linear webhooks. */

import type { Predicate } from "../utils";

export interface LinearTeam {
  id?: string;
  key?: string;
  name?: string;
}

export interface LinearUser {
  id?: string;
  name?: string;
  displayName?: string;
  email?: string;
}

export interface LinearWorkflowState {
  id?: string;
  name?: string;
  type?: string;
}

export interface LinearLabel {
  id?: string;
  name?: string;
}

/** Issue as returned by the `issueCreate` mutation. */
export interface LinearIssue {
  id?: string;
  identifier?: string;
  number?: number;
  title?: string;
  description?: string;
  url?: string;
  priority?: number;
  priorityLabel?: string;
  branchName?: string;
  createdAt?: string;
  updatedAt?: string;
  state?: LinearWorkflowState;
  team?: LinearTeam;
  assignee?: LinearUser;
  creator?: LinearUser;
  project?: { id?: string; name?: string };
  labels?: LinearLabel[];
}

/**
 * Issue as delivered inside a webhook payload. Linear sends flat foreign keys
 * here and inlines only `state`, `team` and `labels` — there is no nested
 * `assignee` object, and the issue URL lives on the envelope rather than here.
 */
export interface LinearWebhookIssue {
  id?: string;
  identifier?: string;
  number?: number;
  title?: string;
  description?: string;
  priority?: number;
  priorityLabel?: string;
  createdAt?: string;
  updatedAt?: string;
  teamId?: string;
  stateId?: string;
  assigneeId?: string;
  creatorId?: string;
  state?: LinearWorkflowState;
  team?: LinearTeam;
  labels?: LinearLabel[];
}

/** Envelope Linear POSTs to the webhook URL. */
export interface LinearWebhookEvent {
  action?: string;
  type?: string;
  url?: string;
  createdAt?: string;
  actor?: {
    id?: string;
    name?: string;
    email?: string;
    type?: string;
  };
  data?: LinearWebhookIssue;
}

/** Metadata SuperPlane stores on Linear nodes during setup. */
export interface LinearNodeMetadata {
  team?: LinearTeam;
}

export interface CreateIssueConfiguration {
  team?: string;
  title?: string;
  project?: string;
  state?: string;
  assignee?: string;
  priority?: string;
  labels?: string[];
}

export interface OnIssueConfiguration {
  team?: string;
  actions?: string[];
  labels?: Predicate[];
}
