import type { IssuesChoiceId, VcsHostId } from "../onboardingFixtures";

import type { FirstRunTicketSource } from "./firstRunTypes";

export function issuesChoiceForTicketSource(source: FirstRunTicketSource | null): IssuesChoiceId | null {
  if (source === "github-issues") return "vcs";
  if (source === "jira") return "jira";
  if (source === "linear") return "linear";
  return null;
}

export function ticketSourceFromIssuesChoice(
  choice: IssuesChoiceId | null,
  vcsHost: VcsHostId | null = "github",
): FirstRunTicketSource | null {
  if (choice === "jira") return "jira";
  if (choice === "linear") return "linear";
  if (choice === "vcs" && vcsHost !== "bitbucket") return "github-issues";
  return null;
}

export function resolveTicketSelection(args: {
  issuesChoice: IssuesChoiceId | null;
  vcsHost: VcsHostId | null;
  githubConnected: boolean;
}): { ticketSource: FirstRunTicketSource | null; issuesChoice: IssuesChoiceId | null } {
  if (args.issuesChoice == null && args.vcsHost === "github" && args.githubConnected) {
    return { ticketSource: "github-issues", issuesChoice: "vcs" };
  }
  return {
    ticketSource: ticketSourceFromIssuesChoice(args.issuesChoice, args.vcsHost),
    issuesChoice: args.issuesChoice,
  };
}

export function canAnalyzeTicketSource(args: {
  ticketSource: FirstRunTicketSource | null;
  githubConnected?: boolean;
  jiraConnected?: boolean;
  jiraProjectId?: string;
  linearConnected?: boolean;
  linearProjectIds?: string[];
}): boolean {
  if (args.ticketSource === "github-issues") return true;
  if (args.ticketSource === "jira") return Boolean(args.jiraConnected && args.jiraProjectId);
  if (args.ticketSource === "linear") return Boolean(args.linearConnected && args.linearProjectIds?.length);
  return false;
}
