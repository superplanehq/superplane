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
    expect(filterAddIntakeTemplates("runtime").map((template) => template.id)).toEqual(["improve-ci-runtime"]);
    expect(filterAddIntakeTemplates("page performance").map((template) => template.id)).toEqual([
      "improve-page-performance",
    ]);
    expect(filterAddIntakeTemplates("")).toHaveLength(6);
  });
});
