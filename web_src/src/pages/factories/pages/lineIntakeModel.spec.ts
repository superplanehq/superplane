import { describe, expect, it } from "vitest";

import {
  ADD_INTAKE_TEMPLATES,
  filterAddIntakeTemplates,
  GITHUB_ISSUES_ANALYZING_TICKETS,
  intakeAutomationFixture,
  intakeSourcesFromFactoryIntakes,
  intakeTicketAnalysisFixture,
  intakeTicketConfidenceScore,
  LINE_INTAKE_SOURCES,
  lineIntakeSourceById,
  sortIntakeTicketsByOutcome,
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

  it("maps declared intakes to configured sources, keyed by intake id", () => {
    expect(
      intakeSourcesFromFactoryIntakes([
        { id: "intake-1", canvasId: "canvas-1", name: "Repository issues", source: "SOURCE_GITHUB_ISSUES" },
        { id: "intake-2", canvasId: "canvas-2", name: "Triage issues", source: "SOURCE_GITHUB_ISSUES" },
        { id: "intake-3", canvasId: "canvas-3", name: "Production errors", source: "SOURCE_SENTRY_EXCEPTIONS" },
        { id: "intake-4", canvasId: "canvas-4", name: "Release notes" },
      ]).map(({ intakeId, appId, source }) => ({ intakeId, appId, id: source.id, name: source.name })),
    ).toEqual([
      { intakeId: "intake-1", appId: "canvas-1", id: "github-issues", name: "Repository issues" },
      { intakeId: "intake-2", appId: "canvas-2", id: "github-issues", name: "Triage issues" },
      { intakeId: "intake-3", appId: "canvas-3", id: "sentry-exceptions", name: "Production errors" },
    ]);
  });

  it("falls back to the source name and reads settings and health", () => {
    const [intake] = intakeSourcesFromFactoryIntakes([
      {
        id: "intake-1",
        canvasId: "canvas-1",
        source: "SOURCE_GITHUB_ISSUES",
        healthy: false,
        settings: {
          confidencePct: 80,
          labels: ["bug"],
          labelFilterMode: "LABEL_FILTER_MODE_EXCLUDE",
          assignment: "ASSIGNMENT_UNASSIGNED",
        },
      },
    ]);

    expect(intake?.source.name).toBe("GitHub issues");
    expect(intake?.healthy).toBe(false);
    expect(intake?.settings).toMatchObject({
      name: "GitHub issues",
      confidencePct: 80,
      labels: ["bug"],
      labelFilterMode: "exclude",
      assignment: "unassigned",
    });
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
      "runnerOpenRouter",
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
        confidenceSummary: "This issue is a good fit for an agent on this factory line.",
        confidenceAnalysis: "The GitHub issue names the retryable status codes.",
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
      summary: "This issue is a good fit for an agent on this factory line.",
      analysis: "The GitHub issue names the retryable status codes.",
      sourceName: "Score",
    };
    expect(fixture.checks).toEqual([check]);
    expect(fixture.phases.find((phase) => phase.id === "score")?.checks).toEqual([check]);
  });

  it("reads the ticket score from the intake percentage", () => {
    expect(intakeTicketConfidenceScore({ id: "gh-1", title: "Ticket", confidencePct: 58 })).toBe(3);
    expect(intakeTicketConfidenceScore({ id: "gh-1", title: "Ticket", confidencePct: 12 })).toBe(1);
    expect(intakeTicketConfidenceScore({ id: "gh-1", title: "Ticket", confidenceScore: 5, confidencePct: 12 })).toBe(5);
    expect(intakeTicketConfidenceScore({ id: "gh-1", title: "Ticket" })).toBeUndefined();
  });

  it("keeps analyzing tickets over the tickets below the minimum confidence", () => {
    expect(
      sortIntakeTicketsByOutcome([
        { id: "gh-1", title: "Below", outcome: "below-threshold", confidencePct: 20 },
        { id: "gh-2", title: "Analyzing" },
      ]).map((ticket) => ticket.id),
    ).toEqual(["gh-2", "gh-1"]);

    expect(GITHUB_ISSUES_ANALYZING_TICKETS.filter((ticket) => ticket.outcome === "below-threshold")).toHaveLength(6);
    expect(
      GITHUB_ISSUES_ANALYZING_TICKETS.filter((ticket) => ticket.outcome === "below-threshold").every(
        (ticket) => (ticket.confidencePct ?? 100) < 60,
      ),
    ).toBe(true);
  });

  // The analysis of a ticket under the minimum confidence is over. The popup
  // must show the finished run and the score, not a run that is still going.
  it("marks the analysis of a ticket below the minimum confidence as finished", () => {
    const fixture = intakeTicketAnalysisFixture({
      id: "gh-issue-11",
      title: "Payments break for some customers",
      outcome: "below-threshold",
      confidencePct: 12,
    });

    expect(fixture.phases.map((phase) => phase.status)).toEqual(["passed", "passed", "passed", "passed"]);
    expect(fixture.checks).toEqual([
      expect.objectContaining({
        name: "Confidence score",
        score: 1,
        level: "caution",
        summary: "This issue is a poor fit for an agent on this factory line.",
      }),
    ]);
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
      "runnerOpenRouter",
      "createWorkOrder",
    ]);
  });

  it("builds Sentry and PagerDuty canvases from catalogue triggers", () => {
    const sentry = intakeAutomationFixture(lineIntakeSourceById("sentry-exceptions")!);
    expect(sentry.phases[0]?.canvas?.nodes.map((node) => node.component)).toEqual([
      "sentry.onIssue",
      "runnerOpenRouter",
      "createWorkOrder",
    ]);

    const pagerduty = intakeAutomationFixture(lineIntakeSourceById("pagerduty-incidents")!);
    expect(pagerduty.phases[0]?.canvas?.nodes.map((node) => node.component)).toEqual([
      "pagerduty.onIncident",
      "runnerOpenRouter",
      "createWorkOrder",
    ]);
  });

  it("lists six add-intake templates including CI and page performance", () => {
    expect(ADD_INTAKE_TEMPLATES).toHaveLength(6);
    expect(ADD_INTAKE_TEMPLATES.map((template) => template.id)).toContain("improve-ci-runtime");
    expect(ADD_INTAKE_TEMPLATES.map((template) => template.id)).toContain("improve-page-performance");
  });

  it("filters add-intake templates by name or description", () => {
    expect(filterAddIntakeTemplates("production").map((template) => template.id)).toEqual(["sentry-exceptions"]);
    expect(filterAddIntakeTemplates("incident").map((template) => template.id)).toEqual(["pagerduty-incidents"]);
    expect(filterAddIntakeTemplates("runtime").map((template) => template.id)).toEqual(["improve-ci-runtime"]);
    expect(filterAddIntakeTemplates("")).toHaveLength(6);
  });
});
