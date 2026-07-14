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
  const positionByNodeId = new Map(layoutNodes.map((node) => [node.id as string, node.position]));

  return layoutEdges.filter((edge, index) => {
    if (!edge.sourceId || !edge.targetId) {
      return false;
    }

    const sourcePosition = positionByNodeId.get(edge.sourceId);
    const targetPosition = positionByNodeId.get(edge.targetId);
    if (!sourcePosition || !targetPosition) {
      return false;
    }

    return !(
      isBackwardLayoutEdge(sourcePosition, targetPosition) &&
      hasLayoutPath(layoutEdges, edge.targetId, edge.sourceId, index)
    );
  });
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

function isBackwardLayoutEdge(sourcePosition: PositionLike, targetPosition: PositionLike): boolean {
  const sourceX = sourcePosition.x ?? 0;
  const sourceY = sourcePosition.y ?? 0;
  const targetX = targetPosition.x ?? 0;
  const targetY = targetPosition.y ?? 0;

  if (targetX !== sourceX) {
    return targetX < sourceX;
  }

  return targetY < sourceY;
}

function hasLayoutPath(
  layoutEdges: LayoutEdge[],
  startNodeId: string,
  targetNodeId: string,
  excludedEdgeIndex: number,
): boolean {
  const adjacencyByNodeId = new Map<string, string[]>();
  layoutEdges.forEach((edge, index) => {
    if (index === excludedEdgeIndex || !edge.sourceId || !edge.targetId) {
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
