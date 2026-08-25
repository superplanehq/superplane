import { describe, expect, it } from "vitest";

import { intakeAutomationCanvasFromApp } from "./intakeAutomationCanvasModel";

describe("intakeAutomationCanvasFromApp", () => {
  it("keeps the app's own nodes and edges", () => {
    const canvas = intakeAutomationCanvasFromApp("GitHub issues", {
      spec: {
        nodes: [
          { id: "trigger", name: "On Issue", type: "TYPE_TRIGGER", component: "github.onIssue" },
          { id: "analysis", name: "Analyze intake", type: "TYPE_ACTION", component: "runnerClaudeCode" },
          { id: "threshold", name: "Meets confidence threshold?", type: "TYPE_ACTION", component: "if" },
          { id: "create", name: "Create Work Order", type: "TYPE_ACTION", component: "createWorkOrder" },
        ],
        edges: [
          { channel: "default", sourceId: "trigger", targetId: "analysis" },
          { channel: "passed", sourceId: "analysis", targetId: "threshold" },
          { channel: "true", sourceId: "threshold", targetId: "create" },
        ],
      },
    });

    expect(canvas?.title).toBe("GitHub issues");
    expect(canvas?.nodes.map((node) => node.id)).toEqual(["trigger", "analysis", "threshold", "create"]);
    expect(canvas?.edges.map((edge) => edge.channel)).toEqual(["default", "passed", "true"]);
  });

  it("marks only the nodes that report an error", () => {
    const canvas = intakeAutomationCanvasFromApp("GitHub issues", {
      spec: {
        nodes: [
          { id: "trigger", name: "On Issue", component: "github.onIssue" },
          {
            id: "analysis",
            name: "Analyze intake",
            component: "runnerClaudeCode",
            errorMessage: "machine type is required",
          },
          { id: "blank", name: "Create Work Order", component: "createWorkOrder", errorMessage: "  " },
        ],
      },
    });

    expect(canvas?.statuses).toEqual({ analysis: "failed" });
    expect(canvas?.metrics).toEqual({});
  });

  it("returns nothing when the app has no graph", () => {
    expect(intakeAutomationCanvasFromApp("GitHub issues", { spec: { nodes: [] } })).toBeUndefined();
    expect(intakeAutomationCanvasFromApp("GitHub issues", undefined)).toBeUndefined();
  });
});
