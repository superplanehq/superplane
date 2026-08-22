import { describe, expect, it } from "vitest";

import {
  ADD_INTAKE_TEMPLATES,
  filterAddIntakeTemplates,
  intakeAutomationFixture,
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
