import type { FactoryRunLayoutEdge, FactoryRunLayoutNode, FactoryRunLayoutPosition } from "./factoryRunLeafLayout";
import {
  EDGE_ROUTE_LANE_SPACING,
  factoryRunChannelPriority,
  factoryRunLeafEdgeKey,
  nodeSize,
} from "./factoryRunLeafLayoutHelpers";

const FORWARD_TRUNK_INTERVAL_GAP = 32;
const FORWARD_TRUNK_ROUTE_Y_SPACING = 16;

type ForwardInterval = { start: number; end: number };

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

type RouteForwardTrunkEdgesOptions = {
  trunkEdges: FactoryRunLayoutEdge[];
  positions: Map<string, FactoryRunLayoutPosition>;
  nodeById: Map<string, FactoryRunLayoutNode>;
  graphRight: number;
  edgeRouteGutters: Map<string, number>;
  edgeRouteOffsetsY: Map<string, number>;
};

export function routeForwardTrunkEdges(options: RouteForwardTrunkEdgesOptions): number {
  const lanes: ForwardInterval[][] = [];
  const sortedEdges = [...options.trunkEdges].sort((a, b) => compareForwardEdges(a, b, options.positions));
  let trunkRight = 0;

  for (const edge of sortedEdges) {
    const interval = forwardInterval(edge, options.positions, options.nodeById);
    if (!interval) continue;
    let laneIndex = lanes.findIndex((lane) => lane.every((placed) => !forwardIntervalsOverlap(interval, placed)));
    if (laneIndex < 0) {
      laneIndex = lanes.length;
      lanes.push([]);
    }
    lanes[laneIndex].push(interval);

    const key = factoryRunLeafEdgeKey(edge.source, edge.target, edge.sourceHandle);
    const gutter = options.graphRight + laneIndex * EDGE_ROUTE_LANE_SPACING;
    options.edgeRouteGutters.set(key, gutter);
    options.edgeRouteOffsetsY.set(key, laneIndex * FORWARD_TRUNK_ROUTE_Y_SPACING);
    trunkRight = Math.max(trunkRight, gutter);
  }

  return trunkRight;
}
