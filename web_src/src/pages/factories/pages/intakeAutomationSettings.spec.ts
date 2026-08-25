import type { CanvasesCanvas } from "@/api-client";
import { describe, expect, it } from "vitest";

import {
  applyIntakeSettingsToCanvas,
  intakeSettingsFromCanvas,
  type IntakeCanvasSettingsContext,
} from "./intakeAutomationSettings";

const context: IntakeCanvasSettingsContext = {
  sourceId: "github-issues",
  triggerNodeId: "trigger",
  analysisNodeId: "analysis",
  createWorkOrderNodeId: "create",
};

const canvas: CanvasesCanvas = {
  metadata: { id: "app-1", name: "GitHub issues", description: "Issue intake" },
  spec: {
    nodes: [
      {
        id: "trigger",
        name: "On Issue",
        component: "github.onIssue",
        configuration: { repository: "acme/api", actions: ["opened"] },
      },
      { id: "analysis", name: "Analyze intake", component: "runnerClaudeCode" },
      {
        id: "threshold",
        name: "Meets confidence threshold?",
        component: "if",
        configuration: { expression: `int($["Analyze intake"].data[0].result.result) >= 65` },
      },
      { id: "create", name: "Create Work Order", component: "createWorkOrder" },
    ],
    edges: [
      { sourceId: "trigger", targetId: "analysis", channel: "default" },
      { sourceId: "analysis", targetId: "threshold", channel: "passed" },
      { sourceId: "threshold", targetId: "create", channel: "true" },
    ],
  },
};

describe("intakeSettingsFromCanvas", () => {
  it("reads the canvas name and threshold from an existing intake", () => {
    expect(intakeSettingsFromCanvas(context, canvas)).toEqual({
      name: "GitHub issues",
      listenMode: "listen",
      confidencePct: 65,
      labelFilterMode: "include",
      labels: [],
      assignment: "any",
    });
  });

  it("reads saved filters from trigger metadata", () => {
    const withSettings: CanvasesCanvas = {
      ...canvas,
      spec: {
        ...canvas.spec,
        nodes: canvas.spec?.nodes?.map((node) =>
          node.id === "trigger"
            ? {
                ...node,
                metadata: {
                  intakeSettings: {
                    listenMode: "listen",
                    confidencePct: 80,
                    labelFilterMode: "exclude",
                    labels: ["documentation"],
                    assignment: "assigned",
                  },
                },
              }
            : node,
        ),
      },
    };

    expect(intakeSettingsFromCanvas(context, withSettings)).toEqual(
      expect.objectContaining({
        confidencePct: 80,
        labelFilterMode: "exclude",
        labels: ["documentation"],
        assignment: "assigned",
      }),
    );
  });
});

describe("applyIntakeSettingsToCanvas", () => {
  it("stores settings and applies threshold, label, and assignment filters", () => {
    const updated = applyIntakeSettingsToCanvas(context, canvas, {
      name: "High-value issues",
      listenMode: "listen",
      confidencePct: 80,
      labelFilterMode: "include",
      labels: ["bug", "enhancement"],
      assignment: "unassigned",
    });
    const trigger = updated.spec?.nodes?.find((node) => node.id === "trigger");
    const threshold = updated.spec?.nodes?.find((node) => node.id === "threshold");

    expect(updated.metadata?.name).toBe("High-value issues");
    expect(trigger?.metadata).toEqual({
      intakeSettings: {
        listenMode: "listen",
        confidencePct: 80,
        labelFilterMode: "include",
        labels: ["bug", "enhancement"],
        assignment: "unassigned",
      },
    });
    expect(threshold?.configuration?.expression).toContain(">= 80");
    expect(threshold?.configuration?.expression).toContain(`["bug","enhancement"]`);
    expect(threshold?.configuration?.expression).toContain("issue.labels.exists");
    expect(threshold?.configuration?.expression).toContain("size(root().data.issue.assignees) == 0");
  });

  it("does not add GitHub filters to other sources", () => {
    const updated = applyIntakeSettingsToCanvas({ ...context, sourceId: "sentry-exceptions" }, canvas, {
      name: "Sentry exceptions",
      listenMode: "listen",
      confidencePct: 70,
      labelFilterMode: "include",
      labels: ["bug"],
      assignment: "assigned",
    });
    const threshold = updated.spec?.nodes?.find((node) => node.id === "threshold");

    expect(threshold?.configuration?.expression).toBe(`int($["Analyze intake"].data[0].result.result) >= 70`);
  });
});
