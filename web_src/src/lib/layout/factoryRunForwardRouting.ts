import type { FactoryRunLayoutEdge, FactoryRunLayoutNode, FactoryRunLayoutPosition } from "./factoryRunLeafLayout";
import {
  EDGE_ROUTE_LANE_SPACING,
  GUTTER_PAD,
  factoryRunChannelPriority,
  factoryRunLeafEdgeKey,
  nodeSize,
} from "./factoryRunLeafLayoutHelpers";

const FORWARD_TRUNK_INTERVAL_GAP = 32;
const FORWARD_TRUNK_ROUTE_Y_SPACING = 16;
const MAX_GUTTER_STEPS = 8;
const NODE_GUTTER_CLEARANCE = 8;

export type ForwardInterval = { start: number; end: number };

type PlacedForwardGutter = { gutter: number; interval: ForwardInterval };

function forwardInterval(
  edge: FactoryRunLayoutEdge,
  positions: Map<string, FactoryRunLayoutPosition>,
  nodeById: Map<string, FactoryRunLayoutNode>,
): ForwardInterval | null {
  const sourcePosition = positions.get(edge.source);
  const targetPosition = positions.get(edge.target);
  if (!sourcePosition || !targetPosition) return null;

  const sourceY = sourcePosition.y + nodeSize(nodeById.get(edge.source)!).height;
  const targetY = targetPosition.y;
  return { start: Math.min(sourceY, targetY), end: Math.max(sourceY, targetY) };
}

function forwardIntervalsOverlap(a: ForwardInterval, b: ForwardInterval): boolean {
  return a.start < b.end + FORWARD_TRUNK_INTERVAL_GAP && b.start < a.end + FORWARD_TRUNK_INTERVAL_GAP;
}

