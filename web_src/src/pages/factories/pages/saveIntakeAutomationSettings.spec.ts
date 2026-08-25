import type { CanvasesCanvas } from "@/api-client";
import { describe, expect, it, vi } from "vitest";

import { saveIntakeAutomationSettings } from "./saveIntakeAutomationSettings";

describe("saveIntakeAutomationSettings", () => {
  it("stages and commits the updated canvas before it renames the app", async () => {
    const calls: string[] = [];
    const stageCanvas = vi.fn(async (_canvasId: string, yaml: string) => {
      calls.push("stage");
      expect(yaml).toContain("name: High-value issues");
      expect(yaml).toContain("confidencePct: 80");
      expect(yaml).toContain('expression: int($["Analyze intake"].data[0].result.result) >= 80');
    });
    const commitCanvas = vi.fn(async () => {
      calls.push("commit");
    });
    const updateCanvas = vi.fn(async () => {
      calls.push("rename");
    });
    const canvas: CanvasesCanvas = {
      metadata: { id: "app-1", name: "GitHub issues" },
      spec: {
        nodes: [
          { id: "trigger", name: "On Issue", component: "github.onIssue" },
          { id: "analysis", name: "Analyze intake", component: "runnerClaudeCode" },
          { id: "threshold", name: "Threshold", component: "if", configuration: { expression: "true" } },
          { id: "create", name: "Create Work Order", component: "createWorkOrder" },
        ],
        edges: [
          { sourceId: "analysis", targetId: "threshold", channel: "passed" },
          { sourceId: "threshold", targetId: "create", channel: "true" },
        ],
      },
    };

    await saveIntakeAutomationSettings({
      canvasId: "app-1",
      context: {
        sourceId: "github-issues",
        triggerNodeId: "trigger",
        analysisNodeId: "analysis",
        createWorkOrderNodeId: "create",
      },
      canvas,
      settings: {
        name: "High-value issues",
        listenMode: "listen",
        confidencePct: 80,
        labelFilterMode: "include",
        labels: [],
        assignment: "any",
      },
      stageCanvas,
      commitCanvas,
      updateCanvas,
    });

    expect(calls).toEqual(["stage", "commit", "rename"]);
    expect(updateCanvas).toHaveBeenCalledWith({ name: "High-value issues" });
  });

  it("does not rename the app when the canvas commit fails", async () => {
    const updateCanvas = vi.fn();

    await expect(
      saveIntakeAutomationSettings({
        canvasId: "app-1",
        context: {
          sourceId: "github-issues",
          triggerNodeId: "trigger",
          analysisNodeId: "analysis",
          createWorkOrderNodeId: "create",
        },
        canvas: { metadata: { id: "app-1", name: "GitHub issues" }, spec: { nodes: [], edges: [] } },
        settings: {
          name: "High-value issues",
          listenMode: "listen",
          confidencePct: 80,
          labelFilterMode: "include",
          labels: [],
          assignment: "any",
        },
        stageCanvas: vi.fn().mockResolvedValue(undefined),
        commitCanvas: vi.fn().mockRejectedValue(new Error("commit failed")),
        updateCanvas,
      }),
    ).rejects.toThrow("commit failed");

    expect(updateCanvas).not.toHaveBeenCalled();
  });
});
