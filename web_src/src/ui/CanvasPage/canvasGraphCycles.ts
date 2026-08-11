import type { SuperplaneComponentsNode as ComponentsNode } from "@/api-client";

/**
 * The loop component is the one component whose input may close a cycle, which
 * is what makes iteration expressible on the canvas.
 */
const LOOP_COMPONENT_NAME = "loop";

type GraphEdge = {
  source?: string | null;
  target?: string | null;
};

export function isLoopNode(node: ComponentsNode): boolean {
  return node.component === LOOP_COMPONENT_NAME;
}

function loopNodeIds(nodes: ComponentsNode[]): Set<string> {
  const ids = new Set<string>();
  for (const node of nodes) {
    if (node.id && isLoopNode(node)) {
      ids.add(node.id);
    }
  }

  return ids;
}

/**
 * Builds the adjacency used for cycle detection, leaving out every edge that
 * points at a loop node for the same reason the server does.
 */
function buildAdjacency(edges: readonly GraphEdge[], loopIds: Set<string>): Map<string, string[]> {
  const outgoing = new Map<string, string[]>();

  for (const edge of edges) {
    const { source, target } = edge;
    if (!source || !target || loopIds.has(target)) {
      continue;
    }

    const existing = outgoing.get(source);
    if (existing) {
      existing.push(target);
      continue;
    }

    outgoing.set(source, [target]);
  }

  return outgoing;
}

function canReach(outgoing: Map<string, string[]>, from: string, to: string): boolean {
  const queue = [from];
  const seen = new Set<string>([from]);

  while (queue.length > 0) {
    const current = queue.shift()!;
    if (current === to) {
      return true;
    }

    for (const next of outgoing.get(current) ?? []) {
      if (seen.has(next)) {
        continue;
      }

      seen.add(next);
      queue.push(next);
    }
  }

  return false;
}

/**
 * Reports whether connecting sourceId to targetId would close a cycle.
 *
 * This mirrors the server-side check in
 * pkg/grpc/actions/canvases/changesets/common.go (CheckForCycles): edges whose
 * *target* is a loop node are left out of the graph entirely, so a loop node's
 * input may close a cycle while every other component's may not. Keeping the
 * two rules identical is the point — the canvas should refuse exactly the
 * connections the server would later reject at publish time, no more and no
 * less.
 *
 * The server runs a topological sort over the whole graph. Here a single edge
 * is being added to a graph that is already acyclic, so it is enough to ask
 * whether the target can already reach the source.
 */
export function wouldCreateCycle(
  nodes: ComponentsNode[],
  edges: readonly GraphEdge[],
  sourceId: string,
  targetId: string,
): boolean {
  if (!sourceId || !targetId) {
    return false;
  }

  const loopIds = loopNodeIds(nodes);

  // The edge being added would itself be excluded from the graph, so it cannot
  // introduce a cycle no matter what the rest of the graph looks like.
  if (loopIds.has(targetId)) {
    return false;
  }

  if (sourceId === targetId) {
    return true;
  }

  return canReach(buildAdjacency(edges, loopIds), targetId, sourceId);
}
