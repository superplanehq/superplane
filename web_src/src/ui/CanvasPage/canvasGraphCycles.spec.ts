import { describe, expect, it } from "vitest";

import type { SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { isLoopNode, wouldCreateCycle } from "./canvasGraphCycles";

function component(id: string, name = "noop"): ComponentsNode {
  return { id, name: id, type: "TYPE_ACTION", component: name };
}

function loop(id: string): ComponentsNode {
  return component(id, "loop");
}

function trigger(id: string): ComponentsNode {
  return { id, name: id, type: "TYPE_TRIGGER", component: "start" };
}

function edge(source: string, target: string) {
  return { source, target };
}

describe("isLoopNode", () => {
  it("identifies loop components", () => {
    expect(isLoopNode(loop("l"))).toBe(true);
    expect(isLoopNode(component("a"))).toBe(false);
    expect(isLoopNode(trigger("t"))).toBe(false);
  });
});

describe("wouldCreateCycle", () => {
  /*
   * The repro from issue #5773: two regular components, then connecting the
   * downstream one back to the upstream one.
   */
  it("rejects connecting a downstream component back to an upstream one", () => {
    const nodes = [component("deploy"), component("deploy-failed")];
    const edges = [edge("deploy", "deploy-failed")];

    expect(wouldCreateCycle(nodes, edges, "deploy-failed", "deploy")).toBe(true);
  });

  it("allows a connection that keeps the graph acyclic", () => {
    const nodes = [component("a"), component("b"), component("c")];
    const edges = [edge("a", "b")];

    expect(wouldCreateCycle(nodes, edges, "b", "c")).toBe(false);
    expect(wouldCreateCycle(nodes, edges, "a", "c")).toBe(false);
  });

  it("rejects a component connecting to itself", () => {
    const nodes = [component("a")];

    expect(wouldCreateCycle(nodes, [], "a", "a")).toBe(true);
  });

  it("rejects a cycle that closes through intermediate nodes", () => {
    const nodes = [component("a"), component("b"), component("c"), component("d")];
    const edges = [edge("a", "b"), edge("b", "c"), edge("c", "d")];

    expect(wouldCreateCycle(nodes, edges, "d", "a")).toBe(true);
    expect(wouldCreateCycle(nodes, edges, "d", "b")).toBe(true);
  });

  /*
   * Mirrors TestCheckForCycles_AllowsFeedbackIntoLoop: feeding a worker's
   * output back into the loop node it came from is the point of the component.
   */
  it("allows feedback into a loop node", () => {
    const nodes = [trigger("trigger"), loop("loop"), component("worker")];
    const edges = [edge("trigger", "loop"), edge("loop", "worker")];

    expect(wouldCreateCycle(nodes, edges, "worker", "loop")).toBe(false);
  });

  it("allows a loop node to connect to itself", () => {
    const nodes = [loop("loop")];

    expect(wouldCreateCycle(nodes, [], "loop", "loop")).toBe(false);
  });

  /*
   * Edges into a loop node are excluded from the graph, exactly as the server
   * excludes them, so a path that only closes through a loop node's input is
   * not a cycle for a regular component either.
   */
  it("does not treat a path through a loop node's input as reachable", () => {
    const nodes = [component("a"), loop("loop"), component("worker")];
    const edges = [edge("a", "loop"), edge("loop", "worker")];

    expect(wouldCreateCycle(nodes, edges, "worker", "a")).toBe(false);
  });

  it("still rejects cycles among regular components in a canvas that has a loop", () => {
    const nodes = [loop("loop"), component("a"), component("b")];
    const edges = [edge("loop", "a"), edge("a", "b")];

    expect(wouldCreateCycle(nodes, edges, "b", "a")).toBe(true);
  });

  it("ignores edges with a missing endpoint", () => {
    const nodes = [component("a"), component("b")];
    const edges = [{ source: "a", target: null }, edge("a", "b")];

    expect(wouldCreateCycle(nodes, edges, "b", "a")).toBe(true);
  });

  it("returns false when either endpoint is missing", () => {
    const nodes = [component("a")];

    expect(wouldCreateCycle(nodes, [], "", "a")).toBe(false);
    expect(wouldCreateCycle(nodes, [], "a", "")).toBe(false);
  });

  it("terminates on a graph that already contains a cycle", () => {
    const nodes = [component("a"), component("b"), component("c")];
    const edges = [edge("a", "b"), edge("b", "a")];

    expect(wouldCreateCycle(nodes, edges, "c", "a")).toBe(false);
    expect(wouldCreateCycle(nodes, edges, "a", "c")).toBe(false);
  });
});
