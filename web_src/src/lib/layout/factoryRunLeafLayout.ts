/**
 * Ephemeral factory run-inspection layout (layered DAG).
 *
 * Sugiyama-inspired:
 * - Rank nodes by longest-path layer from roots.
 * - Spine column = deepest non-leaf chain; forks get columns to the right.
 * - Each node placed once (merges share one slot).
 * - Side vs spine edge class is decided from final geometry only.
 *
 * Does not mutate saved workflow positions.
 */

export const FACTORY_SIDE_HANDLE_ID = "__factorySide";
/** Centered bottom handle for spine edges when MultiBottom stems are suppressed. */
export const FACTORY_SPINE_HANDLE_ID = "__factorySpine";

export type FactoryRunLayoutNode = {
  id: string;
  width?: number;
  height?: number;
};

export type FactoryRunLayoutEdge = {
  id?: string;
  source: string;
  target: string;
  sourceHandle?: string | null;
};

export type FactoryRunLayoutPosition = { x: number; y: number };

export type FactoryRunLeafLayoutResult = {
  positions: Map<string, FactoryRunLayoutPosition>;
  /** Parents that need a Right side handle for off-spine fan-out. */
  sideHandleNodeIds: Set<string>;
  /** Parents that use compact center+side chrome (no MultiBottom true/false stems). */
  compactForkNodeIds: Set<string>;
  /** Compact-fork parents that still have a spine child (centered bottom handle). */
  spineSourceNodeIds: Set<string>;
  /** Edge keys that route side→left. */
  leafEdgeKeys: Set<string>;
  /** Edge keys that stay on the spine (center bottom → top when compact). */
  spineEdgeKeys: Set<string>;
  /** Targets of side edges — use Left target handle. */
  sideTargetNodeIds: Set<string>;
  /** Optional right-gutter X for long / cross-column edges (edge key → x). */
  edgeRouteGutters: Map<string, number>;
};

const DEFAULT_NODE_WIDTH = 280;
const DEFAULT_NODE_HEIGHT = 104;
const MAIN_X = 120;
const SIDE_GAP = 96;
const VERTICAL_GAP = 104;
const COMPONENT_GAP_X = 120;
const GUTTER_PAD = 48;
/** Target must sit at least this far right of source to count as a side fan-out. */
const SIDE_X_THRESHOLD = SIDE_GAP / 2;

/** Prefer these channels on the spine when path lengths tie. */
const SPINE_CHANNEL_PRIORITY: Record<string, number> = {
  default: 0,
  success: 1,
  true: 2,
  passed: 3,
  yes: 4,
  false: 80,
  failed: 90,
  error: 100,
  no: 110,
};

export function factoryRunLeafEdgeKey(source: string, target: string, sourceHandle?: string | null): string {
  return `${source}\0${target}\0${sourceHandle ?? "default"}`;
}

function nodeSize(node: FactoryRunLayoutNode): { width: number; height: number } {
  return {
    width: node.width && node.width > 0 ? node.width : DEFAULT_NODE_WIDTH,
    height: node.height && node.height > 0 ? node.height : DEFAULT_NODE_HEIGHT,
  };
}

function channelPriority(channel: string | null | undefined): number {
  const key = (channel ?? "default").toLowerCase();
  return SPINE_CHANNEL_PRIORITY[key] ?? 50;
}

function compareChildEdges(a: FactoryRunLayoutEdge, b: FactoryRunLayoutEdge): number {
  const byChannel = channelPriority(a.sourceHandle) - channelPriority(b.sourceHandle);
  if (byChannel !== 0) return byChannel;
  const handleA = a.sourceHandle ?? "default";
  const handleB = b.sourceHandle ?? "default";
  const byHandle = handleA.localeCompare(handleB);
  if (byHandle !== 0) return byHandle;
  return a.target.localeCompare(b.target);
}

function hasPath(
  adjacency: Map<string, string[]>,
  start: string,
  target: string,
  visiting = new Set<string>(),
): boolean {
  if (start === target) return true;
  if (visiting.has(start)) return false;
  visiting.add(start);
  for (const next of adjacency.get(start) ?? []) {
    if (hasPath(adjacency, next, target, visiting)) {
      return true;
    }
  }
  visiting.delete(start);
  return false;
}

/** Drop edges that would close a cycle (same idea as resolveForwardLayoutEdges). */
function resolveForwardEdges(edges: FactoryRunLayoutEdge[], nodeIds: Set<string>): FactoryRunLayoutEdge[] {
  const forward: FactoryRunLayoutEdge[] = [];
  const adjacency = new Map<string, string[]>();

  for (const edge of edges) {
    if (!nodeIds.has(edge.source) || !nodeIds.has(edge.target)) {
      continue;
    }
    if (hasPath(adjacency, edge.target, edge.source)) {
      continue;
    }
    forward.push(edge);
    adjacency.set(edge.source, [...(adjacency.get(edge.source) ?? []), edge.target]);
  }

  return forward;
}

