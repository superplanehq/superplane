import type { SuperplaneComponentsNode } from "@/api-client";

function isGroupNode(node: SuperplaneComponentsNode | undefined): boolean {
  return node?.type === "TYPE_WIDGET" && node.widget?.name === "group";
}

function getChildNodeIds(node: SuperplaneComponentsNode): string[] {
  const childNodeIds = node.configuration?.childNodeIds;
  if (!Array.isArray(childNodeIds)) {
    return [];
  }

  return childNodeIds.filter((value): value is string => typeof value === "string" && value.length > 0);
}

function getPositionOffset(node: SuperplaneComponentsNode | undefined): { x: number; y: number } {
  return {
    x: Number(node?.position?.x) || 0,
    y: Number(node?.position?.y) || 0,
  };
}

function buildGroupRelationships(nodes: SuperplaneComponentsNode[]): {
  groupNodesById: Map<string, SuperplaneComponentsNode>;
  parentGroupByChildId: Map<string, string>;
} {
  const groupNodesById = new Map<string, SuperplaneComponentsNode>();
  const parentGroupByChildId = new Map<string, string>();

  for (const node of nodes) {
    if (!isGroupNode(node)) {
      continue;
    }

    if (!node.id) {
      continue;
    }

    groupNodesById.set(node.id, node);
    for (const childId of getChildNodeIds(node)) {
      if (!parentGroupByChildId.has(childId)) {
        parentGroupByChildId.set(childId, node.id);
      }
    }
  }

  return { groupNodesById, parentGroupByChildId };
}

function getNodeGroupOffset(
  nodeId: string,
  groupNodesById: Map<string, SuperplaneComponentsNode>,
  parentGroupByChildId: Map<string, string>,
): { x: number; y: number } | undefined {
  let offsetX = 0;
  let offsetY = 0;
  let currentGroupId = parentGroupByChildId.get(nodeId);
  const visited = new Set<string>();

  for (; currentGroupId && !visited.has(currentGroupId); currentGroupId = parentGroupByChildId.get(currentGroupId)) {
    visited.add(currentGroupId);
    const groupNode = groupNodesById.get(currentGroupId);
    if (!groupNode) {
      break;
    }

    const groupOffset = getPositionOffset(groupNode);
    offsetX += groupOffset.x;
    offsetY += groupOffset.y;
  }

  if (offsetX === 0 && offsetY === 0) {
    return undefined;
  }

  return { x: offsetX, y: offsetY };
}

function resolveGroupOffsets(nodes: SuperplaneComponentsNode[]): Map<string, { x: number; y: number }> {
  const { groupNodesById, parentGroupByChildId } = buildGroupRelationships(nodes);
  const offsetsByNodeId = new Map<string, { x: number; y: number }>();

  for (const node of nodes) {
    if (!node.id || groupNodesById.has(node.id)) {
      continue;
    }

    const offset = getNodeGroupOffset(node.id, groupNodesById, parentGroupByChildId);
    if (offset) {
      offsetsByNodeId.set(node.id, offset);
    }
  }

  return offsetsByNodeId;
}

export function normalizeCanvasNodesWithoutGroups(
  nodes: SuperplaneComponentsNode[] | undefined,
): SuperplaneComponentsNode[] {
  if (!nodes || nodes.length === 0) {
    return nodes || [];
  }

  const hasGroups = nodes.some((node) => isGroupNode(node));
  if (!hasGroups) {
    return nodes;
  }

  const offsetsByNodeId = resolveGroupOffsets(nodes);

  return nodes
    .filter((node) => !isGroupNode(node))
    .map((node) => {
      if (!node.id) {
        return node;
      }

      const offset = offsetsByNodeId.get(node.id);
      if (!offset) {
        return node;
      }

      const position = getPositionOffset(node);
      return {
        ...node,
        position: {
          x: Math.round(position.x + offset.x),
          y: Math.round(position.y + offset.y),
        },
      };
    });
}

export function normalizeCanvasWithoutGroups<T extends { spec?: { nodes?: SuperplaneComponentsNode[] } }>(canvas: T): T;
export function normalizeCanvasWithoutGroups<T extends { spec?: { nodes?: SuperplaneComponentsNode[] } }>(
  canvas: T | null | undefined,
): T | undefined;
export function normalizeCanvasWithoutGroups<T extends { spec?: { nodes?: SuperplaneComponentsNode[] } }>(
  canvas: T | null | undefined,
): T | undefined {
  if (!canvas) {
    return undefined;
  }

  if (!canvas.spec?.nodes) {
    return canvas;
  }

  const normalizedNodes = normalizeCanvasNodesWithoutGroups(canvas.spec.nodes);
  if (normalizedNodes === canvas.spec.nodes) {
    return canvas;
  }

  return {
    ...canvas,
    spec: {
      ...canvas.spec,
      nodes: normalizedNodes,
    },
  };
}
