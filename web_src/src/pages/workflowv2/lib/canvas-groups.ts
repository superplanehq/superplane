import type { CanvasesCanvas, ComponentsNode } from "@/api-client";
import type { CanvasNode } from "@/ui/CanvasPage";
import { GROUP_CHILD_EDGE_PADDING, GROUP_CHILD_MIN_Y_OFFSET, normalizeGroupColor } from "@/ui/groupNode/constants";
import { estimateNodeSize } from "../autoLayout";
import { buildChildToGroupMap, collectGroupChildIds, generateNodeId, generateUniqueNodeName } from "../utils";

const DEFAULT_GROUP_WIDTH = 480;
const DEFAULT_GROUP_HEIGHT = 320;
const GROUP_SIZE_PADDING = 10;

function getNodeIdsToDeleteIncludingGroupChildren(nodes: ComponentsNode[], seedIds: string[]): Set<string> {
  const byId = new Map<string, ComponentsNode>();
  for (const node of nodes) {
    if (node.id) {
      byId.set(node.id, node);
    }
  }

  const toRemove = new Set<string>(seedIds.filter(Boolean));
  let added = true;
  while (added) {
    added = false;
    for (const id of [...toRemove]) {
      const node = byId.get(id);
      if (!node) {
        continue;
      }
      for (const childId of collectGroupChildIds(node)) {
        if (toRemove.has(childId)) {
          continue;
        }
        toRemove.add(childId);
        added = true;
      }
    }
  }

  return toRemove;
}

/**
 * Deletes the requested nodes, also removing any children of deleted groups,
 * then cleans up the remaining groups so they no longer reference deleted
 * child IDs.
 */
export function deleteNodesFromWorkflow(nodes: ComponentsNode[], seedIds: string[]): ComponentsNode[] {
  const removedIds = getNodeIdsToDeleteIncludingGroupChildren(nodes, seedIds);
  const survivingNodes = nodes.filter((node) => !node.id || !removedIds.has(node.id));

  return survivingNodes.map((node) => {
    if (node.type !== "TYPE_WIDGET" || node.widget?.name !== "group") {
      return node;
    }

    const childIds = collectGroupChildIds(node);
    const prunedChildIds = childIds.filter((id) => !removedIds.has(id));
    if (prunedChildIds.length === childIds.length) {
      return node;
    }

    return { ...node, configuration: { ...node.configuration, childNodeIds: prunedChildIds } };
  });
}

export function ungroupWorkflowNode(workflow: CanvasesCanvas, groupNodeId: string): CanvasesCanvas | null {
  const specNodes = workflow?.spec?.nodes || [];
  const groupNode = specNodes.find((node) => node.id === groupNodeId);
  if (!groupNode) {
    return null;
  }

  const childNodeIds = (groupNode.configuration?.childNodeIds as string[]) || [];
  const groupX = groupNode.position?.x || 0;
  const groupY = groupNode.position?.y || 0;

  const updatedNodes = specNodes
    .filter((node) => node.id !== groupNodeId)
    .map((node) => {
      if (!node.id || !childNodeIds.includes(node.id)) {
        return node;
      }

      return {
        ...node,
        position: {
          x: Math.round((node.position?.x || 0) + groupX),
          y: Math.round((node.position?.y || 0) + groupY),
        },
      };
    });

  return { ...workflow, spec: { ...workflow.spec, nodes: updatedNodes } };
}