function compareForwardEdges(
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

function gutterSeeds(
  sourcePosition: FactoryRunLayoutPosition,
  targetPosition: FactoryRunLayoutPosition,
  sourceWidth: number,
  targetWidth: number,
  graphLeft: number,
): number[] {
  return [
    graphLeft - GUTTER_PAD,
    Math.min(sourcePosition.x, targetPosition.x) - GUTTER_PAD,
    Math.max(sourcePosition.x + sourceWidth, targetPosition.x + targetWidth) + GUTTER_PAD,
    sourcePosition.x - GUTTER_PAD,
    sourcePosition.x + sourceWidth + GUTTER_PAD,
    targetPosition.x - GUTTER_PAD,
    targetPosition.x + targetWidth + GUTTER_PAD,
  ];
}

function expandGutterCandidates(seeds: number[]): number[] {
  const gutters = new Set<number>();
  for (const seed of seeds) {
    for (let step = 0; step < MAX_GUTTER_STEPS; step++) {
      gutters.add(seed - step * EDGE_ROUTE_LANE_SPACING);
      gutters.add(seed + step * EDGE_ROUTE_LANE_SPACING);
    }
  }
  return [...gutters];
}

function verticalRangesOverlap(startA: number, endA: number, startB: number, endB: number): boolean {
  return startA < endB && startB < endA;
}

function gutterHitsNode(
  gutter: number,
  interval: ForwardInterval,
  edge: FactoryRunLayoutEdge,
  positions: Map<string, FactoryRunLayoutPosition>,
  nodeById: Map<string, FactoryRunLayoutNode>,
): boolean {
  for (const [id, node] of nodeById) {
    if (id === edge.source || id === edge.target) continue;
    const position = positions.get(id);
    if (!position) continue;
    const size = nodeSize(node);
    if (!verticalRangesOverlap(interval.start, interval.end, position.y, position.y + size.height)) continue;
    if (gutter > position.x - NODE_GUTTER_CLEARANCE && gutter < position.x + size.width + NODE_GUTTER_CLEARANCE) {
      return true;
    }
  }
  return false;
}

function guttersConflict(gutter: number, interval: ForwardInterval, placed: PlacedForwardGutter): boolean {
  return (
    Math.abs(gutter - placed.gutter) < EDGE_ROUTE_LANE_SPACING && forwardIntervalsOverlap(interval, placed.interval)
  );
}

function manhattanViaGutter(sourceX: number, targetX: number, gutter: number): number {
  return Math.abs(sourceX - gutter) + Math.abs(targetX - gutter);
}

function pickForwardGutter(options: {
  edge: FactoryRunLayoutEdge;
  interval: ForwardInterval;
  sourcePosition: FactoryRunLayoutPosition;
  targetPosition: FactoryRunLayoutPosition;
  sourceWidth: number;
  targetWidth: number;
  graphLeft: number;
  placed: PlacedForwardGutter[];
  positions: Map<string, FactoryRunLayoutPosition>;
  nodeById: Map<string, FactoryRunLayoutNode>;
}): number {
  const sourceCenterX = options.sourcePosition.x + options.sourceWidth / 2;
  const targetCenterX = options.targetPosition.x + options.targetWidth / 2;
  const localRight =
    Math.max(options.sourcePosition.x + options.sourceWidth, options.targetPosition.x + options.targetWidth) +
    GUTTER_PAD;
  const candidates = expandGutterCandidates(
    gutterSeeds(
      options.sourcePosition,
      options.targetPosition,
      options.sourceWidth,
      options.targetWidth,
      options.graphLeft,
    ),
  );

  let bestGutter: number | null = null;
  let bestCost = Number.POSITIVE_INFINITY;
  for (const gutter of candidates) {
    if (gutterHitsNode(gutter, options.interval, options.edge, options.positions, options.nodeById)) continue;
    if (options.placed.some((placed) => guttersConflict(gutter, options.interval, placed))) continue;
    const cost = manhattanViaGutter(sourceCenterX, targetCenterX, gutter);
    if (bestGutter == null || cost < bestCost || (cost === bestCost && gutter < bestGutter)) {
      bestGutter = gutter;
      bestCost = cost;
    }
  }
  return bestGutter ?? localRight;
}

function leftGraphLaneIndex(gutter: number, graphLeft: number): number | null {
  const lane0 = graphLeft - GUTTER_PAD;
  if (gutter > lane0) return null;
  const delta = lane0 - gutter;
  if (delta % EDGE_ROUTE_LANE_SPACING !== 0) return null;
  return delta / EDGE_ROUTE_LANE_SPACING;
}

function recordLeftGraphLane(
  lanes: ForwardInterval[][],
  gutter: number,
  interval: ForwardInterval,
  graphLeft: number,
): void {
  const laneIndex = leftGraphLaneIndex(gutter, graphLeft);
  if (laneIndex == null) return;
  while (lanes.length <= laneIndex) {
    lanes.push([]);
  }
  lanes[laneIndex].push(interval);
}

type RouteForwardTrunkEdgesOptions = {
  trunkEdges: FactoryRunLayoutEdge[];
  positions: Map<string, FactoryRunLayoutPosition>;
  nodeById: Map<string, FactoryRunLayoutNode>;
  graphLeft: number;
  edgeRouteGutters: Map<string, number>;
  edgeRouteOffsetsY: Map<string, number>;
};

export function routeForwardTrunkEdges(options: RouteForwardTrunkEdgesOptions): {
  trunkRight: number;
  leftGraphLanes: ForwardInterval[][];
} {
  const placed: PlacedForwardGutter[] = [];
  const leftGraphLanes: ForwardInterval[][] = [];
  const sortedEdges = [...options.trunkEdges].sort((a, b) => compareForwardEdges(a, b, options.positions));
  let trunkRight = 0;

  for (const edge of sortedEdges) {
    const interval = forwardInterval(edge, options.positions, options.nodeById);
    const sourcePosition = options.positions.get(edge.source);
    const targetPosition = options.positions.get(edge.target);
    if (!interval || !sourcePosition || !targetPosition) continue;
    const sourceWidth = nodeSize(options.nodeById.get(edge.source)!).width;
    const targetWidth = nodeSize(options.nodeById.get(edge.target)!).width;
    const gutter = pickForwardGutter({
      edge,
      interval,
      sourcePosition,
      targetPosition,
      sourceWidth,
      targetWidth,
      graphLeft: options.graphLeft,
      placed,
      positions: options.positions,
      nodeById: options.nodeById,
    });
    const overlappingCount = placed.filter((entry) => forwardIntervalsOverlap(entry.interval, interval)).length;
    const key = factoryRunLeafEdgeKey(edge.source, edge.target, edge.sourceHandle);
    options.edgeRouteGutters.set(key, gutter);
    options.edgeRouteOffsetsY.set(key, overlappingCount * FORWARD_TRUNK_ROUTE_Y_SPACING);
    placed.push({ gutter, interval });
    recordLeftGraphLane(leftGraphLanes, gutter, interval, options.graphLeft);
    trunkRight = Math.max(trunkRight, gutter);
  }

  return { trunkRight, leftGraphLanes };
}
