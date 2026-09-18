import type { FactoryRunLayoutEdge, FactoryRunLayoutNode, FactoryRunLayoutPosition } from "./factoryRunLeafLayout";
import {
  DEFAULT_NODE_HEIGHT,
  FORWARD_GUTTER_LANE_SPACING,
  SIDE_X_THRESHOLD,
  VERTICAL_GAP,
  factoryRunChannelPriority,
  factoryRunLeafEdgeKey,
  nodeSize,
} from "./factoryRunLeafLayoutHelpers";

const FORWARD_GUTTER_INTERVAL_GAP = 32;
const FORWARD_GUTTER_ROUTE_Y_SPACING = 16;

type GutterInterval = { start: number; end: number };

type ResolveForwardGutterCandidateOptions = {
  isSide: boolean;
  sourcePos: FactoryRunLayoutPosition;
  targetPos: FactoryRunLayoutPosition;
  sourceLayer: number;
  targetLayer: number;
  sourceCol: number;
  targetCol: number;
};

function isForwardGutterCandidate(options: ResolveForwardGutterCandidateOptions): boolean {
  const { isSide, sourcePos, targetPos, sourceLayer, targetLayer, sourceCol, targetCol } = options;
  const layerSkip = targetLayer - sourceLayer > 1;
  const crossColumn = Math.abs(sourceCol - targetCol) > 0;
  if (isSide || (!layerSkip && !crossColumn)) return false;

  const leftwardMerge = sourcePos.x > targetPos.x + SIDE_X_THRESHOLD;
  if (!leftwardMerge) return true;

  const nearby = targetPos.y - sourcePos.y < (DEFAULT_NODE_HEIGHT + VERTICAL_GAP) * 3;
  return !nearby;
}

function forwardGutterInterval(
  edge: FactoryRunLayoutEdge,
  positions: Map<string, FactoryRunLayoutPosition>,
  nodeById: Map<string, FactoryRunLayoutNode>,
): GutterInterval | null {
  const sourcePosition = positions.get(edge.source);
  const targetPosition = positions.get(edge.target);
  const sourceNode = nodeById.get(edge.source);
  if (!sourcePosition || !targetPosition || !sourceNode) return null;

  const sourceY = sourcePosition.y + nodeSize(sourceNode).height;
  const targetY = targetPosition.y;
  return { start: Math.min(sourceY, targetY), end: Math.max(sourceY, targetY) };
}

function forwardGutterIntervalsOverlap(a: GutterInterval, b: GutterInterval): boolean {
  return a.start < b.end + FORWARD_GUTTER_INTERVAL_GAP && b.start < a.end + FORWARD_GUTTER_INTERVAL_GAP;
}

function compareForwardGutterEdges(
  a: FactoryRunLayoutEdge,
  b: FactoryRunLayoutEdge,
  positions: Map<string, FactoryRunLayoutPosition>,
): number {
  const sourceA = positions.get(a.source);
  const sourceB = positions.get(b.source);
  const bySourceX = (sourceA?.x ?? 0) - (sourceB?.x ?? 0);
  if (bySourceX !== 0) return bySourceX;
  const bySourceY = (sourceA?.y ?? 0) - (sourceB?.y ?? 0);
  if (bySourceY !== 0) return bySourceY;
  const byChannel = factoryRunChannelPriority(a.sourceHandle) - factoryRunChannelPriority(b.sourceHandle);
  if (byChannel !== 0) return byChannel;
  return (a.id ?? factoryRunLeafEdgeKey(a.source, a.target, a.sourceHandle)).localeCompare(
    b.id ?? factoryRunLeafEdgeKey(b.source, b.target, b.sourceHandle),
  );
}

type RouteForwardGutterEdgesOptions = {
  edges: FactoryRunLayoutEdge[];
  positions: Map<string, FactoryRunLayoutPosition>;
  nodeById: Map<string, FactoryRunLayoutNode>;
  graphRight: number;
  edgeRouteGutters: Map<string, number>;
  edgeRouteOffsetsY: Map<string, number>;
};

export function routeForwardGutterEdges(options: RouteForwardGutterEdgesOptions): void {
  const lanes: GutterInterval[][] = [];
  const sortedEdges = [...options.edges].sort((a, b) => compareForwardGutterEdges(a, b, options.positions));

  for (const edge of sortedEdges) {
    const interval = forwardGutterInterval(edge, options.positions, options.nodeById);
    if (!interval) continue;
    let laneIndex = lanes.findIndex((lane) => lane.every((placed) => !forwardGutterIntervalsOverlap(interval, placed)));
    if (laneIndex < 0) {
      laneIndex = lanes.length;
      lanes.push([]);
    }
    lanes[laneIndex].push(interval);

    const key = factoryRunLeafEdgeKey(edge.source, edge.target, edge.sourceHandle);
    options.edgeRouteGutters.set(key, options.graphRight + laneIndex * FORWARD_GUTTER_LANE_SPACING);
    options.edgeRouteOffsetsY.set(key, laneIndex * FORWARD_GUTTER_ROUTE_Y_SPACING);
  }
}

type ClassifyComponentEdgesOptions = {
  componentEdges: FactoryRunLayoutEdge[];
  positions: Map<string, FactoryRunLayoutPosition>;
  layer: Map<string, number>;
  column: Map<string, number>;
  leafEdgeKeys: Set<string>;
  spineEdgeKeys: Set<string>;
  sideHandleNodeIds: Set<string>;
  sideTargetNodeIds: Set<string>;
  forwardGutterEdges: FactoryRunLayoutEdge[];
};

export function classifyComponentEdges(options: ClassifyComponentEdgesOptions): void {
  const {
    componentEdges,
    positions,
    layer,
    column,
    leafEdgeKeys,
    spineEdgeKeys,
    sideHandleNodeIds,
    sideTargetNodeIds,
    forwardGutterEdges,
  } = options;
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
    if (
      isForwardGutterCandidate({
        isSide,
        sourcePos,
        targetPos,
        sourceLayer: layer.get(edge.source) ?? 0,
        targetLayer: layer.get(edge.target) ?? 0,
        sourceCol: column.get(edge.source) ?? 0,
        targetCol: column.get(edge.target) ?? 0,
      })
    ) {
      forwardGutterEdges.push(edge);
    }
  }
}

type MarkDisplaySourceNodesOptions = {
  componentEdges: FactoryRunLayoutEdge[];
  leafEdgeKeys: Set<string>;
  spineEdgeKeys: Set<string>;
  displaySourceNodeIds: Set<string>;
  spineSourceNodeIds: Set<string>;
};

export function markDisplaySourceNodes(options: MarkDisplaySourceNodesOptions): void {
  const { componentEdges, leafEdgeKeys, spineEdgeKeys, displaySourceNodeIds, spineSourceNodeIds } = options;
  for (const edge of componentEdges) {
    const key = factoryRunLeafEdgeKey(edge.source, edge.target, edge.sourceHandle);
    if (!leafEdgeKeys.has(key) && !spineEdgeKeys.has(key)) continue;
    displaySourceNodeIds.add(edge.source);
    if (spineEdgeKeys.has(key)) {
      spineSourceNodeIds.add(edge.source);
    }
  }
}
