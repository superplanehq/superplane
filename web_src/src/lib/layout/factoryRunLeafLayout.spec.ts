import { describe, expect, it } from "vitest";
import { FACTORY_SIDE_HANDLE_ID, factoryRunLeafEdgeKey, layoutFactoryRunLeafGraph } from "./factoryRunLeafLayout";

function expectNoOverlaps(positions: Map<string, { x: number; y: number }>, width = 280, height = 104, gap = 8) {
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
  it("keeps multiple roots and their branches in separate saved-position lanes", () => {
    const result = layoutFactoryRunLeafGraph(
      [
        { id: "merge", position: { x: 576, y: 978 } },
        { id: "right-root", position: { x: 1044, y: 258 } },
        { id: "left-root", position: { x: 576, y: 258 } },
        { id: "right-filter", position: { x: 1044, y: 618 } },
        { id: "left-filter", position: { x: 576, y: 618 } },
      ],
      [
        { source: "right-root", target: "right-filter", sourceHandle: "default" },
        { source: "left-root", target: "left-filter", sourceHandle: "default" },
        { source: "right-filter", target: "merge", sourceHandle: "default" },
        { source: "left-filter", target: "merge", sourceHandle: "default" },
      ],
    );

    const leftRoot = result.positions.get("left-root")!;
    const rightRoot = result.positions.get("right-root")!;
    const leftFilter = result.positions.get("left-filter")!;
    const rightFilter = result.positions.get("right-filter")!;
    const merge = result.positions.get("merge")!;

    expect(leftRoot.y).toBe(rightRoot.y);
    expect(leftRoot.x).toBeLessThan(rightRoot.x);
    expect(leftFilter.x).toBe(leftRoot.x);
    expect(rightFilter.x).toBe(rightRoot.x);
    expect(leftFilter.y).toBe(rightFilter.y);
    expect(merge.x).toBe(leftFilter.x);
    expect(merge.y).toBeGreaterThan(leftFilter.y);
    expectNoOverlaps(result.positions);
  });

  it("assigns overlapping feedback edges to separate left gutters", () => {
    const result = layoutFactoryRunLeafGraph(
      [
        { id: "trigger", position: { x: 768, y: 1848 } },
        { id: "loop", position: { x: 769, y: 2064 } },
        { id: "workflow", position: { x: 1104, y: 2304 } },
        { id: "runner", position: { x: 1200, y: 2592 } },
        { id: "done", position: { x: 699, y: 2424 } },
      ],
      [
        { id: "trigger-loop", source: "trigger", target: "loop", sourceHandle: "default" },
        { id: "loop-workflow", source: "loop", target: "workflow", sourceHandle: "next" },
        { id: "workflow-loop", source: "workflow", target: "loop", sourceHandle: "passed" },
        { id: "workflow-runner", source: "workflow", target: "runner", sourceHandle: "failed" },
        { id: "runner-loop-passed", source: "runner", target: "loop", sourceHandle: "passed" },
        { id: "runner-loop-failed", source: "runner", target: "loop", sourceHandle: "failed" },
        { id: "loop-done", source: "loop", target: "done", sourceHandle: "done" },
      ],
    );

    const feedbackKeys = [
      factoryRunLeafEdgeKey("workflow", "loop", "passed"),
      factoryRunLeafEdgeKey("runner", "loop", "passed"),
      factoryRunLeafEdgeKey("runner", "loop", "failed"),
    ];
    const gutters = feedbackKeys.map((key) => result.edgeRouteGutters.get(key));
    const yOffsets = feedbackKeys.map((key) => result.edgeRouteOffsetsY.get(key));
    const graphLeft = Math.min(...[...result.positions.values()].map((position) => position.x));

    expect(feedbackKeys.every((key) => result.feedbackEdgeKeys.has(key))).toBe(true);
    expect(new Set(gutters).size).toBe(3);
    expect(yOffsets).toEqual([0, 16, 32]);
    expect(gutters.every((gutter) => gutter != null && gutter < graphLeft)).toBe(true);
    expect(result.displaySourceNodeIds.has("runner")).toBe(true);
    expect(result.spineSourceNodeIds.has("runner")).toBe(true);
    expectNoOverlaps(result.positions);
  });

  it("reuses a feedback gutter when vertical intervals do not overlap", () => {
    const result = layoutFactoryRunLeafGraph(
      [{ id: "root" }, { id: "a" }, { id: "b" }, { id: "c" }, { id: "d" }],
      [
        { source: "root", target: "a", sourceHandle: "default" },
        { source: "a", target: "b", sourceHandle: "default" },
        { source: "b", target: "a", sourceHandle: "repeat" },
        { source: "b", target: "c", sourceHandle: "default" },
        { source: "c", target: "d", sourceHandle: "default" },
        { source: "d", target: "c", sourceHandle: "repeat" },
      ],
    );

    const firstGutter = result.edgeRouteGutters.get(factoryRunLeafEdgeKey("b", "a", "repeat"));
    const secondGutter = result.edgeRouteGutters.get(factoryRunLeafEdgeKey("d", "c", "repeat"));

    expect(firstGutter).toBeDefined();
    expect(secondGutter).toBe(firstGutter);
  });

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

  it("parks a terminal leaf one column right when extra roots merge into the spine", () => {
    const result = layoutFactoryRunLeafGraph(
      [
        { id: "onComment" },
        { id: "onReview" },
        { id: "onReviewComment" },
        { id: "findPr" },
        { id: "addActivity" },
        { id: "claude" },
      ],
      [
        { source: "onComment", target: "findPr", sourceHandle: "default" },
        { source: "onReview", target: "findPr", sourceHandle: "default" },
        { source: "onReviewComment", target: "findPr", sourceHandle: "default" },
        { source: "findPr", target: "addActivity", sourceHandle: "found" },
        { source: "addActivity", target: "claude", sourceHandle: "default" },
      ],
    );

    const findPr = result.positions.get("findPr")!;
    const addActivity = result.positions.get("addActivity")!;
    const claude = result.positions.get("claude")!;
    const onReview = result.positions.get("onReview")!;
    const onReviewComment = result.positions.get("onReviewComment")!;

    expect(addActivity.x).toBe(findPr.x);
    expect(claude.x).toBe(onReview.x);
    expect(claude.x).toBeLessThan(onReviewComment.x);
    expect(claude.y).toBe(addActivity.y);
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
      [{ id: "router" }, { id: "success" }, { id: "s1" }, { id: "s2" }, { id: "failed" }, { id: "f1" }],
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
    expect(result.displaySourceNodeIds.has("router")).toBe(true);
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

  it("places the first side child beside the parent and stacks later siblings below", () => {
    const result = layoutFactoryRunLeafGraph(
      [{ id: "loop" }, { id: "done" }, { id: "next" }],
      [
        { source: "loop", target: "done", sourceHandle: "done" },
        { source: "loop", target: "next", sourceHandle: "next" },
      ],
    );

    const loop = result.positions.get("loop")!;
    const done = result.positions.get("done")!;
    const next = result.positions.get("next")!;

    expect(done.x).toBe(next.x);
    expect(done.x).toBeGreaterThan(loop.x);
    // First side output sits to the right of Loop (same row).
    expect(done.y).toBe(loop.y);
    expect(next.y).toBeGreaterThan(done.y);
    // Side-column gap is larger than spine VERTICAL_GAP alone (104 + 88).
    expect(next.y - done.y).toBeGreaterThanOrEqual(104 + 88);
    expectNoOverlaps(result.positions);
  });

  it("prefers default/true channels on the spine when branch depths are equal", () => {
    const result = layoutFactoryRunLeafGraph(
      [{ id: "router" }, { id: "ok" }, { id: "okLeaf" }, { id: "bad" }, { id: "badLeaf" }],
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
    // Short leftward merges must not take the far-right gutter (giant drift).
    expect(result.edgeRouteGutters.has(factoryRunLeafEdgeKey("side", "merge", "default"))).toBe(false);
    expectNoOverlaps(result.positions);
  });

  it("does not route a side merge through the far-right gutter even when layers skip", () => {
    // true chain longer on spine; false side merges back — longest-path can inflate join layer.
    const result = layoutFactoryRunLeafGraph(
      [{ id: "root" }, { id: "if2" }, { id: "noop3" }, { id: "t1" }, { id: "t2" }, { id: "join" }, { id: "noop8" }],
      [
        { source: "root", target: "if2", sourceHandle: "default" },
        { source: "root", target: "noop8", sourceHandle: "failed" },
        { source: "if2", target: "t1", sourceHandle: "true" },
        { source: "t1", target: "t2", sourceHandle: "default" },
        { source: "t2", target: "join", sourceHandle: "default" },
        { source: "if2", target: "noop3", sourceHandle: "false" },
        { source: "noop3", target: "join", sourceHandle: "default" },
        { source: "noop8", target: "join", sourceHandle: "default" },
      ],
    );

    const if2 = result.positions.get("if2")!;
    const noop3 = result.positions.get("noop3")!;
    const join = result.positions.get("join")!;
    const mergeKey = factoryRunLeafEdgeKey("noop3", "join", "default");

    expect(noop3.x).toBeGreaterThan(if2.x);
    expect(join.x).toBe(if2.x);
    expect(result.edgeRouteGutters.has(mergeKey)).toBe(false);

    const longKey = factoryRunLeafEdgeKey("noop8", "join", "default");
    const longGutter = result.edgeRouteGutters.get(longKey);
    // Long leftward drops may jog locally beside the source — never use a far graph-right trunk.
    if (longGutter != null) {
      expect(longGutter).toBeLessThanOrEqual(result.positions.get("noop8")!.x + 280 + 48);
    }
    expectNoOverlaps(result.positions);
  });

  it("converges two branches without classifying the join as a leftward side edge", () => {
    const result = layoutFactoryRunLeafGraph(
      [{ id: "root" }, { id: "left1" }, { id: "left2" }, { id: "right1" }, { id: "right2" }, { id: "join" }],
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
