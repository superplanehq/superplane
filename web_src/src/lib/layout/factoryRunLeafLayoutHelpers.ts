import type { FactoryRunLayoutEdge, FactoryRunLayoutNode, FactoryRunLayoutPosition } from "./factoryRunLeafLayout";
import { FACTORY_NODE_CARD_HEIGHT, FACTORY_NODE_CARD_WIDTH } from "@/lib/factoryCanvasChrome";

export function factoryRunLeafEdgeKey(source: string, target: string, sourceHandle?: string | null): string {
  return `${source}\0${target}\0${sourceHandle ?? "default"}`;
}

export const DEFAULT_NODE_WIDTH = FACTORY_NODE_CARD_WIDTH;
export const DEFAULT_NODE_HEIGHT = FACTORY_NODE_CARD_HEIGHT;
export const MAIN_X = 120;
const SIDE_GAP = 96;
export const VERTICAL_GAP = 104;
const SIDE_COLUMN_EXTRA_GAP = 88;
export const COMPONENT_GAP_X = 120;
export const GUTTER_PAD = 48;
export const SIDE_X_THRESHOLD = SIDE_GAP / 2;

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

export function nodeSize(node: FactoryRunLayoutNode): { width: number; height: number } {
  return {
    width: node.width && node.width > 0 ? node.width : DEFAULT_NODE_WIDTH,
    height: node.height && node.height > 0 ? node.height : DEFAULT_NODE_HEIGHT,
  };
}

export function factoryRunChannelPriority(channel: string | null | undefined): number {
  const key = (channel ?? "default").toLowerCase();
  return SPINE_CHANNEL_PRIORITY[key] ?? 50;
}

