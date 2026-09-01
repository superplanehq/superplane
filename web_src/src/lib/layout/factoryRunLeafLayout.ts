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
  firstSideColumnForComponent,
  computeComponentLayers,
  computeComponentSpineColumns,
  createSpinePickers,
  findWeakComponents,
  partitionLayoutEdges,
  placeComponentNodes,
  type SpinePickers,
} from "./factoryRunLeafLayoutHelpers";
import { classifyComponentEdges, markDisplaySourceNodes } from "./factoryRunEdgeClassification";
import { routeFeedbackEdges } from "./factoryRunFeedbackRouting";

export { factoryRunLeafEdgeKey } from "./factoryRunLeafLayoutHelpers";

export const FACTORY_SIDE_HANDLE_ID = "__factorySide";
/** Centered bottom handle for spine edges when MultiBottom stems are suppressed. */
export const FACTORY_SPINE_HANDLE_ID = "__factorySpine";

export type FactoryRunLayoutNode = {
  id: string;
  width?: number;
  height?: number;
  position?: FactoryRunLayoutPosition;
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
  /** Sources that use route-based display ports instead of channel stems. */
  displaySourceNodeIds: Set<string>;
  /** Display sources that need a centered bottom handle. */
  spineSourceNodeIds: Set<string>;
  /** Edge keys that route side→left. */
  leafEdgeKeys: Set<string>;
  /** Edge keys that stay on the spine (center bottom → top when compact). */
  spineEdgeKeys: Set<string>;
  /** Targets of side edges — use Left target handle. */
  sideTargetNodeIds: Set<string>;
  /** Optional vertical-gutter X for long, cross-column, and feedback edges. */
  edgeRouteGutters: Map<string, number>;
  /** Small endpoint-turn Y stagger for feedback edges that use adjacent gutters. */
  edgeRouteOffsetsY: Map<string, number>;
  /** Edges removed from the DAG for ranking but preserved for display routing. */
  feedbackEdgeKeys: Set<string>;
};

type ComponentLayoutContext = {
  outgoing: Map<string, FactoryRunLayoutEdge[]>;
  incoming: Map<string, FactoryRunLayoutEdge[]>;
  forwardEdges: FactoryRunLayoutEdge[];
  feedbackEdges: FactoryRunLayoutEdge[];
  nodeById: Map<string, FactoryRunLayoutNode>;
  pickers: SpinePickers;
  positions: Map<string, FactoryRunLayoutPosition>;
  sideHandleNodeIds: Set<string>;
  displaySourceNodeIds: Set<string>;
  spineSourceNodeIds: Set<string>;
  leafEdgeKeys: Set<string>;
  spineEdgeKeys: Set<string>;
  sideTargetNodeIds: Set<string>;
  edgeRouteGutters: Map<string, number>;
  edgeRouteOffsetsY: Map<string, number>;
  feedbackEdgeKeys: Set<string>;
};

function savedCoordinate(value: number | undefined): number {
  return typeof value === "number" && Number.isFinite(value) ? value : Number.MAX_SAFE_INTEGER;
}

function sortRootsBySavedPosition(roots: string[], nodeById: Map<string, FactoryRunLayoutNode>): string[] {
  return [...roots].sort((a, b) => {
    const positionA = nodeById.get(a)?.position;
    const positionB = nodeById.get(b)?.position;
    const byX = savedCoordinate(positionA?.x) - savedCoordinate(positionB?.x);
    if (byX !== 0) return byX;
    const byY = savedCoordinate(positionA?.y) - savedCoordinate(positionB?.y);
    if (byY !== 0) return byY;
    return a.localeCompare(b);
  });
}