export function groupWorkflowNodes(
  workflow: CanvasesCanvas,
  bounds: { x: number; y: number; width: number; height: number },
  nodePositions: Array<{ id: string; x: number; y: number }>,
): CanvasesCanvas {
  const specNodes = workflow?.spec?.nodes || [];
  const nodeIds = nodePositions.map((node) => node.id);

  const groupPadding = 40;
  const groupLabelHeight = 72;
  const groupX = Math.round(bounds.x - groupPadding);
  const groupY = Math.round(bounds.y - groupPadding - groupLabelHeight);

  const existingNodeNames = specNodes.map((node) => node.name || "");
  const uniqueNodeName = generateUniqueNodeName("group", existingNodeNames);
  const newGroupId = generateNodeId("group", uniqueNodeName);

  const groupNode: ComponentsNode = {
    id: newGroupId,
    name: uniqueNodeName,
    type: "TYPE_WIDGET",
    widget: { name: "group" },
    configuration: {
      label: "Group",
      description: "",
      color: "purple",
      childNodeIds: nodeIds,
    },
    position: { x: groupX, y: groupY },
  };

  const absolutePositionMap = new Map(nodePositions.map((node) => [node.id, { x: node.x, y: node.y }]));
  const updatedNodes = specNodes.map((node) => {
    if (!node.id || !nodeIds.includes(node.id)) {
      return node;
    }

    const absolutePosition = absolutePositionMap.get(node.id);
    if (!absolutePosition) {
      return node;
    }

    return {
      ...node,
      position: {
        x: Math.round(absolutePosition.x - groupX),
        y: Math.round(absolutePosition.y - groupY),
      },
    };
  });

  return { ...workflow, spec: { ...workflow.spec, nodes: [groupNode, ...updatedNodes] } };
}

export function wireGroupParentChildRelationships(workflow: CanvasesCanvas, nodes: CanvasNode[]): CanvasNode[] {
  const groupChildMap = buildChildToGroupMap(workflow?.spec?.nodes || []);

  const wiredNodes = nodes.map((node) => {
    const parentId = groupChildMap.get(node.id);
    if (!parentId) return node;

    const x = Math.max(node.position?.x ?? 0, GROUP_CHILD_EDGE_PADDING);
    const y = Math.max(node.position?.y ?? 0, GROUP_CHILD_MIN_Y_OFFSET);
    return { ...node, parentId, position: { x, y } };
  });

  const groupNodeIds = new Set(wiredNodes.filter((node) => node.data?.type === "group").map((node) => node.id));
  return [
    ...wiredNodes.filter((node) => groupNodeIds.has(node.id)),
    ...wiredNodes.filter((node) => !groupNodeIds.has(node.id)),
  ];
}

export function buildGroupNodeData(node: ComponentsNode): CanvasNode["data"] {
  const label = node.name || "Group";

  return {
    type: "group",
    label,
    state: "pending" as const,
    outputChannels: [],
    group: {
      groupLabel: (node.configuration?.label as string) || label,
      groupDescription: (node.configuration?.description as string) || "",
      groupColor: normalizeGroupColor(node.configuration?.color as string),
    },
  };
}

export function computeGroupSize(
  groupNode: ComponentsNode,
  allNodes: ComponentsNode[],
): { width: number; height: number } {
  const childIds = collectGroupChildIds(groupNode);
  if (childIds.length === 0) return { width: DEFAULT_GROUP_WIDTH, height: DEFAULT_GROUP_HEIGHT };

  let maxX = 0;
  let maxY = 0;
  let found = false;

  for (const childId of childIds) {
    const child = allNodes.find((node) => node.id === childId);
    if (!child?.position) continue;

    found = true;
    const cx = child.position.x ?? 0;
    const cy = child.position.y ?? 0;
    const { width, height } = estimateNodeSize(child);
    if (cx + width > maxX) maxX = cx + width;
    if (cy + height > maxY) maxY = cy + height;
  }

  if (!found) return { width: DEFAULT_GROUP_WIDTH, height: DEFAULT_GROUP_HEIGHT };

  return {
    width: Math.max(DEFAULT_GROUP_WIDTH, Math.round(maxX + GROUP_SIZE_PADDING)),
    height: Math.max(DEFAULT_GROUP_HEIGHT, Math.round(maxY + GROUP_SIZE_PADDING)),
  };
}

export function prepareGroupNode(node: ComponentsNode, allNodes: ComponentsNode[]): CanvasNode {
  const { width, height } = computeGroupSize(node, allNodes);

  return {
    id: node.id!,
    type: "group",
    position: { x: node.position?.x ?? 0, y: node.position?.y ?? 0 },
    selectable: true,
    width,
    height,
    style: { width, height, zIndex: -1 },
    data: buildGroupNodeData(node),
  };
}
