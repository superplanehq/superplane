import { describe, expect, it } from "vitest";
import { normalizeCanvasNodesWithoutGroups } from "./normalize";

describe("normalizeCanvasNodesWithoutGroups", () => {
  it("removes group widgets and flattens nested child positions", () => {
    const normalized = normalizeCanvasNodesWithoutGroups([
      {
        id: "outer-group",
        type: "TYPE_WIDGET",
        name: "Outer Group",
        widget: { name: "group" },
        position: { x: 100, y: 200 },
        configuration: { childNodeIds: ["inner-group"] },
      },
      {
        id: "inner-group",
        type: "TYPE_WIDGET",
        name: "Inner Group",
        widget: { name: "group" },
        position: { x: 20, y: 30 },
        configuration: { childNodeIds: ["component-1"] },
      },
      {
        id: "component-1",
        type: "TYPE_COMPONENT",
        name: "Component 1",
        position: { x: 5, y: 6 },
        component: { name: "noop" },
      },
    ]);

    expect(normalized).toHaveLength(1);
    expect(normalized[0]?.id).toBe("component-1");
    expect(normalized[0]?.position).toEqual({ x: 125, y: 236 });
  });

  it("removes id-less group widgets", () => {
    const normalized = normalizeCanvasNodesWithoutGroups([
      {
        type: "TYPE_WIDGET",
        name: "Legacy Group",
        widget: { name: "group" },
        position: { x: 50, y: 60 },
      },
      {
        id: "component-1",
        type: "TYPE_COMPONENT",
        name: "Component 1",
        position: { x: 5, y: 6 },
        component: { name: "noop" },
      },
    ]);

    expect(normalized).toHaveLength(1);
    expect(normalized[0]?.id).toBe("component-1");
    expect(normalized[0]?.position).toEqual({ x: 5, y: 6 });
  });
});
