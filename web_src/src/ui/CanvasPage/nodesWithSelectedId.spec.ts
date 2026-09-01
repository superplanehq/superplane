import { describe, expect, it } from "vitest";

import { nodesWithSelectedId, selectOpenSidebarNode } from "./nodesWithSelectedId";

describe("nodesWithSelectedId", () => {
  it("selects only the requested node", () => {
    const nodes = [
      { id: "a", selected: false },
      { id: "b", selected: true },
    ];

    expect(nodesWithSelectedId(nodes, "a")).toEqual([
      { id: "a", selected: true },
      { id: "b", selected: false },
    ]);
  });

  it("returns the same array when selection is already correct", () => {
    const nodes = [
      { id: "a", selected: true },
      { id: "b", selected: false },
    ];

    expect(nodesWithSelectedId(nodes, "a")).toBe(nodes);
  });

  it("leaves nodes unchanged when the requested id is missing", () => {
    const nodes = [{ id: "a", selected: false }];

    expect(nodesWithSelectedId(nodes, "missing")).toBe(nodes);
  });

  it("clears selection when no node id is requested", () => {
    const nodes = [
      { id: "a", selected: true },
      { id: "b", selected: false },
    ];

    expect(nodesWithSelectedId(nodes, null)).toEqual([
      { id: "a", selected: false },
      { id: "b", selected: false },
    ]);
  });
});

describe("selectOpenSidebarNode", () => {
  it("selects the open sidebar node", () => {
    const nodes = [{ id: "a", selected: false }];

    expect(selectOpenSidebarNode(nodes, "a")).toEqual([{ id: "a", selected: true }]);
  });

  it("leaves nodes unchanged when the sidebar has no node", () => {
    const nodes = [{ id: "a", selected: true }];

    expect(selectOpenSidebarNode(nodes, null)).toBe(nodes);
  });
});
