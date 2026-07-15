type PositionLike = {
  x?: number;
  y?: number;
};

type LayoutEdge = {
  sourceId?: string;
  targetId?: string;
  channel?: string;
};

type LayoutNode = {
  id?: string;
  position?: PositionLike;
};

export function resolveForwardLayoutEdges<T extends LayoutEdge>(layoutNodes: LayoutNode[], layoutEdges: T[]): T[] {
  const layoutNodeIds = new Set(layoutNodes.map((node) => node.id).filter((id): id is string => Boolean(id)));
  const forwardEdges: T[] = [];

  for (const edge of layoutEdges) {
    if (!edge.sourceId || !edge.targetId) {
      continue;
    }

    if (!layoutNodeIds.has(edge.sourceId) || !layoutNodeIds.has(edge.targetId)) {
      continue;
    }

    if (hasLayoutPath(forwardEdges, edge.targetId, edge.sourceId)) {
      continue;
    }

    forwardEdges.push(edge);
  }

  return forwardEdges;
}

export function appendUniqueChannels(first: string[], second: string[]): string[] {
  const seen = new Set<string>();
  const result: string[] = [];

  for (const channel of [...first, ...second]) {
    if (seen.has(channel)) {
      continue;
    }

    seen.add(channel);
    result.push(channel);
  }

  return result;
}

function hasLayoutPath(layoutEdges: LayoutEdge[], startNodeId: string, targetNodeId: string): boolean {
  const adjacencyByNodeId = new Map<string, string[]>();
  layoutEdges.forEach((edge) => {
    if (!edge.sourceId || !edge.targetId) {
      return;
    }

    adjacencyByNodeId.set(edge.sourceId, [...(adjacencyByNodeId.get(edge.sourceId) || []), edge.targetId]);
  });

  const visited = new Set<string>();
  const queue = [startNodeId];
  while (queue.length > 0) {
    const current = queue.shift();
    if (!current) {
      continue;
    }
    if (current === targetNodeId) {
      return true;
    }
    if (visited.has(current)) {
      continue;
    }

    visited.add(current);
    queue.push(...(adjacencyByNodeId.get(current) || []));
  }

  return false;
}
