import { describe, expect, it } from "bun:test";

import { factoryNodeCardSize } from "@/lib/factoryCanvasChrome";

import { compactLineCanvasGraph } from "./compactLineCanvasGraph";
import type { SplitRunCanvasModel } from "./splitRunCanvases";

function messyIfCanvas(): SplitRunCanvasModel {
  return {
    key: "implementation",
    title: "Create Implementation Plan",
    nodes: [
      { id: "on-run", name: "On Run", component: "triggerOnRun", position: { x: 12, y: 0 } },
      { id: "if", name: "From GH issue?", component: "flowIf", position: { x: 40, y: 180 } },
      {
        id: "comment",
        name: "Progress Started Comment",
        component: "githubCreateIssueComment",
        position: { x: 8, y: 360 },
      },
      { id: "label", name: "Add Factory Label", component: "githubAddIssueLabel", position: { x: 20, y: 540 } },
      {
        id: "true-agent",
        name: "Agent - Plan for GH Issue",
        component: "runnerClaudeCode",
        position: { x: 220, y: 720 },
      },
      { id: "artifact", name: "Add Task Artifact", component: "addWorkOrderArtifact", position: { x: -40, y: 900 } },
      {
        id: "false-agent",
        name: "Draft Implementation Plan",
        component: "runnerClaudeCode",
        configuration: {
          steps: [
            { name: "Clone Repo", type: "bash" },
            { name: "Write Implementation Plan", type: "prompt" },
            { name: "Use plan as output", type: "bash" },
          ],
        },
        position: { x: 480, y: 400 },
      },
    ],
    edges: [
      { sourceId: "on-run", targetId: "if", channel: "default" },
      { sourceId: "if", targetId: "comment", channel: "true" },
      { sourceId: "comment", targetId: "label", channel: "default" },
      { sourceId: "label", targetId: "true-agent", channel: "default" },
      { sourceId: "true-agent", targetId: "artifact", channel: "passed" },
      { sourceId: "if", targetId: "false-agent", channel: "false" },
      { sourceId: "false-agent", targetId: "true-agent", channel: "passed" },
    ],
    statuses: {},
    metrics: {},
  };
}

describe("compactLineCanvasGraph", () => {
  it("keeps saved node positions and the original edge channels", () => {
    const { nodes, edges } = compactLineCanvasGraph(messyIfCanvas(), null, undefined, false);
    const byId = new Map(nodes.map((node) => [node.id, node]));

    const onRun = byId.get("on-run")!;
    const ifNode = byId.get("if")!;
    const falseAgent = byId.get("false-agent")!;
    const artifact = byId.get("artifact")!;
    const trueAgent = byId.get("true-agent")!;

    expect(onRun.position).toEqual({ x: 52, y: 0 });
    expect(ifNode.position).toEqual({ x: 80, y: 180 });
    expect(falseAgent.position).toEqual({ x: 520, y: 400 });
    expect(artifact.position).toEqual({ x: 0, y: 900 });
    expect(falseAgent.data.isSideTarget).toBe(false);
    expect(falseAgent.data.steps).toEqual(["Clone Repo", "Write Implementation Plan", "Use plan as output"]);
    expect({ width: falseAgent.width, height: falseAgent.height }).toEqual(factoryNodeCardSize(3));
    expect({ width: trueAgent.width, height: trueAgent.height }).toEqual(factoryNodeCardSize());

    const falseEdge = edges.find((edge) => edge.source === "if" && edge.target === "false-agent");
    const trueEdge = edges.find((edge) => edge.source === "if" && edge.target === "comment");
    expect(falseEdge?.sourceHandle).toBe("false");
    expect(trueEdge?.sourceHandle).toBe("true");
    expect(trueEdge?.type).toBe("custom");
  });

  it("marks the selected node so the compact canvas can highlight it", () => {
    const { nodes } = compactLineCanvasGraph(messyIfCanvas(), "comment", undefined, false);
    expect(nodes.find((node) => node.id === "comment")?.data.isSelected).toBe(true);
    expect(nodes.find((node) => node.id === "on-run")?.data.isSelected).toBe(false);
  });

  it("attaches an edit href only to the selected node", () => {
    const { nodes } = compactLineCanvasGraph(
      messyIfCanvas(),
      "comment",
      undefined,
      false,
      (nodeId) => `/edit?node=${nodeId}`,
    );
    expect(nodes.find((node) => node.id === "comment")?.data.editHref).toBe("/edit?node=comment");
    expect(nodes.find((node) => node.id === "on-run")?.data.editHref).toBeUndefined();
  });
});
