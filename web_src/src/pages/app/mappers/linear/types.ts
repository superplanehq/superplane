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

export interface OnIssueLabelConfiguration {
  team?: string;
  labels?: string[];
}

export interface OnIssueCommentConfiguration {
  team?: string;
  actions?: string[];
  contentFilter?: string;
}

export interface OnIssueAttachmentConfiguration {
  team?: string;
  actions?: string[];
}

export interface CreateAttachmentConfiguration {
  issue?: string;
  title?: string;
  url?: string;
  subtitle?: string;
  iconUrl?: string;
}

export interface DeleteAttachmentConfiguration {
  issue?: string;
  attachment?: string;
}

export interface GetIssueConfiguration {
  issue?: string;
}

export interface UpdateIssueConfiguration {
  team?: string;
  issue?: string;
  title?: string;
  description?: string;
  state?: string;
  assignee?: string;
  priority?: string;
  labels?: string[];
  project?: string;
}

/**
 * Comment as returned by the `commentCreate` and `commentUpdate` mutations.
 * `user` is null for bot/integration authors, and `editedAt` stays null until
 * the body is changed.
 */
export interface LinearComment {
  id?: string;
  body?: string;
  url?: string;
  createdAt?: string;
  updatedAt?: string;
  editedAt?: string;
  user?: LinearUser;
  issue?: { id?: string; identifier?: string; title?: string; url?: string };
}

/**
 * Attachment as returned by the `attachmentCreate` mutation. No `iconUrl`:
 * Linear accepts one on create but never exposes it on the attachment.
 */
export interface LinearAttachment {
  id?: string;
  title?: string;
  subtitle?: string;
  url?: string;
  createdAt?: string;
  updatedAt?: string;
  creator?: LinearUser;
  issue?: { id?: string; identifier?: string; title?: string; url?: string };
}

/** Result of the `deleteAttachment` action, which Linear answers with only a success flag. */
export interface LinearAttachmentDeletion {
  id?: string;
  deleted?: boolean;
}

/**
 * Envelope Linear POSTs for an attachment event. `data` is the attachment; Linear
 * sends only `issueId`, and the trigger adds `issue` by looking it up.
 */
export interface LinearAttachmentWebhookEvent {
  action?: string;
  type?: string;
  createdAt?: string;
  actor?: {
    id?: string;
    name?: string;
    email?: string;
    type?: string;
  };
  data?: LinearAttachment & { issueId?: string };
}

/** Envelope Linear POSTs for a comment event. `data` is the comment. */
export interface LinearCommentWebhookEvent {
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
  data?: LinearComment;
}

export interface AddIssueLabelConfiguration {
  team?: string;
  issue?: string;
  labels?: string[];
  newLabels?: string[];
}

export interface RemoveIssueLabelConfiguration {
  team?: string;
  issue?: string;
  labels?: string[];
  failIfNotFound?: boolean;
}

export interface AddReactionConfiguration {
  target?: string;
  issue?: string;
  comment?: string;
  emoji?: string;
}

/**
 * Reaction as returned by the `reactionCreate` mutation. Linear normalizes emoji
 * aliases on write, so `emoji` can differ from the value that was sent.
 */
export interface LinearReaction {
  id?: string;
  emoji?: string;
  createdAt?: string;
  user?: LinearUser;
}

export interface AddIssueCommentConfiguration {
  issue?: string;
  body?: string;
}

export interface UpdateIssueCommentConfiguration {
  issue?: string;
  comment?: string;
  body?: string;
}
