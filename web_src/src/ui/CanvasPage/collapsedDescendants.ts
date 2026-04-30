import type { Edge as ReactFlowEdge, Node as ReactFlowNode } from "@xyflow/react";

type CollapsibleNodeData = {
  type?: unknown;
  component?: { collapsed?: boolean };
  trigger?: { collapsed?: boolean };
  composite?: { collapsed?: boolean };
};

export function isCanvasNodeCollapsed(node: ReactFlowNode | undefined): boolean {
  const data = node?.data as CollapsibleNodeData | undefined;
  if (!data) {
    return false;
  }

  const nodeType = data?.type;
  if (nodeType !== "component" && nodeType !== "trigger" && nodeType !== "composite") {
    return false;
  }

  return Boolean(data[nodeType]?.collapsed);
}

export function getCollapsedDescendantNodeIds(nodes: ReactFlowNode[], edges: ReactFlowEdge[]): Set<string> {
  const nodeIds = new Set(nodes.map((node) => node.id));
  const collapsedNodeIds = nodes.filter(isCanvasNodeCollapsed).map((node) => node.id);

  if (collapsedNodeIds.length === 0 || edges.length === 0) {
    return new Set();
  }

  const outgoingNodeIdsBySourceId = new Map<string, string[]>();
  for (const edge of edges) {
    if (!nodeIds.has(edge.source) || !nodeIds.has(edge.target)) {
      continue;
    }

    const targetIds = outgoingNodeIdsBySourceId.get(edge.source) ?? [];
    targetIds.push(edge.target);
    outgoingNodeIdsBySourceId.set(edge.source, targetIds);
  }

  const hiddenNodeIds = new Set<string>();
  for (const collapsedNodeId of collapsedNodeIds) {
    const visitedNodeIds = new Set([collapsedNodeId]);
    const queue = [...(outgoingNodeIdsBySourceId.get(collapsedNodeId) ?? [])];

    while (queue.length > 0) {
      const nodeId = queue.shift();
      if (!nodeId || visitedNodeIds.has(nodeId)) {
        continue;
      }

      visitedNodeIds.add(nodeId);
      hiddenNodeIds.add(nodeId);
      queue.push(...(outgoingNodeIdsBySourceId.get(nodeId) ?? []));
    }
  }

  return hiddenNodeIds;
}
