import { Position, type Edge as ReactFlowEdge, type Node as ReactFlowNode } from "@xyflow/react";
import { describe, expect, it } from "vitest";
import {
  applyVerticalOverlay,
  buildLayoutCanvasFromFlow,
  computeVerticalLayoutPositions,
  getCanvasStructureSignature,
} from "./verticalLayout";

function node(id: string, x: number, y: number, type = "component"): ReactFlowNode {
  return { id, position: { x, y }, data: { type }, type: "default" };
}

function edge(source: string, target: string, sourceHandle: string | null = null): ReactFlowEdge {
  return { id: `${source}-${target}`, source, target, sourceHandle };
}

describe("buildLayoutCanvasFromFlow", () => {
  it("maps annotation nodes to widgets and others to actions", () => {
    const canvas = buildLayoutCanvasFromFlow([node("a", 0, 0), node("note", 10, 10, "annotation")], []);
    const nodes = canvas.spec?.nodes ?? [];
    expect(nodes.find((n) => n.id === "a")?.type).toBe("TYPE_ACTION");
    expect(nodes.find((n) => n.id === "note")?.type).toBe("TYPE_WIDGET");
  });

  it("maps edges with channel from the source handle", () => {
    const canvas = buildLayoutCanvasFromFlow([node("a", 0, 0), node("b", 0, 0)], [edge("a", "b", "success")]);
    expect(canvas.spec?.edges).toEqual([{ sourceId: "a", targetId: "b", channel: "success" }]);
  });
});

describe("computeVerticalLayoutPositions", () => {
  it("returns an empty map for an empty graph", async () => {
    expect((await computeVerticalLayoutPositions([], [])).size).toBe(0);
  });

  it("lays connected nodes top-to-bottom", async () => {
    const nodes = [node("a", 0, 0), node("b", 0, 0), node("c", 0, 0)];
    const edges = [edge("a", "b"), edge("b", "c")];

    const positions = await computeVerticalLayoutPositions(nodes, edges);
    expect(positions.get("a")!.y).toBeLessThan(positions.get("b")!.y);
    expect(positions.get("b")!.y).toBeLessThan(positions.get("c")!.y);
  });
});

describe("getCanvasStructureSignature", () => {
  it("is stable regardless of ordering", () => {
    const a = getCanvasStructureSignature([node("a", 0, 0), node("b", 0, 0)], [edge("a", "b")]);
    const b = getCanvasStructureSignature([node("b", 0, 0), node("a", 0, 0)], [edge("a", "b")]);
    expect(a).toBe(b);
  });

  it("changes when structure changes", () => {
    const before = getCanvasStructureSignature([node("a", 0, 0), node("b", 0, 0)], []);
    const after = getCanvasStructureSignature([node("a", 0, 0), node("b", 0, 0)], [edge("a", "b")]);
    expect(before).not.toBe(after);
  });

  it("ignores position-only changes", () => {
    const before = getCanvasStructureSignature([node("a", 0, 0)], []);
    const after = getCanvasStructureSignature([node("a", 999, 999)], []);
    expect(before).toBe(after);
  });
});

describe("applyVerticalOverlay", () => {
  it("overrides positions, disables dragging, and re-orients handles", () => {
    const positions = new Map([["a", { x: 100, y: 200 }]]);
    const [overlayed] = applyVerticalOverlay([node("a", 5, 5)], positions);

    expect(overlayed.position).toEqual({ x: 100, y: 200 });
    expect(overlayed.draggable).toBe(false);
    expect(overlayed.sourcePosition).toBe(Position.Bottom);
    expect(overlayed.targetPosition).toBe(Position.Top);
    expect((overlayed.data as { _orientation?: string })._orientation).toBe("vertical");
  });

  it("falls back to the freeform position when the engine has no position yet", () => {
    const [overlayed] = applyVerticalOverlay([node("a", 5, 6)], new Map());
    expect(overlayed.position).toEqual({ x: 5, y: 6 });
  });

  it("keeps annotation nodes in place but re-orients handles", () => {
    const [overlayed] = applyVerticalOverlay([node("note", 5, 6, "annotation")], new Map([["note", { x: 0, y: 0 }]]));
    expect(overlayed.position).toEqual({ x: 5, y: 6 });
    expect(overlayed.draggable).toBeUndefined();
    expect((overlayed.data as { _orientation?: string })._orientation).toBe("vertical");
  });

  it("does not mutate the source nodes", () => {
    const source = node("a", 5, 5);
    applyVerticalOverlay([source], new Map([["a", { x: 100, y: 200 }]]));
    expect(source.position).toEqual({ x: 5, y: 5 });
    expect(source.draggable).toBeUndefined();
  });
});
