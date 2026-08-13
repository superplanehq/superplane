import { describe, expect, it } from "vitest";
import { FACTORY_SIDE_HANDLE_ID, factoryRunLeafEdgeKey, layoutFactoryRunLeafGraph } from "./factoryRunLeafLayout";

function expectNoOverlaps(
  positions: Map<string, { x: number; y: number }>,
  width = 280,
  height = 104,
  gap = 8,
) {
  const ids = [...positions.keys()];
  for (let i = 0; i < ids.length; i++) {
    for (let j = i + 1; j < ids.length; j++) {
      const a = positions.get(ids[i])!;
      const b = positions.get(ids[j])!;
      const overlapX = a.x < b.x + width - gap && b.x < a.x + width - gap;
      const overlapY = a.y < b.y + height - gap && b.y < a.y + height - gap;
      expect(overlapX && overlapY, `${ids[i]} overlaps ${ids[j]}`).toBe(false);
    }
  }
}

describe("layoutFactoryRunLeafGraph", () => {
  it("keeps the longest non-leaf path on the spine and parks leaves to the right", () => {
    const result = layoutFactoryRunLeafGraph(
      [{ id: "a" }, { id: "b" }, { id: "leaf1" }, { id: "leaf2" }, { id: "d" }, { id: "e" }],
      [
        { source: "a", target: "b", sourceHandle: "default" },
        { source: "b", target: "leaf1", sourceHandle: "default" },
        { source: "b", target: "leaf2", sourceHandle: "default" },
        { source: "b", target: "d", sourceHandle: "default" },
        { source: "d", target: "e", sourceHandle: "default" },
      ],
    );

    const a = result.positions.get("a")!;
    const b = result.positions.get("b")!;
    const d = result.positions.get("d")!;
    const e = result.positions.get("e")!;
    const leaf1 = result.positions.get("leaf1")!;
    const leaf2 = result.positions.get("leaf2")!;

    expect(a.x).toBe(b.x);
    expect(b.x).toBe(d.x);
    expect(b.y).toBeGreaterThan(a.y);
    expect(d.y).toBeGreaterThan(b.y);

    expect(leaf1.x).toBeGreaterThan(b.x);
    expect(leaf2.x).toBe(leaf1.x);
    expect(e.x).toBeGreaterThan(d.x);
    expect(result.sideHandleNodeIds.has("b")).toBe(true);
    expect(result.sideHandleNodeIds.has("d")).toBe(true);
    expect(result.leafEdgeKeys.has(factoryRunLeafEdgeKey("b", "leaf1", "default"))).toBe(true);
    expect(result.leafEdgeKeys.has(factoryRunLeafEdgeKey("b", "d", "default"))).toBe(false);
    expect(result.leafEdgeKeys.has(factoryRunLeafEdgeKey("d", "e", "default"))).toBe(true);
    expectNoOverlaps(result.positions);
  });

  it("treats a terminal-only successor as a side leaf", () => {
    const result = layoutFactoryRunLeafGraph(
      [{ id: "a" }, { id: "b" }],
      [{ source: "a", target: "b", sourceHandle: "default" }],
    );

    const a = result.positions.get("a")!;
    const b = result.positions.get("b")!;
    expect(b.x).toBeGreaterThan(a.x);
    expect(result.sideHandleNodeIds.has("a")).toBe(true);
    expect(result.leafEdgeKeys.size).toBe(1);
    expect(FACTORY_SIDE_HANDLE_ID).toBe("__factorySide");
  });

  it("puts the shorter fork to the right as a subtree when a router has two non-leaf branches", () => {
    const result = layoutFactoryRunLeafGraph(
      [
        { id: "router" },
        { id: "success" },
        { id: "s1" },
        { id: "s2" },
        { id: "failed" },
        { id: "f1" },
      ],
      [
        { source: "router", target: "success", sourceHandle: "true" },
        { source: "success", target: "s1", sourceHandle: "default" },
        { source: "s1", target: "s2", sourceHandle: "default" },
        { source: "router", target: "failed", sourceHandle: "false" },
        { source: "failed", target: "f1", sourceHandle: "default" },
      ],
    );

    const router = result.positions.get("router")!;
    const success = result.positions.get("success")!;
    const s1 = result.positions.get("s1")!;
    const failed = result.positions.get("failed")!;
    const f1 = result.positions.get("f1")!;
    const s2 = result.positions.get("s2")!;

    expect(success.x).toBe(router.x);
    expect(s1.x).toBe(router.x);
    expect(success.y).toBeGreaterThan(router.y);
    expect(s1.y).toBeGreaterThan(success.y);

    expect(failed.x).toBeGreaterThan(router.x);
    expect(f1.x).toBeGreaterThan(failed.x);

    expect(s2.x).toBeGreaterThan(s1.x);
    expect(result.leafEdgeKeys.has(factoryRunLeafEdgeKey("router", "failed", "false"))).toBe(true);
    expect(result.leafEdgeKeys.has(factoryRunLeafEdgeKey("router", "success", "true"))).toBe(false);
    expect(result.compactForkNodeIds.has("router")).toBe(true);
    expect(result.spineSourceNodeIds.has("router")).toBe(true);
    expect(result.spineEdgeKeys.has(factoryRunLeafEdgeKey("router", "success", "true"))).toBe(true);
    expectNoOverlaps(result.positions);
  });

  it("nests deeper forks further right without overlapping", () => {
    const result = layoutFactoryRunLeafGraph(
      [
        { id: "a" },
        { id: "b" },
        { id: "c" },
        { id: "side1" },
        { id: "side2" },
        { id: "forkA" },
        { id: "forkB" },
        { id: "cLeaf" },
      ],
      [
        { source: "a", target: "b", sourceHandle: "default" },
        { source: "b", target: "c", sourceHandle: "default" },
        { source: "c", target: "cLeaf", sourceHandle: "default" },
        { source: "b", target: "side1", sourceHandle: "failed" },
        { source: "b", target: "side2", sourceHandle: "error" },
        { source: "side1", target: "forkA", sourceHandle: "true" },
        { source: "side1", target: "forkB", sourceHandle: "false" },
      ],
    );

    const b = result.positions.get("b")!;
    const c = result.positions.get("c")!;
    const side1 = result.positions.get("side1")!;
    const side2 = result.positions.get("side2")!;
    const forkA = result.positions.get("forkA")!;
    const forkB = result.positions.get("forkB")!;

    expect(c.x).toBe(b.x);
    expect(side1.x).toBeGreaterThan(b.x);
    expect(side2.x).toBe(side1.x);
    expect(forkA.x).toBeGreaterThan(side1.x);
    expect(forkB.x).toBe(forkA.x);
    expectNoOverlaps(result.positions);
  });

  it("prefers default/true channels on the spine when branch depths are equal", () => {
    const result = layoutFactoryRunLeafGraph(
      [
        { id: "router" },
        { id: "ok" },
        { id: "okLeaf" },
        { id: "bad" },
        { id: "badLeaf" },
      ],
      [
        { source: "router", target: "bad", sourceHandle: "false" },
        { source: "bad", target: "badLeaf", sourceHandle: "default" },
        { source: "router", target: "ok", sourceHandle: "true" },
        { source: "ok", target: "okLeaf", sourceHandle: "default" },
      ],
    );

    const router = result.positions.get("router")!;
    const ok = result.positions.get("ok")!;
    const bad = result.positions.get("bad")!;

    expect(ok.x).toBe(router.x);
    expect(bad.x).toBeGreaterThan(router.x);
  });

  it("places a merge child once and does not side-remap leftward merge edges", () => {
    // spine: a → b → merge
    // side: a → side → merge
    const result = layoutFactoryRunLeafGraph(
      [{ id: "a" }, { id: "b" }, { id: "side" }, { id: "merge" }],
      [
        { source: "a", target: "b", sourceHandle: "true" },
        { source: "b", target: "merge", sourceHandle: "default" },
        { source: "a", target: "side", sourceHandle: "false" },
        { source: "side", target: "merge", sourceHandle: "default" },
      ],
    );

    const a = result.positions.get("a")!;
    const b = result.positions.get("b")!;
    const side = result.positions.get("side")!;
    const merge = result.positions.get("merge")!;

    expect(b.x).toBe(a.x);
    expect(merge.x).toBe(a.x);
    expect(side.x).toBeGreaterThan(a.x);
    expect(merge.y).toBeGreaterThan(b.y);

    // side → merge goes left/down to spine column — vertical, not side remap.
    expect(result.leafEdgeKeys.has(factoryRunLeafEdgeKey("side", "merge", "default"))).toBe(false);
    expect(result.spineEdgeKeys.has(factoryRunLeafEdgeKey("side", "merge", "default"))).toBe(true);
    expect(result.edgeRouteGutters.has(factoryRunLeafEdgeKey("side", "merge", "default"))).toBe(true);
    expectNoOverlaps(result.positions);
  });

  it("converges two branches without classifying the join as a leftward side edge", () => {
    const result = layoutFactoryRunLeafGraph(
      [
        { id: "root" },
        { id: "left1" },
        { id: "left2" },
        { id: "right1" },
        { id: "right2" },
        { id: "join" },
      ],
      [
        { source: "root", target: "left1", sourceHandle: "true" },
        { source: "left1", target: "left2", sourceHandle: "default" },
        { source: "left2", target: "join", sourceHandle: "default" },
        { source: "root", target: "right1", sourceHandle: "false" },
        { source: "right1", target: "right2", sourceHandle: "default" },
        { source: "right2", target: "join", sourceHandle: "default" },
      ],
    );

    const root = result.positions.get("root")!;
    const left1 = result.positions.get("left1")!;
    const right1 = result.positions.get("right1")!;
    const join = result.positions.get("join")!;

    expect(left1.x).toBe(root.x);
    expect(right1.x).toBeGreaterThan(root.x);
    expect(join.x).toBe(root.x);

    expect(result.leafEdgeKeys.has(factoryRunLeafEdgeKey("right2", "join", "default"))).toBe(false);
    expect(result.spineEdgeKeys.has(factoryRunLeafEdgeKey("right2", "join", "default"))).toBe(true);
    expectNoOverlaps(result.positions);
  });
});
