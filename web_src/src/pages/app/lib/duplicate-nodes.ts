import type { ComponentsEdge, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { generateNodeId, generateUniqueNodeName } from "../utils";

type DuplicatedNodesResult = {
  newNodes: ComponentsNode[];
  nodeIdMap: Record<string, string>;
};

function duplicateBaseName(node: ComponentsNode): string {
  const trimmedName = node.name?.trim();
  if (trimmedName) return trimmedName;
  if ((node.type === "TYPE_TRIGGER" || node.type === "TYPE_ACTION") && node.component) return node.component;
  return "node";
}

export function buildDuplicatedNodes(specNodes: ComponentsNode[], nodeIds: string[]): DuplicatedNodesResult {
  const existingNodeNames = specNodes.map((node) => node.name || "").filter(Boolean);
  const newNodes: ComponentsNode[] = [];
  const nodeIdMap: Record<string, string> = {};

  for (const nodeId of nodeIds) {
    const nodeToDuplicate = specNodes.find((node) => node.id === nodeId);
    if (!nodeToDuplicate) continue;

    const baseName = duplicateBaseName(nodeToDuplicate);
    const allNames = [...existingNodeNames, ...newNodes.map((node) => node.name || "")];
    const uniqueNodeName = generateUniqueNodeName(baseName, allNames);
    const newNodeId = generateNodeId(baseName, uniqueNodeName);

    nodeIdMap[nodeId] = newNodeId;
    newNodes.push({
      ...nodeToDuplicate,
      id: newNodeId,
      name: uniqueNodeName,
      position: {
        x: (nodeToDuplicate.position?.x || 0) + 50,
        y: (nodeToDuplicate.position?.y || 0) + 50,
      },
      isCollapsed: false,
    });
  }

  return { newNodes, nodeIdMap };
}

export function buildDuplicatedEdges(
  edges: ComponentsEdge[],
  duplicatedNodeIds: Set<string>,
  nodeIdMap: Record<string, string>,
): ComponentsEdge[] {
  return edges
    .filter(
      (edge) =>
        edge.sourceId != null &&
        edge.targetId != null &&
        duplicatedNodeIds.has(edge.sourceId) &&
        duplicatedNodeIds.has(edge.targetId),
    )
    .map((edge) => ({
      ...edge,
      sourceId: nodeIdMap[edge.sourceId!],
      targetId: nodeIdMap[edge.targetId!],
    }));
}