function layoutOneComponent(component: string[], componentOriginX: number, ctx: ComponentLayoutContext): number {
  const componentSet = new Set(component);
  const componentRoots = component.filter((id) =>
    (ctx.incoming.get(id) ?? []).every((e) => !componentSet.has(e.source)),
  );
  const rootsForComponent = sortRootsBySavedPosition(
    componentRoots.length > 0 ? componentRoots : [component[0]],
    ctx.nodeById,
  );

  const layer = computeComponentLayers(component, componentSet, rootsForComponent, ctx.incoming);
  const spineColumns = computeComponentSpineColumns(
    rootsForComponent,
    componentSet,
    ctx.outgoing,
    ctx.pickers.pickMainChildEdge,
  );
  const column = assignComponentColumns({
    component,
    componentSet,
    spineColumns,
    firstSideColumn: firstSideColumnForComponent(spineColumns, layer),
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
    spineColumns,
  });

  const componentForwardEdges = ctx.forwardEdges.filter(
    (edge) => componentSet.has(edge.source) && componentSet.has(edge.target),
  );
  classifyComponentEdges({
    componentEdges: componentForwardEdges,
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

  const componentFeedbackEdges = ctx.feedbackEdges.filter(
    (edge) => componentSet.has(edge.source) && componentSet.has(edge.target),
  );
  const graphLeft = Math.min(...component.map((id) => ctx.positions.get(id)?.x ?? componentOriginX));
  routeFeedbackEdges({
    feedbackEdges: componentFeedbackEdges,
    positions: ctx.positions,
    nodeById: ctx.nodeById,
    graphLeft,
    feedbackEdgeKeys: ctx.feedbackEdgeKeys,
    spineEdgeKeys: ctx.spineEdgeKeys,
    edgeRouteGutters: ctx.edgeRouteGutters,
    edgeRouteOffsetsY: ctx.edgeRouteOffsetsY,
  });

  markDisplaySourceNodes({
    componentEdges: [...componentForwardEdges, ...componentFeedbackEdges],
    leafEdgeKeys: ctx.leafEdgeKeys,
    spineEdgeKeys: ctx.spineEdgeKeys,
    displaySourceNodeIds: ctx.displaySourceNodeIds,
    spineSourceNodeIds: ctx.spineSourceNodeIds,
  });

  return maxRight + COMPONENT_GAP_X;
}

function emptyFactoryRunLeafLayoutResult(): FactoryRunLeafLayoutResult {
  return {
    positions: new Map(),
    sideHandleNodeIds: new Set(),
    displaySourceNodeIds: new Set(),
    spineSourceNodeIds: new Set(),
    leafEdgeKeys: new Set(),
    spineEdgeKeys: new Set(),
    sideTargetNodeIds: new Set(),
    edgeRouteGutters: new Map(),
    edgeRouteOffsetsY: new Map(),
    feedbackEdgeKeys: new Set(),
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
  const displaySourceNodeIds = new Set<string>();
  const spineSourceNodeIds = new Set<string>();
  const leafEdgeKeys = new Set<string>();
  const spineEdgeKeys = new Set<string>();
  const sideTargetNodeIds = new Set<string>();
  const edgeRouteGutters = new Map<string, number>();
  const edgeRouteOffsetsY = new Map<string, number>();
  const feedbackEdgeKeys = new Set<string>();

  const nodeById = new Map(nodes.map((node) => [node.id, node]));
  const nodeIds = new Set(nodes.map((node) => node.id));
  const { forwardEdges, feedbackEdges } = partitionLayoutEdges(edges, nodeIds);
  const { outgoing, incoming } = buildAdjacency(nodes, forwardEdges);
  const pickers = createSpinePickers(outgoing);
  const components = findWeakComponents(nodes, forwardEdges, incoming);

  const ctx: ComponentLayoutContext = {
    outgoing,
    incoming,
    forwardEdges,
    feedbackEdges,
    nodeById,
    pickers,
    positions,
    sideHandleNodeIds,
    displaySourceNodeIds,
    spineSourceNodeIds,
    leafEdgeKeys,
    spineEdgeKeys,
    sideTargetNodeIds,
    edgeRouteGutters,
    edgeRouteOffsetsY,
    feedbackEdgeKeys,
  };

  let componentOriginX = MAIN_X;
  for (const component of components) {
    componentOriginX = layoutOneComponent(component, componentOriginX, ctx);
  }

  return {
    positions,
    sideHandleNodeIds,
    displaySourceNodeIds,
    spineSourceNodeIds,
    leafEdgeKeys,
    spineEdgeKeys,
    sideTargetNodeIds,
    edgeRouteGutters,
    edgeRouteOffsetsY,
    feedbackEdgeKeys,
  };
}
