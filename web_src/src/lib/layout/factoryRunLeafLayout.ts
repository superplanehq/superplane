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

import {
  COMPONENT_GAP_X,
  GUTTER_PAD,
  MAIN_X,
  assignComponentColumns,
  buildAdjacency,
  classifyComponentEdges,
  computeComponentLayers,
  computeComponentSpine,
  createSpinePickers,
  findWeakComponents,
  markCompactForkNodes,
  placeComponentNodes,
  resolveForwardEdges,
  type SpinePickers,
} from "./factoryRunLeafLayoutHelpers";

export { factoryRunLeafEdgeKey } from "./factoryRunLeafLayoutHelpers";

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

type ComponentLayoutContext = {
  outgoing: Map<string, FactoryRunLayoutEdge[]>;
  incoming: Map<string, FactoryRunLayoutEdge[]>;
  forwardEdges: FactoryRunLayoutEdge[];
  nodeById: Map<string, FactoryRunLayoutNode>;
  pickers: SpinePickers;
  positions: Map<string, FactoryRunLayoutPosition>;
  sideHandleNodeIds: Set<string>;
  compactForkNodeIds: Set<string>;
  spineSourceNodeIds: Set<string>;
  leafEdgeKeys: Set<string>;
  spineEdgeKeys: Set<string>;
  sideTargetNodeIds: Set<string>;
  edgeRouteGutters: Map<string, number>;
};

function layoutOneComponent(component: string[], componentOriginX: number, ctx: ComponentLayoutContext): number {
  const componentSet = new Set(component);
  const componentRoots = component.filter((id) =>
    (ctx.incoming.get(id) ?? []).every((e) => !componentSet.has(e.source)),
  );
  const rootsForComponent = componentRoots.length > 0 ? componentRoots : [component[0]];

  const layer = computeComponentLayers(component, componentSet, rootsForComponent, ctx.incoming);
  const spine = computeComponentSpine(rootsForComponent, componentSet, ctx.outgoing, ctx.pickers.pickMainChildEdge);
  const column = assignComponentColumns({
    component,
    componentSet,
    spine,
    layer,
    incoming: ctx.incoming,
    isMainSuccessor: ctx.pickers.isMainSuccessor,
  });

  const maxRight = placeComponentNodes({
    component,
    componentSet,
    column,
    layer,
    componentOriginX,
    nodeById: ctx.nodeById,
    incoming: ctx.incoming,
    positions: ctx.positions,
  });

  const componentEdges = ctx.forwardEdges.filter((e) => componentSet.has(e.source) && componentSet.has(e.target));
  classifyComponentEdges({
    componentEdges,
    positions: ctx.positions,
    layer,
    column,
    graphRight: maxRight + GUTTER_PAD,
    leafEdgeKeys: ctx.leafEdgeKeys,
    spineEdgeKeys: ctx.spineEdgeKeys,
    sideHandleNodeIds: ctx.sideHandleNodeIds,
    sideTargetNodeIds: ctx.sideTargetNodeIds,
    edgeRouteGutters: ctx.edgeRouteGutters,
  });

  markCompactForkNodes({
    component,
    outgoing: ctx.outgoing,
    leafEdgeKeys: ctx.leafEdgeKeys,
    spineEdgeKeys: ctx.spineEdgeKeys,
    compactForkNodeIds: ctx.compactForkNodeIds,
    spineSourceNodeIds: ctx.spineSourceNodeIds,
  });

  return maxRight + COMPONENT_GAP_X;
}

function emptyFactoryRunLeafLayoutResult(): FactoryRunLeafLayoutResult {
  return {
    positions: new Map(),
    sideHandleNodeIds: new Set(),
    compactForkNodeIds: new Set(),
    spineSourceNodeIds: new Set(),
    leafEdgeKeys: new Set(),
    spineEdgeKeys: new Set(),
    sideTargetNodeIds: new Set(),
    edgeRouteGutters: new Map(),
  };
}

export function layoutFactoryRunLeafGraph(
  nodes: FactoryRunLayoutNode[],
  edges: FactoryRunLayoutEdge[],
): FactoryRunLeafLayoutResult {
  if (nodes.length === 0) {
    return emptyFactoryRunLeafLayoutResult();
  }

  const positions = new Map<string, FactoryRunLayoutPosition>();
  const sideHandleNodeIds = new Set<string>();
  const compactForkNodeIds = new Set<string>();
  const spineSourceNodeIds = new Set<string>();
  const leafEdgeKeys = new Set<string>();
  const spineEdgeKeys = new Set<string>();
  const sideTargetNodeIds = new Set<string>();
  const edgeRouteGutters = new Map<string, number>();

  const nodeById = new Map(nodes.map((node) => [node.id, node]));
  const nodeIds = new Set(nodes.map((node) => node.id));
  const forwardEdges = resolveForwardEdges(edges, nodeIds);
  const { outgoing, incoming } = buildAdjacency(nodes, forwardEdges);
  const pickers = createSpinePickers(outgoing);
  const components = findWeakComponents(nodes, forwardEdges, incoming);

  const ctx: ComponentLayoutContext = {
    outgoing,
    incoming,
    forwardEdges,
    nodeById,
    pickers,
    positions,
    sideHandleNodeIds,
    compactForkNodeIds,
    spineSourceNodeIds,
    leafEdgeKeys,
    spineEdgeKeys,
    sideTargetNodeIds,
    edgeRouteGutters,
  };

  let componentOriginX = MAIN_X;
  for (const component of components) {
    componentOriginX = layoutOneComponent(component, componentOriginX, ctx);
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
