import type { CanvasesCanvas, ComponentsNode } from "@/api-client";
import { collectGroupChildIds, generateNodeId, generateUniqueNodeName } from "../utils";

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
