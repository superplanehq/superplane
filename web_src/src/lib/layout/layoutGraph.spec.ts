import { describe, expect, it } from "vitest";
import { resolveConnectedComponentNodeIds, resolveDisconnectedComponents } from "./layoutGraph";

type TestNode = { id?: string };

const nodes = (...ids: string[]): TestNode[] => ids.map((id) => ({ id }));

describe("resolveConnectedComponentNodeIds", () => {
  it("returns every node id when there are no seeds", () => {
    const result = resolveConnectedComponentNodeIds(nodes("a", "b", "c"), [], []);
    expect(result).toEqual(["a", "b", "c"]);
  });

  it("returns only the ids reachable from the seeds", () => {
    const edges = [
      { sourceId: "a", targetId: "b" },
      { sourceId: "b", targetId: "c" },
    ];

    const result = resolveConnectedComponentNodeIds(nodes("a", "b", "c", "d"), edges, ["a"]);

    expect(result).toEqual(["a", "b", "c"]);
  });

  it("ignores edges that dangle outside the node set or self-loop", () => {
    const edges = [
      { sourceId: "a", targetId: "a" },
      { sourceId: "a", targetId: "missing" },
    ];

    const result = resolveConnectedComponentNodeIds(nodes("a", "b"), edges, ["a"]);

    expect(result).toEqual(["a"]);
  });
});

describe("resolveDisconnectedComponents", () => {
  it("returns an empty array when there are no nodes", () => {
    expect(resolveDisconnectedComponents([], [])).toEqual([]);
  });

  it("partitions nodes into their disconnected sub-graphs", () => {
    const edges = [
      { sourceId: "a", targetId: "b" },
      { sourceId: "c", targetId: "d" },
    ];

    const components = resolveDisconnectedComponents(nodes("a", "b", "c", "d", "e"), edges);

    expect(components.map((component) => component.map((node) => node.id))).toEqual([["a", "b"], ["c", "d"], ["e"]]);
  });

  it("preserves the original node objects", () => {
    const [nodeA, nodeB] = nodes("a", "b");
    const components = resolveDisconnectedComponents([nodeA, nodeB], [{ sourceId: "a", targetId: "b" }]);

    expect(components).toHaveLength(1);
    expect(components[0]).toContain(nodeA);
    expect(components[0]).toContain(nodeB);
  });
});
