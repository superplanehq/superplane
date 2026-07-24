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

type GraphEdge = {
  sourceId?: string;
  targetId?: string;
};

/**
 * Builds an undirected adjacency map for the given nodes, ignoring edges that
 * dangle outside the node set or loop back onto themselves. Shared by both the
 * connected-component scoping and the disconnected-component packing logic.
 */
function buildUndirectedAdjacency(nodes: LayoutNode[], edges: GraphEdge[]): Map<string, string[]> {
  const nodeIds = new Set(nodes.map((node) => node.id).filter((id): id is string => Boolean(id)));
  const adjacencyByNodeId = new Map<string, string[]>();

  for (const id of nodeIds) {
    adjacencyByNodeId.set(id, []);
  }

  for (const edge of edges) {
    const { sourceId, targetId } = edge;
    if (!sourceId || !targetId || sourceId === targetId) {
      continue;
    }

    if (!nodeIds.has(sourceId) || !nodeIds.has(targetId)) {
      continue;
    }

    adjacencyByNodeId.get(sourceId)?.push(targetId);
    adjacencyByNodeId.get(targetId)?.push(sourceId);
  }

  return adjacencyByNodeId;
}

/**
 * Returns every node id reachable from the given seeds via a breadth-first walk
 * over the adjacency map (the seeds themselves included).
 */
function collectReachableNodeIds(adjacencyByNodeId: Map<string, string[]>, seedNodeIds: string[]): Set<string> {
  const visited = new Set<string>();
  const queue = [...seedNodeIds];

  while (queue.length > 0) {
    const current = queue.shift();
    if (!current || visited.has(current)) {
      continue;
    }

    visited.add(current);
    for (const neighbor of adjacencyByNodeId.get(current) || []) {
      if (!visited.has(neighbor)) {
        queue.push(neighbor);
      }
    }
  }

  return visited;
}

/**
 * Resolves the ids of every node in the connected component(s) that contain the
 * given seeds. With no seeds the full node set is returned unchanged.
 */
export function resolveConnectedComponentNodeIds(
  nodes: LayoutNode[],
  edges: GraphEdge[],
  seedNodeIds: string[],
): string[] {
  if (seedNodeIds.length === 0) {
    return nodes.map((node) => node.id as string);
  }

  const adjacencyByNodeId = buildUndirectedAdjacency(nodes, edges);
  const reachable = collectReachableNodeIds(adjacencyByNodeId, seedNodeIds);
  return nodes.map((node) => node.id as string).filter((id) => reachable.has(id));
}

/**
 * Partitions the nodes into their disconnected sub-graphs, preserving
 * breadth-first visitation order within each component so downstream packing
 * stays deterministic.
 */
export function resolveDisconnectedComponents<T extends LayoutNode>(nodes: T[], edges: GraphEdge[]): T[][] {
  if (nodes.length === 0) {
    return [];
  }

  const adjacencyByNodeId = buildUndirectedAdjacency(nodes, edges);
  const nodesById = new Map(nodes.map((node) => [node.id as string, node]));
  const visited = new Set<string>();
  const components: T[][] = [];

  for (const node of nodes) {
    const seedId = node.id as string;
    if (!seedId || visited.has(seedId)) {
      continue;
    }

    const collected: T[] = [];
    const queue = [seedId];
    while (queue.length > 0) {
      const current = queue.shift();
      if (!current || visited.has(current)) {
        continue;
      }

      visited.add(current);
      const currentNode = nodesById.get(current);
      if (currentNode) {
        collected.push(currentNode);
      }

      for (const neighbor of adjacencyByNodeId.get(current) || []) {
        if (!visited.has(neighbor)) {
          queue.push(neighbor);
        }
      }
    }

    if (collected.length > 0) {
      components.push(collected);
    }
  }

  return components;
}

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
