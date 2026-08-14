import type { FactoryRunLayoutEdge, FactoryRunLayoutNode, FactoryRunLayoutPosition } from "./factoryRunLeafLayout";
import { GUTTER_PAD, factoryRunChannelPriority, factoryRunLeafEdgeKey, nodeSize } from "./factoryRunLeafLayoutHelpers";

const FEEDBACK_LANE_SPACING = 48;
const FEEDBACK_INTERVAL_GAP = 32;
const FEEDBACK_ROUTE_Y_SPACING = 16;

type FeedbackInterval = { start: number; end: number };

function feedbackInterval(
  edge: FactoryRunLayoutEdge,
  positions: Map<string, FactoryRunLayoutPosition>,
  nodeById: Map<string, FactoryRunLayoutNode>,
): FeedbackInterval | null {
  const sourcePosition = positions.get(edge.source);
  const targetPosition = positions.get(edge.target);
  if (!sourcePosition || !targetPosition) return null;

  const sourceY = sourcePosition.y + nodeSize(nodeById.get(edge.source)!).height;
  const targetY = targetPosition.y;
  return { start: Math.min(sourceY, targetY), end: Math.max(sourceY, targetY) };
}

function feedbackIntervalsOverlap(a: FeedbackInterval, b: FeedbackInterval): boolean {
  return a.start < b.end + FEEDBACK_INTERVAL_GAP && b.start < a.end + FEEDBACK_INTERVAL_GAP;
}

function compareFeedbackEdges(
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

type RouteFeedbackEdgesOptions = {
  feedbackEdges: FactoryRunLayoutEdge[];
  positions: Map<string, FactoryRunLayoutPosition>;
  nodeById: Map<string, FactoryRunLayoutNode>;
  graphLeft: number;
  feedbackEdgeKeys: Set<string>;
  spineEdgeKeys: Set<string>;
  edgeRouteGutters: Map<string, number>;
  edgeRouteOffsetsY: Map<string, number>;
};

export function routeFeedbackEdges(options: RouteFeedbackEdgesOptions): void {
  const lanes: FeedbackInterval[][] = [];
  const sortedEdges = [...options.feedbackEdges].sort((a, b) => compareFeedbackEdges(a, b, options.positions));

  for (const edge of sortedEdges) {
    const interval = feedbackInterval(edge, options.positions, options.nodeById);
    if (!interval) continue;
    let laneIndex = lanes.findIndex((lane) => lane.every((placed) => !feedbackIntervalsOverlap(interval, placed)));
    if (laneIndex < 0) {
      laneIndex = lanes.length;
      lanes.push([]);
    }
    lanes[laneIndex].push(interval);

    const key = factoryRunLeafEdgeKey(edge.source, edge.target, edge.sourceHandle);
    options.feedbackEdgeKeys.add(key);
    options.spineEdgeKeys.add(key);
    options.edgeRouteGutters.set(key, options.graphLeft - GUTTER_PAD - laneIndex * FEEDBACK_LANE_SPACING);
    options.edgeRouteOffsetsY.set(key, laneIndex * FEEDBACK_ROUTE_Y_SPACING);
  }
}
