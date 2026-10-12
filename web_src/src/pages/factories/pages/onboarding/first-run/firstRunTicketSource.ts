import type { IssuesChoiceId } from "../onboardingFixtures";

import type { FirstRunTicketSource } from "./firstRunTypes";

export const DEFAULT_TICKET_SOURCE: FirstRunTicketSource = "github-issues";

export function issuesChoiceForTicketSource(source: FirstRunTicketSource | null): IssuesChoiceId | null {
  if (source === "github-issues") return "vcs";
  if (source === "jira") return "jira";
  if (source === "linear") return "linear";
  return null;
}

export function ticketSourceFromIssuesChoice(choice: IssuesChoiceId | null): FirstRunTicketSource {
  if (choice === "jira") return "jira";
  if (choice === "linear") return "linear";
  return DEFAULT_TICKET_SOURCE;
}

export function canAnalyzeTicketSource(args: {
  ticketSource: FirstRunTicketSource | null;
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