export function layoutFactoryRunLeafGraph(
  nodes: FactoryRunLayoutNode[],
  edges: FactoryRunLayoutEdge[],
): FactoryRunLeafLayoutResult {
  const positions = new Map<string, FactoryRunLayoutPosition>();
  const sideHandleNodeIds = new Set<string>();
  const compactForkNodeIds = new Set<string>();
  const spineSourceNodeIds = new Set<string>();
  const leafEdgeKeys = new Set<string>();
  const spineEdgeKeys = new Set<string>();
  const sideTargetNodeIds = new Set<string>();
  const edgeRouteGutters = new Map<string, number>();

  if (nodes.length === 0) {
    return {
      positions,
      sideHandleNodeIds,
      compactForkNodeIds,
      spineSourceNodeIds,
      leafEdgeKeys,
      spineEdgeKeys,
      sideTargetNodeIds,
      edgeRouteGutters,
    };
  }

  const nodeById = new Map(nodes.map((node) => [node.id, node]));
  const nodeIds = new Set(nodes.map((node) => node.id));
  const forwardEdges = resolveForwardEdges(edges, nodeIds);

  const outgoing = new Map<string, FactoryRunLayoutEdge[]>();
  const incoming = new Map<string, FactoryRunLayoutEdge[]>();
  for (const node of nodes) {
    outgoing.set(node.id, []);
    incoming.set(node.id, []);
  }
  for (const edge of forwardEdges) {
    outgoing.get(edge.source)!.push(edge);
    incoming.get(edge.target)!.push(edge);
  }
  for (const [id, childEdges] of outgoing) {
    outgoing.set(id, [...childEdges].sort(compareChildEdges));
  }

  const outDegree = (id: string) => outgoing.get(id)?.length ?? 0;
  const isLeaf = (id: string) => outDegree(id) === 0;

  const depthMemo = new Map<string, number>();
  const depthVisiting = new Set<string>();

  function longestDepth(id: string): number {
    const cached = depthMemo.get(id);
    if (cached !== undefined) return cached;
    if (depthVisiting.has(id)) return 0;
    depthVisiting.add(id);
    let best = 0;
    for (const edge of outgoing.get(id) ?? []) {
      best = Math.max(best, 1 + longestDepth(edge.target));
    }
    depthVisiting.delete(id);
    depthMemo.set(id, best);
    return best;
  }

  function pickMainChildEdge(childEdges: FactoryRunLayoutEdge[]): FactoryRunLayoutEdge | null {
    const nonLeaf = childEdges.filter((edge) => !isLeaf(edge.target));
    if (nonLeaf.length === 0) return null;

    let best = nonLeaf[0];
    let bestDepth = longestDepth(best.target);
    for (let i = 1; i < nonLeaf.length; i++) {
      const edge = nonLeaf[i];
      const depth = longestDepth(edge.target);
      if (depth > bestDepth) {
        best = edge;
        bestDepth = depth;
        continue;
      }
      if (depth < bestDepth) continue;
      if (compareChildEdges(edge, best) < 0) {
        best = edge;
      }
    }
    return best;
  }

  function isMainSuccessor(parentId: string, childId: string): boolean {
    const main = pickMainChildEdge(outgoing.get(parentId) ?? []);
    return main?.target === childId;
  }

  // Weakly connected components.
  const undirected = new Map<string, Set<string>>();
  for (const node of nodes) {
    undirected.set(node.id, new Set());
  }
  for (const edge of forwardEdges) {
    undirected.get(edge.source)!.add(edge.target);
    undirected.get(edge.target)!.add(edge.source);
  }

  const roots = nodes.filter((node) => (incoming.get(node.id) ?? []).length === 0).map((node) => node.id);
  const startIds = roots.length > 0 ? roots : [nodes[0].id];

  const components: string[][] = [];
  const componentSeen = new Set<string>();
  for (const start of startIds) {
    if (componentSeen.has(start)) continue;
    const stack = [start];
    const component: string[] = [];
    componentSeen.add(start);
    while (stack.length > 0) {
      const id = stack.pop()!;
      component.push(id);
      for (const neighbor of undirected.get(id) ?? []) {
        if (componentSeen.has(neighbor)) continue;
        componentSeen.add(neighbor);
        stack.push(neighbor);
      }
    }
    components.push(component);
  }
  for (const node of nodes) {
    if (componentSeen.has(node.id)) continue;
    components.push([node.id]);
    componentSeen.add(node.id);
  }

  let componentOriginX = MAIN_X;

  for (const component of components) {
    const componentSet = new Set(component);
    const componentRoots = component.filter((id) => (incoming.get(id) ?? []).every((e) => !componentSet.has(e.source)));
    const rootsForComponent = componentRoots.length > 0 ? componentRoots : [component[0]];

    // Layer = longest path from any component root.
    const layer = new Map<string, number>();
    for (const id of component) {
      layer.set(id, 0);
    }

    let changed = true;
    let guard = 0;
    while (changed && guard < component.length + 2) {
      changed = false;
      guard += 1;
      for (const id of component) {
        let best = rootsForComponent.includes(id) ? 0 : 0;
        const parents = (incoming.get(id) ?? []).filter((e) => componentSet.has(e.source));
        if (parents.length === 0) {
          best = 0;
        } else {
          best = Math.max(...parents.map((e) => (layer.get(e.source) ?? 0) + 1));
        }
        if ((layer.get(id) ?? 0) !== best) {
          layer.set(id, best);
          changed = true;
        }
      }
    }

    // Spine = deepest non-leaf chain from each root.
    const spine = new Set<string>();
    for (const rootId of rootsForComponent) {
      let current: string | null = rootId;
      while (current) {
        spine.add(current);
        const main = pickMainChildEdge((outgoing.get(current) ?? []).filter((e) => componentSet.has(e.target)));
        current = main?.target ?? null;
        if (current && spine.has(current)) {
          break;
        }
      }
    }

    // Column assignment (DAG, each node once).
    const column = new Map<string, number>();
    for (const id of spine) {
      column.set(id, 0);
    }

    const byLayer = [...component].sort((a, b) => (layer.get(a) ?? 0) - (layer.get(b) ?? 0) || a.localeCompare(b));
    for (const id of byLayer) {
      if (column.has(id)) continue;
      const parents = (incoming.get(id) ?? []).filter((e) => componentSet.has(e.source));
      if (parents.length === 0) {
        column.set(id, 0);
        continue;
      }
      const parentCols = parents.map((edge) => column.get(edge.source) ?? 0);
      // Merges / joins: sit under leftmost parent (prefer spine col 0).
      if (parents.length > 1) {
        column.set(id, Math.min(...parentCols));
        continue;
      }
      const parentCol = parentCols[0];
      const parentId = parents[0].source;
      column.set(id, isMainSuccessor(parentId, id) ? parentCol : parentCol + 1);
    }

    // Coordinates: layer base Y, pack per column so stacked siblings never collide.
    let maxRight = componentOriginX;
    const strideY = DEFAULT_NODE_HEIGHT + VERTICAL_GAP;
    const strideX = DEFAULT_NODE_WIDTH + SIDE_GAP;

    const byColumn = new Map<number, string[]>();
    for (const id of component) {
      const col = column.get(id) ?? 0;
      const list = byColumn.get(col) ?? [];
      list.push(id);
      byColumn.set(col, list);
    }

    for (const [col, ids] of byColumn) {
      ids.sort((a, b) => (layer.get(a) ?? 0) - (layer.get(b) ?? 0) || a.localeCompare(b));
      const x = componentOriginX + col * strideX;
      let nextY = 0;
      for (const id of ids) {
        const lyr = layer.get(id) ?? 0;
        const size = nodeSize(nodeById.get(id)!);
        const y = Math.max(lyr * strideY, nextY);
        positions.set(id, { x, y });
        maxRight = Math.max(maxRight, x + size.width);
        nextY = y + size.height + VERTICAL_GAP;
      }
    }

    // Geometry-based edge class + gutters for this component.
    const componentEdges = forwardEdges.filter((e) => componentSet.has(e.source) && componentSet.has(e.target));
    const graphRight = maxRight + GUTTER_PAD;

    for (const edge of componentEdges) {
      const sourcePos = positions.get(edge.source);
      const targetPos = positions.get(edge.target);
      if (!sourcePos || !targetPos) continue;

      const key = factoryRunLeafEdgeKey(edge.source, edge.target, edge.sourceHandle);
      const isSide = targetPos.x >= sourcePos.x + SIDE_X_THRESHOLD;

      if (isSide) {
        leafEdgeKeys.add(key);
        sideHandleNodeIds.add(edge.source);
        sideTargetNodeIds.add(edge.target);
      } else {
        spineEdgeKeys.add(key);
      }

      const sourceLayer = layer.get(edge.source) ?? 0;
      const targetLayer = layer.get(edge.target) ?? 0;
      const layerSkip = targetLayer - sourceLayer > 1;
      const crossColumn = Math.abs((column.get(edge.source) ?? 0) - (column.get(edge.target) ?? 0)) > 0;

      // Long vertical or merge-style cross edges need a right gutter (not short side fan-outs).
      if (!isSide && (layerSkip || crossColumn)) {
        edgeRouteGutters.set(key, graphRight);
      }
    }

    for (const id of component) {
      const outs = outgoing.get(id) ?? [];
      const hasSide = outs.some((edge) =>
        leafEdgeKeys.has(factoryRunLeafEdgeKey(edge.source, edge.target, edge.sourceHandle)),
      );
      const hasSpine = outs.some((edge) =>
        spineEdgeKeys.has(factoryRunLeafEdgeKey(edge.source, edge.target, edge.sourceHandle)),
      );
      if (hasSide) {
        compactForkNodeIds.add(id);
        if (hasSpine) {
          spineSourceNodeIds.add(id);
        }
      }
    }

    componentOriginX = maxRight + COMPONENT_GAP_X;
  }

  return {
    positions,
    sideHandleNodeIds,
    compactForkNodeIds,
    spineSourceNodeIds,
    leafEdgeKeys,
    spineEdgeKeys,
    sideTargetNodeIds,
    edgeRouteGutters,
  };
}
