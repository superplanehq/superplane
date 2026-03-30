import type { Node as ReactFlowNode, NodeChange, NodePositionChange } from "@xyflow/react";
import {
  GROUP_CHILD_EDGE_PADDING,
  GROUP_CHILD_MIN_Y_OFFSET,
  GROUP_MIN_HEIGHT,
  GROUP_MIN_WIDTH,
  GROUP_RESIZE_PADDING,
} from "../groupNode/constants";

function getNodeType(node?: ReactFlowNode): string | undefined {
  return (node?.data as { type?: string } | undefined)?.type;
}

function isGroupNode(node?: ReactFlowNode): boolean {
  return getNodeType(node) === "group";
}

/**
 * React Flow reports child-node drag updates in the group's local coordinate
 * space. This clamps those updates so children cannot be dragged into the
 * group's header area or flush against the left border.
 */
export function clampGroupChildNodePositionChanges<TNode extends ReactFlowNode>(
  changes: NodeChange[],
  nodes: TNode[],
): NodeChange[] {
  const nodesById = new Map(nodes.map((node) => [node.id, node]));

  return changes.map((change) => {
    if (change.type !== "position") {
      return change;
    }

    const posChange = change as NodePositionChange;
    if (!posChange.position) {
      return change;
    }

    const node = nodesById.get(posChange.id);
    if (!node?.parentId) {
      return change;
    }

    const parent = nodesById.get(node.parentId);
    if (!isGroupNode(parent)) {
      return change;
    }

    const x = Math.max(posChange.position.x, GROUP_CHILD_EDGE_PADDING);
    const y = Math.max(posChange.position.y, GROUP_CHILD_MIN_Y_OFFSET);

    if (x === posChange.position.x && y === posChange.position.y) {
      return change;
    }

    return {
      ...posChange,
      position: { ...posChange.position, x, y },
    };
  });
}

/**
 * Computes the minimum group box needed to contain all of its current child
 * nodes, including the extra padding we want around the visible content.
 */
export function computeGroupSizeFromChildren<TNode extends ReactFlowNode>(
  groupId: string,
  nodes: TNode[],
): { width: number; height: number } | null {
  const children = nodes.filter((node) => node.parentId === groupId);
  if (children.length === 0) {
    return null;
  }

  let maxRight = 0;
  let maxBottom = 0;

  for (const child of children) {
    const cx = child.position?.x ?? 0;
    const cy = child.position?.y ?? 0;
    const cw = child.measured?.width ?? child.width ?? 240;
    const ch = child.measured?.height ?? child.height ?? 80;
    maxRight = Math.max(maxRight, cx + cw);
    maxBottom = Math.max(maxBottom, cy + ch);
  }

  return {
    width: Math.max(GROUP_MIN_WIDTH, Math.round(maxRight + GROUP_RESIZE_PADDING)),
    height: Math.max(GROUP_MIN_HEIGHT, Math.round(maxBottom + GROUP_RESIZE_PADDING)),
  };
}

/**
 * When a child node moves or its dimensions change, the parent group may need
 * to expand. This detects affected groups and resizes only those groups.
 */
export function resizeGroupsAfterChildChanges<TNode extends ReactFlowNode>(
  changes: NodeChange[],
  nodes: TNode[],
  setNodes: (updater: (nodes: TNode[]) => TNode[]) => void,
) {
  const childChangedIds = new Set(
    changes.filter((change) => change.type === "dimensions" || change.type === "position").map((change) => change.id),
  );
  if (childChangedIds.size === 0) {
    return;
  }

  const affectedGroupIds = new Set<string>();
  for (const node of nodes) {
    if (node.parentId && childChangedIds.has(node.id)) {
      affectedGroupIds.add(node.parentId);
    }
  }
  if (affectedGroupIds.size === 0) {
    return;
  }

  setNodes((currentNodes) => {
    let changed = false;
    const updated = currentNodes.map((node) => {
      if (!affectedGroupIds.has(node.id)) {
        return node;
      }

      const size = computeGroupSizeFromChildren(node.id, currentNodes);
      if (!size) {
        return node;
      }

      const currentW = node.width ?? 0;
      const currentH = node.height ?? 0;
      if (Math.abs(currentW - size.width) < 1 && Math.abs(currentH - size.height) < 1) {
        return node;
      }

      changed = true;
      return {
        ...node,
        width: size.width,
        height: size.height,
        style: { ...node.style, width: size.width, height: size.height, zIndex: -1 },
      };
    });

    return changed ? updated : currentNodes;
  });
}