function compareChildEdges(a: FactoryRunLayoutEdge, b: FactoryRunLayoutEdge): number {
  const byChannel = factoryRunChannelPriority(a.sourceHandle) - factoryRunChannelPriority(b.sourceHandle);
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

export type FactoryRunEdgePartition = {
  forwardEdges: FactoryRunLayoutEdge[];
  feedbackEdges: FactoryRunLayoutEdge[];
};

export function partitionLayoutEdges(edges: FactoryRunLayoutEdge[], nodeIds: Set<string>): FactoryRunEdgePartition {
  const forward: FactoryRunLayoutEdge[] = [];
  const feedback: FactoryRunLayoutEdge[] = [];
  const adjacency = new Map<string, string[]>();

  for (const edge of edges) {
    if (!nodeIds.has(edge.source) || !nodeIds.has(edge.target)) {
      continue;
    }
    if (hasPath(adjacency, edge.target, edge.source)) {
      feedback.push(edge);
      continue;
    }
    forward.push(edge);
    adjacency.set(edge.source, [...(adjacency.get(edge.source) ?? []), edge.target]);
  }

  return { forwardEdges: forward, feedbackEdges: feedback };
}

export function buildAdjacency(
  nodes: FactoryRunLayoutNode[],
  forwardEdges: FactoryRunLayoutEdge[],
): {
  outgoing: Map<string, FactoryRunLayoutEdge[]>;
  incoming: Map<string, FactoryRunLayoutEdge[]>;
} {
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
  return { outgoing, incoming };
}

export type SpinePickers = {
  pickMainChildEdge: (childEdges: FactoryRunLayoutEdge[]) => FactoryRunLayoutEdge | null;
  isMainSuccessor: (parentId: string, childId: string) => boolean;
};

export function createSpinePickers(outgoing: Map<string, FactoryRunLayoutEdge[]>): SpinePickers {
  const isLeaf = (id: string) => (outgoing.get(id)?.length ?? 0) === 0;
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

  return { pickMainChildEdge, isMainSuccessor };
}

function buildUndirectedGraph(
  nodes: FactoryRunLayoutNode[],
  forwardEdges: FactoryRunLayoutEdge[],
): Map<string, Set<string>> {
  const undirected = new Map<string, Set<string>>();
  for (const node of nodes) {
    undirected.set(node.id, new Set());
  }
  for (const edge of forwardEdges) {
    undirected.get(edge.source)!.add(edge.target);
    undirected.get(edge.target)!.add(edge.source);
  }
  return undirected;
}

function collectConnectedComponent(
  start: string,
  undirected: Map<string, Set<string>>,
  componentSeen: Set<string>,
): string[] {
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
  return component;
}

function appendOrphanComponents(
  nodes: FactoryRunLayoutNode[],
  componentSeen: Set<string>,
  components: string[][],
): void {
  for (const node of nodes) {
    if (componentSeen.has(node.id)) continue;
    components.push([node.id]);
    componentSeen.add(node.id);
  }
}

export function findWeakComponents(
  nodes: FactoryRunLayoutNode[],
  forwardEdges: FactoryRunLayoutEdge[],
  incoming: Map<string, FactoryRunLayoutEdge[]>,
): string[][] {
  const undirected = buildUndirectedGraph(nodes, forwardEdges);
  const roots = nodes.filter((node) => (incoming.get(node.id) ?? []).length === 0).map((node) => node.id);
  const startIds = roots.length > 0 ? roots : [nodes[0].id];

  const components: string[][] = [];
  const componentSeen = new Set<string>();
  for (const start of startIds) {
    if (componentSeen.has(start)) continue;
    components.push(collectConnectedComponent(start, undirected, componentSeen));
  }
  appendOrphanComponents(nodes, componentSeen, components);
  return components;
}

export function computeComponentLayers(
  component: string[],
  componentSet: Set<string>,
  rootsForComponent: string[],
  incoming: Map<string, FactoryRunLayoutEdge[]>,
): Map<string, number> {
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
  return layer;
}

export function computeComponentSpineColumns(
  rootsForComponent: string[],
  componentSet: Set<string>,
  outgoing: Map<string, FactoryRunLayoutEdge[]>,
  pickMainChildEdge: SpinePickers["pickMainChildEdge"],
): Map<string, number> {
  const spineColumns = new Map<string, number>();
  rootsForComponent.forEach((rootId, rootColumn) => {
    let current: string | null = rootId;
    const visited = new Set<string>();
    while (current) {
      const currentColumn = spineColumns.get(current);
      if (currentColumn == null || rootColumn < currentColumn) {
        spineColumns.set(current, rootColumn);
      }
      visited.add(current);
      const main = pickMainChildEdge((outgoing.get(current) ?? []).filter((e) => componentSet.has(e.target)));
      current = main?.target ?? null;
      if (current && visited.has(current)) {
        break;
      }
    }
  });
  return spineColumns;
}

type AssignComponentColumnsOptions = {
  component: string[];
  componentSet: Set<string>;
  spineColumns: Map<string, number>;
  firstSideColumn: number;
  layer: Map<string, number>;
  incoming: Map<string, FactoryRunLayoutEdge[]>;
  isMainSuccessor: SpinePickers["isMainSuccessor"];
};

export function assignComponentColumns(options: AssignComponentColumnsOptions): Map<string, number> {
  const { component, componentSet, spineColumns, firstSideColumn, layer, incoming, isMainSuccessor } = options;
  const column = new Map(spineColumns);

  const byLayer = [...component].sort((a, b) => (layer.get(a) ?? 0) - (layer.get(b) ?? 0) || a.localeCompare(b));
  for (const id of byLayer) {
    if (column.has(id)) continue;
    const parents = (incoming.get(id) ?? []).filter((e) => componentSet.has(e.source));
    if (parents.length === 0) {
      column.set(id, 0);
      continue;
    }
    const parentCols = parents.map((edge) => column.get(edge.source) ?? 0);
    if (parents.length > 1) {
      column.set(id, Math.min(...parentCols));
      continue;
    }
    const parentCol = parentCols[0];
    const parentId = parents[0].source;
    column.set(id, isMainSuccessor(parentId, id) ? parentCol : Math.max(parentCol + 1, firstSideColumn));
  }
  return column;
}

function parentYsBeside(
  id: string,
  componentSet: Set<string>,
  incoming: Map<string, FactoryRunLayoutEdge[]>,
  positions: Map<string, FactoryRunLayoutPosition>,
): number[] {
  return (incoming.get(id) ?? [])
    .filter((edge) => componentSet.has(edge.source))
    .map((edge) => positions.get(edge.source)?.y)
    .filter((y): y is number => y != null);
}

function groupIdsByColumn(component: string[], column: Map<string, number>): Map<number, string[]> {
  const byColumn = new Map<number, string[]>();
  for (const id of component) {
    const col = column.get(id) ?? 0;
    const list = byColumn.get(col) ?? [];
    list.push(id);
    byColumn.set(col, list);
  }
  return byColumn;
}

type PlaceColumnNodeOptions = {
  id: string;
  x: number;
  col: number;
  preferredY: number;
  nextY: number;
  size: { width: number; height: number };
  componentSet: Set<string>;
  incoming: Map<string, FactoryRunLayoutEdge[]>;
  positions: Map<string, FactoryRunLayoutPosition>;
  isSpineNode: boolean;
};

function placeOneColumnNode(options: PlaceColumnNodeOptions): { y: number; right: number } {
  const { id, x, col, nextY, size, componentSet, incoming, positions, isSpineNode } = options;
  let preferredY = options.preferredY;
  if (col > 0 && !isSpineNode) {
    const parentYs = parentYsBeside(id, componentSet, incoming, positions);
    if (parentYs.length > 0) {
      preferredY = Math.min(...parentYs);
    }
  }
  const y = Math.max(preferredY, nextY);
  positions.set(id, { x, y });
  return { y, right: x + size.width };
}

type PlaceComponentNodesOptions = {
  component: string[];
  componentSet: Set<string>;
  column: Map<string, number>;
  layer: Map<string, number>;
  componentOriginX: number;
  nodeById: Map<string, FactoryRunLayoutNode>;
  incoming: Map<string, FactoryRunLayoutEdge[]>;
  positions: Map<string, FactoryRunLayoutPosition>;
  spineColumns: Map<string, number>;
};

export function placeComponentNodes(options: PlaceComponentNodesOptions): number {
  const { component, componentSet, column, layer, componentOriginX, nodeById, incoming, positions, spineColumns } =
    options;
  let maxRight = componentOriginX;
  const strideY = DEFAULT_NODE_HEIGHT + VERTICAL_GAP;
  const strideX = DEFAULT_NODE_WIDTH + SIDE_GAP;
  const byColumn = groupIdsByColumn(component, column);

  const columnsAsc = [...byColumn.keys()].sort((a, b) => a - b);
  for (const col of columnsAsc) {
    const ids = byColumn.get(col)!;
    ids.sort((a, b) => (layer.get(a) ?? 0) - (layer.get(b) ?? 0) || a.localeCompare(b));
    const x = componentOriginX + col * strideX;
    const hasSpineNodes = ids.some((id) => spineColumns.has(id));
    const gapY = hasSpineNodes ? VERTICAL_GAP : VERTICAL_GAP + SIDE_COLUMN_EXTRA_GAP;
    let nextY = 0;
    for (const id of ids) {
      const size = nodeSize(nodeById.get(id)!);
      const placed = placeOneColumnNode({
        id,
        x,
        col,
        preferredY: (layer.get(id) ?? 0) * strideY,
        nextY,
        size,
        componentSet,
        incoming,
        positions,
        isSpineNode: spineColumns.has(id),
      });
      maxRight = Math.max(maxRight, placed.right);
      nextY = placed.y + size.height + gapY;
    }
  }
  return maxRight;
}
