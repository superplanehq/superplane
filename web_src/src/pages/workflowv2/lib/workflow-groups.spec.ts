import { describe, expect, it, vi } from "vitest";
import type { CanvasesCanvas, ComponentsNode } from "@/api-client";
import { deleteNodesFromWorkflow, groupWorkflowNodes, ungroupWorkflowNode } from "./workflow-groups";

describe("deleteNodesFromWorkflow", () => {
  it("deletes groups recursively and removes stale child references from surviving groups", () => {
    const nodes: ComponentsNode[] = [
      {
        id: "group-root",
        type: "TYPE_WIDGET",
        widget: { name: "group" },
        configuration: {
          childNodeIds: ["group-nested", "leaf-a"],
        },
      },
      {
        id: "group-nested",
        type: "TYPE_WIDGET",
        widget: { name: "group" },
        configuration: {
          childNodeIds: ["leaf-b"],
        },
      },
      {
        id: "leaf-a",
        type: "TYPE_COMPONENT",
      },
      {
        id: "leaf-b",
        type: "TYPE_COMPONENT",
      },
      {
        id: "group-survivor",
        type: "TYPE_WIDGET",
        widget: { name: "group" },
        configuration: {
          childNodeIds: ["leaf-a", "leaf-c"],
        },
      },
      {
        id: "leaf-c",
        type: "TYPE_COMPONENT",
      },
    ];

    const result = deleteNodesFromWorkflow(nodes, ["group-root"]);

    expect(result.map((node) => node.id)).toEqual(["group-survivor", "leaf-c"]);
    expect(result[0].configuration?.childNodeIds).toEqual(["leaf-c"]);
  });
});

describe("ungroupWorkflowNode", () => {
  it("removes the group node and converts child positions back to canvas coordinates", () => {
    const workflow: CanvasesCanvas = {
      spec: {
        nodes: [
          {
            id: "group-1",
            type: "TYPE_WIDGET",
            widget: { name: "group" },
            configuration: {
              childNodeIds: ["child-1"],
            },
            position: { x: 100, y: 50 },
          },
          {
            id: "child-1",
            type: "TYPE_COMPONENT",
            position: { x: 10, y: 20 },
          },
        ],
      },
    };

    const result = ungroupWorkflowNode(workflow, "group-1");

    expect(result?.spec?.nodes).toEqual([
      {
        id: "child-1",
        type: "TYPE_COMPONENT",
        position: { x: 110, y: 70 },
      },
    ]);
  });

  it("returns null when the group node cannot be found", () => {
    expect(ungroupWorkflowNode({ spec: { nodes: [] } }, "missing-group")).toBeNull();
  });
});

describe("groupWorkflowNodes", () => {
  it("creates a new group node and converts selected node positions to group-relative coordinates", () => {
    vi.spyOn(Math, "random").mockReturnValue(0.123456789);

    const workflow: CanvasesCanvas = {
      spec: {
        nodes: [
          {
            id: "child-1",
            name: "Child 1",
            type: "TYPE_COMPONENT",
            position: { x: 120, y: 220 },
          },
          {
            id: "child-2",
            name: "Child 2",
            type: "TYPE_COMPONENT",
            position: { x: 200, y: 260 },
          },
          {
            id: "other-node",
            name: "Other",
            type: "TYPE_COMPONENT",
            position: { x: 500, y: 600 },
          },
        ],
      },
    };

    const result = groupWorkflowNodes(workflow, { x: 100, y: 200, width: 300, height: 200 }, [
      { id: "child-1", x: 120, y: 220 },
      { id: "child-2", x: 200, y: 260 },
    ]);

    const groupNode = result.spec?.nodes?.[0];
    const child1 = result.spec?.nodes?.find((node) => node.id === "child-1");
    const child2 = result.spec?.nodes?.find((node) => node.id === "child-2");
    const otherNode = result.spec?.nodes?.find((node) => node.id === "other-node");

    expect(groupNode).toMatchObject({
      id: "group-group-4fzzzx",
      name: "group",
      type: "TYPE_WIDGET",
      widget: { name: "group" },
      configuration: {
        childNodeIds: ["child-1", "child-2"],
      },
      position: { x: 60, y: 88 },
    });
    expect(child1?.position).toEqual({ x: 60, y: 132 });
    expect(child2?.position).toEqual({ x: 140, y: 172 });
    expect(otherNode?.position).toEqual({ x: 500, y: 600 });
  });
});
