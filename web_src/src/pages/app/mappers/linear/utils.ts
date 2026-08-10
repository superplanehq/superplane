import type React from "react";
import type { MetadataItem } from "@/ui/metadataList";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { ExecutionInfo, OutputPayload } from "../types";
import type {
  LinearAttachment,
  LinearAttachmentDeletion,
  LinearComment,
  LinearIssue,
  LinearReaction,
  LinearTeam,
  LinearUser,
  LinearWebhookIssue,
} from "./types";

/** Adds a detail row only when there is a real value, rather than padding with dashes. */
export function addDetail(details: Record<string, string>, label: string, value: string | undefined): void {
  if (value && value.trim() !== "") {
    details[label] = value;
  }
}

/** "ENG-142 · Deploy pipeline fails on retry", falling back to whichever half exists. */
export function getIssueLabel(issue: LinearIssue | LinearWebhookIssue | undefined): string {
  if (!issue) return "";

  if (issue.identifier && issue.title) {
    return `${issue.identifier} · ${issue.title}`;
  }

  return issue.identifier || issue.title || "";
}

export function getUserLabel(user: LinearUser | undefined): string | undefined {
  if (!user) return undefined;
  return user.displayName || user.name || user.email;
}

export function getTeamLabel(team: LinearTeam | undefined, configuredTeam: string | undefined): string | undefined {
  if (team?.name || team?.key) {
    return team.name || team.key;
  }

  return configuredTeam;
}

export function addTeamMetadata(
  metadata: MetadataItem[],
  team: LinearTeam | undefined,
  configuredTeam: string | undefined,
): void {
  const label = getTeamLabel(team, configuredTeam);
  if (label) {
    metadata.push({ icon: "users", label });
  }
}

/**
 * Execution details shared by the issue-returning actions. The timestamp comes
 * first, and at most six rows are shown, prioritising the fields a user cares
 * about and always including the link to the issue when present.
 */
export function buildIssueDetails(execution: ExecutionInfo): Record<string, string> {
  const details: Record<string, string> = {
    "Executed At": execution.createdAt ? new Date(execution.createdAt).toLocaleString() : "-",
  };

  const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
  const issue = outputs?.default?.[0]?.data as LinearIssue | undefined;
  if (!issue) return details;

  addDetail(details, "Issue", issue.identifier);
  addDetail(details, "Issue URL", issue.url);
  addDetail(details, "Title", issue.title);
  addDetail(details, "Status", issue.state?.name);
  addDetail(details, "Assignee", getUserLabel(issue.assignee));

  return details;
}

/** Subtitle shared by the issue-returning actions: the issue label, else a relative time. */
export function buildIssueSubtitle(execution: ExecutionInfo): string | React.ReactNode {
  const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
  const issue = outputs?.default?.[0]?.data as LinearIssue | undefined;

  const label = getIssueLabel(issue);
  if (label) return label;

  if (execution.createdAt) {
    return renderTimeAgo(new Date(execution.createdAt));
  }

  return "";
}

/**
 * Execution details for the addIssueComment action. The timestamp comes first,
 * at most six rows are shown, and the link to the comment is always included
 * when present.
 */
export function buildCommentDetails(execution: ExecutionInfo): Record<string, string> {
  const details: Record<string, string> = {
    "Executed At": execution.createdAt ? new Date(execution.createdAt).toLocaleString() : "-",
  };

  const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
  const comment = outputs?.default?.[0]?.data as LinearComment | undefined;
  if (!comment) return details;

  addDetail(details, "Comment URL", comment.url);
  addDetail(details, "Issue", getIssueLabel(comment.issue));
  addDetail(details, "Author", getUserLabel(comment.user));
  addDetail(details, "Comment", comment.body);

  return details;
}

/**
 * Execution details for the updateIssueComment action: the comment rows plus the
 * edit time, which Linear sets on the first edit and is the proof the update
 * landed. Six rows at most, timestamp first.
 */
export function buildCommentUpdateDetails(execution: ExecutionInfo): Record<string, string> {
  const details = buildCommentDetails(execution);

  const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
  const comment = outputs?.default?.[0]?.data as LinearComment | undefined;
  if (comment?.editedAt) {
    addDetail(details, "Edited At", new Date(comment.editedAt).toLocaleString());
  }

  return details;
}

/** Execution details for the createAttachment action, timestamp first, at most six rows. */
export function buildAttachmentDetails(execution: ExecutionInfo): Record<string, string> {
  const details: Record<string, string> = {
    "Executed At": execution.createdAt ? new Date(execution.createdAt).toLocaleString() : "-",
  };

  const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
  const attachment = outputs?.default?.[0]?.data as LinearAttachment | undefined;
  if (!attachment) return details;

  addDetail(details, "Attachment URL", attachment.url);
  addDetail(details, "Issue", getIssueLabel(attachment.issue));
  addDetail(details, "Title", attachment.title);
  addDetail(details, "Subtitle", attachment.subtitle);

  return details;
}

/** Subtitle for the createAttachment action: the attachment title, else a relative time. */
export function buildAttachmentSubtitle(execution: ExecutionInfo): string | React.ReactNode {
  const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
  const attachment = outputs?.default?.[0]?.data as LinearAttachment | undefined;

  if (attachment?.title) return attachment.title;

  if (execution.createdAt) {
    return renderTimeAgo(new Date(execution.createdAt));
  }

  return "";
}

/** Execution details for the deleteAttachment action, which only returns the deleted ID. */
export function buildAttachmentDeletionDetails(execution: ExecutionInfo): Record<string, string> {
  const details: Record<string, string> = {
    "Executed At": execution.createdAt ? new Date(execution.createdAt).toLocaleString() : "-",
  };

  const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
  const deletion = outputs?.default?.[0]?.data as LinearAttachmentDeletion | undefined;
  if (!deletion) return details;

  addDetail(details, "Attachment", deletion.id);
  addDetail(details, "Deleted", deletion.deleted ? "Yes" : "No");

  return details;
}

/** Subtitle for the addIssueComment action: the issue label, else a relative time. */
export function buildCommentSubtitle(execution: ExecutionInfo): string | React.ReactNode {
  const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
  const comment = outputs?.default?.[0]?.data as LinearComment | undefined;

  const label = getIssueLabel(comment?.issue);
  if (label) return label;

  if (execution.createdAt) {
    return renderTimeAgo(new Date(execution.createdAt));
  }

  return "";
}

/** Execution details for the addReaction action, timestamp first, at most six rows. */
export function buildReactionDetails(execution: ExecutionInfo): Record<string, string> {
  const details: Record<string, string> = {
    "Executed At": execution.createdAt ? new Date(execution.createdAt).toLocaleString() : "-",
  };

  const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
  const reaction = outputs?.default?.[0]?.data as LinearReaction | undefined;
  if (!reaction) return details;

  addDetail(details, "Reaction", reaction.emoji);
  addDetail(details, "Reaction ID", reaction.id);
  addDetail(details, "Author", getUserLabel(reaction.user));

  return details;
}

/** Subtitle for the addReaction action: the emoji, else a relative time. */
export function buildReactionSubtitle(execution: ExecutionInfo): string | React.ReactNode {
  const outputs = execution.outputs as { default?: OutputPayload[] } | undefined;
  const reaction = outputs?.default?.[0]?.data as LinearReaction | undefined;

  if (reaction?.emoji) return reaction.emoji;

  if (execution.createdAt) {
    return renderTimeAgo(new Date(execution.createdAt));
  }

  return "";
}
