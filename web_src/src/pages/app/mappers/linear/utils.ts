import type { MetadataItem } from "@/ui/metadataList";
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
