import { describe, expect, it } from "vitest";

import { ACME_ONBOARDING_FACTORY_KEY, PRIMARY_FACTORY_KEY } from "../__fixtures__/factoryPageResponses";
import {
  ADD_INTAKE_TEMPLATES,
  filterAddIntakeTemplates,
  intakeAutomationAppId,
  intakeAutomationFixture,
  intakeTicketAnalysisFixture,
  LINE_INTAKE_SOURCES,
  lineIntakeSourceById,
  lineIntakeSourcesForFactory,
} from "./lineIntakeModel";

describe("lineIntakeModel", () => {
  it("defines GitHub, Sentry, and PagerDuty as automations that feed Backlog", () => {
    expect(LINE_INTAKE_SOURCES.map((source) => source.id)).toEqual([
      "github-issues",
      "sentry-exceptions",
      "pagerduty-incidents",
    ]);

    const github = lineIntakeSourceById("github-issues");
    expect(github?.listen.kind).toBe("webhook");
    expect(github?.accept.destination).toBe("backlog");
  });

  it("keeps Sentry and PagerDuty on Semaphore and GitHub issues only on Acme", () => {
    expect(lineIntakeSourcesForFactory(PRIMARY_FACTORY_KEY).map((source) => source.id)).toEqual([
      "github-issues",
      "sentry-exceptions",
      "pagerduty-incidents",
    ]);
    expect(lineIntakeSourcesForFactory(ACME_ONBOARDING_FACTORY_KEY).map((source) => source.id)).toEqual([
      "github-issues",
    ]);
  });

  it("prefers the GitHub issues intake app for the editor", () => {
    expect(intakeAutomationAppId([{ id: "app-acme-planner" }, { id: "app-github-issues-intake" }])).toBe(
      "app-github-issues-intake",
    );
    expect(intakeAutomationAppId([{ id: "app-acme-planner" }])).toBe("app-acme-planner");
    expect(intakeAutomationAppId([])).toBeUndefined();
  });

  it("builds a ticket analysis fixture with ingest, analyze, plan, and score", () => {
    const fixture = intakeTicketAnalysisFixture({
      id: "gh-issue-1",
      title: "Handle duplicate refunds on retry",
    });

    expect(fixture.title).toBe("Handle duplicate refunds on retry");
    expect(fixture.phases.map((phase) => phase.id)).toEqual(["ingest", "analyze", "plan", "score"]);
    expect(fixture.currentPhaseId).toBe("analyze");
    expect(
      intakeTicketAnalysisFixture({ id: "gh-issue-1", title: "Handle duplicate refunds on retry" }, { complete: true })
        .currentPhaseId,
    ).toBe("score");
    expect(fixture.phases[0]?.canvas?.nodes.map((node) => node.name)).toEqual([
      "Ingest",
      "Analyze ticket",
      "Create plan",
      "Score",
    ]);
    expect(fixture.phases[0]?.canvas?.nodes.map((node) => node.component)).toEqual([
      "github.onIssue",
      "runnerClaudeCode",
      "addWorkOrderArtifact",
      "reportWorkOrderCheck",
    ]);
  });

  it("gives each finished analysis automation a duration and passed step state", () => {
    const fixture = intakeTicketAnalysisFixture(
      { id: "gh-issue-1", title: "Handle duplicate refunds on retry" },
      { complete: true },
    );

    expect(fixture.phases.map((phase) => [phase.id, phase.status, phase.duration])).toEqual([
      ["ingest", "passed", "2s"],
      ["analyze", "passed", "3m 45s"],
      ["plan", "passed", "18s"],
      ["score", "passed", "7s"],
    ]);
    expect(fixture.phases[0]?.canvas?.statuses).toEqual({
      "ticket-ingest": "triggered",
      "ticket-analyze": "passed",
      "ticket-plan": "passed",
      "ticket-score": "passed",
    });
    expect(fixture.phases[0]?.canvas?.metrics).toEqual({
      "ticket-ingest": "2s",
      "ticket-analyze": "3m 45s",
      "ticket-plan": "18s",
      "ticket-score": "7s",
    });
  });

  it("attaches the Plan tab markdown as plan.md on Create plan", () => {
    const planMarkdown = "## Goal\n\nRetry webhook delivery with bounded backoff.";
    const fixture = intakeTicketAnalysisFixture(
      {
        id: "gh-issue-1",
        title: "Handle duplicate refunds on retry",
        planMarkdown,
      },
      { complete: true },
    );

    expect(fixture.phases.find((phase) => phase.id === "plan")?.artifacts).toEqual([
      expect.objectContaining({
        type: "TYPE_MARKDOWN",
        data: expect.objectContaining({
          name: "plan.md",
          title: "plan.md",
          body: planMarkdown,
        }),
      }),
    ]);
    expect(fixture.phases.find((phase) => phase.id === "analyze")?.artifacts).toEqual([]);
  });

  it("attaches a confidence check on Score from the ticket score", () => {
    const fixture = intakeTicketAnalysisFixture(
      {
        id: "gh-issue-1",
        title: "Handle duplicate refunds on retry",
        confidenceScore: 5,
        confidenceSummary: "Analysis complete. The ticket is ready to implement.",
        confidenceAnalysis: "Acceptance criteria name the retryable status codes.",
      },
      { complete: true },
    );

    const check = {
      id: "gh-issue-1-confidence",
      name: "Confidence score",
      score: 5,
      maxScore: 5,
      format: "fraction",
      level: "positive",
      summary: "Analysis complete. The ticket is ready to implement.",
      analysis: "Acceptance criteria name the retryable status codes.",
      sourceName: "Score",
    };
    expect(fixture.checks).toEqual([check]);
    expect(fixture.phases.find((phase) => phase.id === "score")?.checks).toEqual([check]);
  });

  it("keeps later analysis steps pending while analyze runs", () => {
    const fixture = intakeTicketAnalysisFixture({
      id: "gh-issue-1",
      title: "Handle duplicate refunds on retry",
    });

    expect(fixture.phases.map((phase) => [phase.id, phase.status, phase.duration])).toEqual([
      ["ingest", "passed", "2s"],
      ["analyze", "running", "3m 12s so far"],
      ["plan", "pending", "—"],
      ["score", "pending", "—"],
    ]);
    expect(fixture.checks).toEqual([]);
    expect(fixture.phases.find((phase) => phase.id === "score")?.checks).toEqual([]);
    expect(fixture.phases[0]?.canvas?.statuses).toEqual({
      "ticket-ingest": "triggered",
      "ticket-analyze": "running",
      "ticket-plan": "pending",
      "ticket-score": "pending",
    });
  });

  it("imports the GitHub issue as details markdown and a link artifact", () => {
    const fixture = intakeTicketAnalysisFixture({
      id: "gh-issue-1",
      title: "Handle duplicate refunds on retry",
      detailsMarkdown: "Refund retries must stay idempotent.",
      issueKey: "PAY-843",
      issueUrl: "https://github.com/acme/payments-service/issues/843",
    });

    expect(fixture.phases[0]?.artifacts).toEqual([
      expect.objectContaining({
        type: "TYPE_MARKDOWN",
        data: expect.objectContaining({
          name: "details.md",
          title: "details.md",
          body: "Refund retries must stay idempotent.",
        }),
      }),
      expect.objectContaining({
        type: "TYPE_LINK",
        data: expect.objectContaining({
          title: "PAY-843",
          url: "https://github.com/acme/payments-service/issues/843",
        }),
      }),
    ]);
    expect(fixture.phases.slice(1).every((phase) => phase.artifacts.length === 0)).toBe(true);
  });

  it("builds a split-run fixture with listen, evaluate, and backlog steps", () => {
    const github = lineIntakeSourceById("github-issues");
    expect(github).toBeDefined();
    const fixture = intakeAutomationFixture(github!);

    expect(fixture.title).toBe("GitHub issues");
    expect(fixture.phases.map((phase) => phase.id)).toEqual(["listen", "evaluate", "backlog"]);
    expect(fixture.currentPhaseId).toBe("evaluate");
    expect(fixture.waitingNotes[0]?.text).toContain("Backlog");

    const canvas = fixture.phases[0]?.canvas;
    expect(canvas?.nodes.map((node) => node.component)).toEqual([
      "github.onIssue",
      "runnerClaudeCode",
      "createWorkOrder",
    ]);
  });

  it("builds Sentry and PagerDuty canvases from catalogue triggers", () => {
    const sentry = intakeAutomationFixture(lineIntakeSourceById("sentry-exceptions")!);
    expect(sentry.phases[0]?.canvas?.nodes.map((node) => node.component)).toEqual([
      "sentry.onIssue",
      "runnerClaudeCode",
      "createWorkOrder",
    ]);

    const pagerduty = intakeAutomationFixture(lineIntakeSourceById("pagerduty-incidents")!);
    expect(pagerduty.phases[0]?.canvas?.nodes.map((node) => node.component)).toEqual([
      "pagerduty.onIncident",
      "runnerClaudeCode",
      "createWorkOrder",
    ]);
  });

  it("lists six add-intake templates including CI and page performance", () => {
    expect(ADD_INTAKE_TEMPLATES).toHaveLength(6);
    expect(ADD_INTAKE_TEMPLATES.map((template) => template.id)).toContain("improve-ci-runtime");
    expect(ADD_INTAKE_TEMPLATES.map((template) => template.id)).toContain("improve-page-performance");
  });

  it("filters add-intake templates by name or description", () => {
    expect(filterAddIntakeTemplates("runtime").map((template) => template.id)).toEqual(["improve-ci-runtime"]);
    expect(filterAddIntakeTemplates("page performance").map((template) => template.id)).toEqual([
      "improve-page-performance",
    ]);
    expect(filterAddIntakeTemplates("")).toHaveLength(6);
  });
});
