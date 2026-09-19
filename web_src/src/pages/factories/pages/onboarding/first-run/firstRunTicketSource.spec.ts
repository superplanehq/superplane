import { describe, expect, it } from "bun:test";

import {
  canAnalyzeTicketSource,
  DEFAULT_TICKET_SOURCE,
  issuesChoiceForTicketSource,
  ticketSourceFromIssuesChoice,
} from "./firstRunTicketSource";

describe("firstRunTicketSource", () => {
  it("maps ticket rows to onboarding issues choices", () => {
    expect(issuesChoiceForTicketSource("github-issues")).toBe("vcs");
    expect(issuesChoiceForTicketSource("jira")).toBe("jira");
    expect(issuesChoiceForTicketSource("linear")).toBeNull();
    expect(issuesChoiceForTicketSource(null)).toBeNull();
  });

  it("defaults an unanswered issues choice to GitHub Issues", () => {
    expect(ticketSourceFromIssuesChoice(null)).toBe(DEFAULT_TICKET_SOURCE);
    expect(ticketSourceFromIssuesChoice("vcs")).toBe("github-issues");
    expect(ticketSourceFromIssuesChoice("jira")).toBe("jira");
  });

  it("allows analysis for GitHub Issues without extra setup", () => {
    expect(canAnalyzeTicketSource({ ticketSource: "github-issues" })).toBe(true);
    expect(canAnalyzeTicketSource({ ticketSource: null })).toBe(false);
    expect(canAnalyzeTicketSource({ ticketSource: "linear" })).toBe(false);
  });

  it("allows analysis for Jira only after a connection and a project", () => {
    expect(canAnalyzeTicketSource({ ticketSource: "jira" })).toBe(false);
    expect(canAnalyzeTicketSource({ ticketSource: "jira", jiraConnected: true })).toBe(false);
    expect(canAnalyzeTicketSource({ ticketSource: "jira", jiraConnected: true, jiraProjectId: "PAY" })).toBe(true);
  });
});
