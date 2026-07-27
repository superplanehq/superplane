import type React from "react";
import type { MetadataItem } from "@/ui/metadataList";
import { renderTimeAgo } from "@/components/TimeAgo";
import type { ExecutionInfo, OutputPayload } from "../types";
import type { LinearIssue, LinearTeam, LinearUser, LinearWebhookIssue } from "./types";

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
