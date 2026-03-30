import { describe, expect, it, vi } from "vitest";
import type { Node as ReactFlowNode, NodeChange } from "@xyflow/react";
import {
  GROUP_CHILD_EDGE_PADDING,
  GROUP_CHILD_MIN_Y_OFFSET,
  GROUP_MIN_HEIGHT,
  GROUP_MIN_WIDTH,
  GROUP_RESIZE_PADDING,
} from "../groupNode/constants";
import {
  clampGroupChildNodePositionChanges,
  computeGroupSizeFromChildren,
  resizeGroupsAfterChildChanges,
} from "./groupLayout";

type TestNode = ReactFlowNode<{ type?: string }>;

function createNode(overrides: Partial<TestNode> & Pick<TestNode, "id" | "position">): TestNode {
  const { id, position, ...rest } = overrides;
  return {
    id,
    position,
    data: rest.data ?? {},
    ...rest,
  } as TestNode;
}

describe("clampGroupChildNodePositionChanges", () => {
  it("clamps child nodes inside group bounds", () => {
    const nodes: TestNode[] = [
      createNode({
        id: "group-1",
        position: { x: 0, y: 0 },
        data: { type: "group" },
      }),
      createNode({
        id: "child-1",
        parentId: "group-1",
        position: { x: 20, y: 140 },
      }),
    ];

    const changes: NodeChange[] = [
      {
        id: "child-1",
        type: "position",
        position: { x: 2, y: 10 },
      },
    ];

    expect(clampGroupChildNodePositionChanges(changes, nodes)).toEqual([
      {
        id: "child-1",
        type: "position",
        position: { x: GROUP_CHILD_EDGE_PADDING, y: GROUP_CHILD_MIN_Y_OFFSET },
      },
    ]);
  });

  it("leaves non-group or non-child node changes unchanged", () => {
    const nodes: TestNode[] = [
      createNode({
        id: "plain-node",
        position: { x: 0, y: 0 },
      }),
    ];

    const changes: NodeChange[] = [
      {
        id: "plain-node",
        type: "position",
        position: { x: 5, y: 10 },
      },
      {
        id: "plain-node",
        type: "remove",
      },
    ];

    expect(clampGroupChildNodePositionChanges(changes, nodes)).toEqual(changes);
  });
});

describe("computeGroupSizeFromChildren", () => {
  it("returns null when the group has no children", () => {
    const nodes: TestNode[] = [
      createNode({
        id: "group-1",
        position: { x: 0, y: 0 },
        data: { type: "group" },
      }),
    ];

    expect(computeGroupSizeFromChildren("group-1", nodes)).toBeNull();
  });

  it("uses child bounds with padding and minimum dimensions", () => {
    const nodes: TestNode[] = [
      createNode({
        id: "child-1",
        parentId: "group-1",
        position: { x: 50, y: 80 },
        measured: { width: 300, height: 100 },
      }),
      createNode({
        id: "child-2",
        parentId: "group-1",
        position: { x: 400, y: 260 },
        width: 120,
        height: 50,
      }),
    ];

    expect(computeGroupSizeFromChildren("group-1", nodes)).toEqual({
      width: Math.max(GROUP_MIN_WIDTH, Math.round(520 + GROUP_RESIZE_PADDING)),
      height: Math.max(GROUP_MIN_HEIGHT, Math.round(310 + GROUP_RESIZE_PADDING)),
    });
  });
});

describe("resizeGroupsAfterChildChanges", () => {
  it("does nothing when no child position or dimension changes are present", () => {
    const setNodes = vi.fn();
    const nodes: TestNode[] = [
      createNode({
        id: "group-1",
        position: { x: 0, y: 0 },
        data: { type: "group" },
      }),
    ];

    resizeGroupsAfterChildChanges([{ id: "group-1", type: "remove" }], nodes, setNodes);

    expect(setNodes).not.toHaveBeenCalled();
  });

  it("resizes only affected groups to fit their children", () => {
    const setNodes = vi.fn();
    const nodes: TestNode[] = [
      createNode({
        id: "group-1",
        position: { x: 0, y: 0 },
        data: { type: "group" },
        width: GROUP_MIN_WIDTH,
        height: GROUP_MIN_HEIGHT,
        style: { border: "1px solid black" },
      }),
      createNode({
        id: "child-1",
        parentId: "group-1",
        position: { x: 420, y: 300 },
        measured: { width: 120, height: 80 },
      }),
      createNode({
        id: "group-2",
        position: { x: 0, y: 0 },
        data: { type: "group" },
        width: 999,
        height: 999,
      }),
    ];

    resizeGroupsAfterChildChanges(
      [
        {
          id: "child-1",
          type: "position",
          position: { x: 420, y: 300 },
        },
      ],
      nodes,
      setNodes,
    );

    expect(setNodes).toHaveBeenCalledTimes(1);
    const updater = setNodes.mock.calls[0][0] as (currentNodes: TestNode[]) => TestNode[];
    const updatedNodes = updater(nodes);
    const resizedGroup = updatedNodes.find((node) => node.id === "group-1");
    const untouchedGroup = updatedNodes.find((node) => node.id === "group-2");

    expect(resizedGroup).toMatchObject({
      width: 570,
      height: 410,
      style: {
        border: "1px solid black",
        width: 570,
        height: 410,
        zIndex: -1,
      },
    });
    expect(untouchedGroup).toMatchObject({
      width: 999,
      height: 999,
    });
  });
});
