import yaml from "js-yaml";
import { describe, expect, it } from "vitest";

import { buildIntakeAutomationYaml } from "./intakeAutomationTemplate";
import type { LineIntakeSourceId } from "./lineIntakeModel";

interface CanvasYaml {
  metadata: { name: string };
  spec: {
    nodes: Array<{
      id: string;
      component: string;
      configuration?: Record<string, unknown>;
    }>;
    edges: Array<{ channel: string; sourceId: string; targetId: string }>;
  };
}

describe("buildIntakeAutomationYaml", () => {
  it.each([
    ["github-issues", "github.onIssue"],
    ["sentry-exceptions", "sentry.onIssue"],
    ["pagerduty-incidents", "pagerduty.onIncident"],
  ] satisfies Array<[LineIntakeSourceId, string]>)("builds an analysis intake for %s", (sourceId, triggerComponent) => {
    const document = yaml.load(buildIntakeAutomationYaml(sourceId, 65)) as CanvasYaml;

    expect(document.metadata.name).toBeTruthy();
    expect(document.spec.nodes.map((node) => node.component)).toEqual([
      triggerComponent,
      "runnerClaudeCode",
      "if",
      "createWorkOrder",
    ]);
    expect(document.spec.nodes.find((node) => node.id === `${sourceId}-threshold`)?.configuration).toEqual({
      expression: 'int($["Analyze intake"].data[0].result.result) >= 65',
    });
    expect(document.spec.edges).toContainEqual({
      channel: "true",
      sourceId: `${sourceId}-threshold`,
      targetId: `${sourceId}-create`,
    });
  });

  it("clamps the confidence threshold", () => {
    const document = yaml.load(buildIntakeAutomationYaml("github-issues", 120)) as CanvasYaml;

    expect(document.spec.nodes.find((node) => node.id === "github-issues-threshold")?.configuration).toEqual({
      expression: 'int($["Analyze intake"].data[0].result.result) >= 100',
    });
  });
});
