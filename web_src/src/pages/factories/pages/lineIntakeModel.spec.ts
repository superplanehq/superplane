import { describe, expect, it } from "vitest";

import {
  ADD_INTAKE_TEMPLATES,
  filterAddIntakeTemplates,
  intakeAutomationFixture,
  intakeSourcesFromFactoryIntakes,
  intakeTicketAnalysisFixture,
  LINE_INTAKE_SOURCES,
  lineIntakeSourceById,
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
      "runnerClaudeCode",
      "addWorkOrderArtifact",
      "reportWorkOrderCheck",
    ]);
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
    expect(filterAddIntakeTemplates("production").map((template) => template.id)).toEqual(["sentry-exceptions"]);
    expect(filterAddIntakeTemplates("incident").map((template) => template.id)).toEqual(["pagerduty-incidents"]);
    expect(filterAddIntakeTemplates("runtime").map((template) => template.id)).toEqual(["improve-ci-runtime"]);
    expect(filterAddIntakeTemplates("")).toHaveLength(6);
  });
});
