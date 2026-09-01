import { Position, type Edge } from "@xyflow/react";
import { describe, expect, it } from "vitest";

import { factoryNodeCardSize } from "@/lib/factoryCanvasChrome";
import { FACTORY_SIDE_HANDLE_ID, FACTORY_SPINE_HANDLE_ID } from "@/lib/layout/factoryRunLeafLayout";

import { compactLineCanvasGraph } from "./compactLineCanvasGraph";
import type { SplitRunCanvasModel } from "./splitRunCanvases";

type RoutedEdge = Edge & {
  sourcePosition?: Position;
  targetPosition?: Position;
};

function messyIfCanvas(): SplitRunCanvasModel {
  return {
    key: "planning",
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
      { id: "artifact", name: "Add Plan Artifact", component: "addWorkOrderArtifact", position: { x: -40, y: 900 } },
      {
        id: "false-agent",
        name: "Agent - No GH Issue Plan",
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
  it("lays out the factory spine like the main run canvas and badges channel names", () => {
    const { nodes, edges } = compactLineCanvasGraph(messyIfCanvas(), null, undefined, false);
    const byId = new Map(nodes.map((node) => [node.id, node]));

    const onRun = byId.get("on-run")!;
    const ifNode = byId.get("if")!;
    const comment = byId.get("comment")!;
    const label = byId.get("label")!;
    const trueAgent = byId.get("true-agent")!;
    const artifact = byId.get("artifact")!;
    const falseAgent = byId.get("false-agent")!;

    expect(ifNode.position.x).toBe(onRun.position.x);
    expect(comment.position.x).toBe(onRun.position.x);
    expect(label.position.x).toBe(onRun.position.x);
    expect(ifNode.position.y).toBeGreaterThan(onRun.position.y);
    expect(comment.position.y).toBeGreaterThan(ifNode.position.y);
    expect(label.position.y).toBeGreaterThan(comment.position.y);
    expect(trueAgent.position.y).toBeGreaterThan(label.position.y);
    expect(artifact.position.x).not.toBe(-40);
    expect(falseAgent.position.x).toBeGreaterThan(ifNode.position.x);
    expect(falseAgent.position.y).toBe(ifNode.position.y);
    expect(falseAgent.data.isSideTarget).toBe(true);
    expect(falseAgent.data.steps).toEqual(["Clone Repo", "Write Implementation Plan", "Use plan as output"]);
    expect({ width: falseAgent.width, height: falseAgent.height }).toEqual(factoryNodeCardSize(3));
    expect({ width: trueAgent.width, height: trueAgent.height }).toEqual(factoryNodeCardSize());

    const trueEdge = edges.find((edge) => edge.source === "if" && edge.target === "comment");
    const falseEdge = edges.find((edge) => edge.source === "if" && edge.target === "false-agent") as
      | RoutedEdge
      | undefined;
    const mergeEdge = edges.find((edge) => edge.source === "false-agent" && edge.target === "true-agent");
    expect(trueEdge?.sourceHandle).toBe(FACTORY_SPINE_HANDLE_ID);
    expect(falseEdge?.sourceHandle).toBe(FACTORY_SIDE_HANDLE_ID);
    expect(falseEdge?.sourcePosition).toBe(Position.Right);
    expect(falseEdge?.targetPosition).toBe(Position.Left);
    expect(falseEdge?.data).toMatchObject({ channelLabel: "false" });
    expect(mergeEdge?.data).toMatchObject({ channelLabel: "passed" });
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
