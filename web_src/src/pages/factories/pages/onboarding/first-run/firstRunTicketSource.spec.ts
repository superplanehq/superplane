import { describe, expect, it } from "bun:test";

import {
  canAnalyzeTicketSource,
  issuesChoiceForTicketSource,
  ticketSourceFromIssuesChoice,
} from "./firstRunTicketSource";

describe("firstRunTicketSource", () => {
  it("maps ticket rows to onboarding issues choices", () => {
    expect(issuesChoiceForTicketSource("github-issues")).toBe("vcs");
    expect(issuesChoiceForTicketSource("jira")).toBe("jira");
    expect(issuesChoiceForTicketSource("linear")).toBe("linear");
    expect(issuesChoiceForTicketSource(null)).toBeNull();
  });

  it("keeps an unanswered issues choice unselected", () => {
    expect(ticketSourceFromIssuesChoice(null)).toBeNull();
    expect(ticketSourceFromIssuesChoice("skip")).toBeNull();
    expect(ticketSourceFromIssuesChoice("vcs")).toBe("github-issues");
    expect(ticketSourceFromIssuesChoice("jira")).toBe("jira");
    expect(ticketSourceFromIssuesChoice("linear")).toBe("linear");
  });

  it("ignores a saved GitHub backlog choice for Bitbucket", () => {
    expect(ticketSourceFromIssuesChoice("vcs", "bitbucket")).toBeNull();
    expect(ticketSourceFromIssuesChoice("jira", "bitbucket")).toBe("jira");
    expect(ticketSourceFromIssuesChoice("linear", "bitbucket")).toBe("linear");
  });

  it("requires a GitHub connection for analysis", () => {
    expect(canAnalyzeTicketSource({ ticketSource: "github-issues" })).toBe(false);
    expect(canAnalyzeTicketSource({ ticketSource: "github-issues", githubConnected: true })).toBe(true);
    expect(canAnalyzeTicketSource({ ticketSource: null })).toBe(false);
  });

  it("allows analysis for Jira only after a connection and a project", () => {
    expect(canAnalyzeTicketSource({ ticketSource: "jira" })).toBe(false);
    expect(canAnalyzeTicketSource({ ticketSource: "jira", jiraConnected: true })).toBe(false);
    expect(canAnalyzeTicketSource({ ticketSource: "jira", jiraConnected: true, jiraProjectId: "PAY" })).toBe(true);
  });

  it("allows analysis for Linear only after a connection and a project", () => {
    expect(canAnalyzeTicketSource({ ticketSource: "linear" })).toBe(false);
    expect(canAnalyzeTicketSource({ ticketSource: "linear", linearConnected: true })).toBe(false);
    expect(
      canAnalyzeTicketSource({ ticketSource: "linear", linearConnected: true, linearProjectIds: ["project-1"] }),
    ).toBe(true);
  });
});
