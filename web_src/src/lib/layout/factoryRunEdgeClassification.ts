import type { FactoryRunLayoutEdge, FactoryRunLayoutPosition } from "./factoryRunLeafLayout";
import {
  DEFAULT_NODE_HEIGHT,
  DEFAULT_NODE_WIDTH,
  GUTTER_PAD,
  SIDE_X_THRESHOLD,
  VERTICAL_GAP,
  factoryRunLeafEdgeKey,
} from "./factoryRunLeafLayoutHelpers";

type LongEdgeRoute = "direct" | "trunk" | "leftward";

function longEdgeRoute(options: {
  isSide: boolean;
  sourcePos: FactoryRunLayoutPosition;
  targetPos: FactoryRunLayoutPosition;
  sourceLayer: number;
  targetLayer: number;
  sourceCol: number;
  targetCol: number;
}): LongEdgeRoute {
  const layerSkip = options.targetLayer - options.sourceLayer > 1;
  const crossColumn = Math.abs(options.sourceCol - options.targetCol) > 0;
  if (options.isSide || (!layerSkip && !crossColumn)) return "direct";
  if (options.sourcePos.x > options.targetPos.x + SIDE_X_THRESHOLD) return "leftward";
  return "trunk";
}

function resolveLeftwardMergeGutter(
  key: string,
  sourcePos: FactoryRunLayoutPosition,
  targetPos: FactoryRunLayoutPosition,
  edgeRouteGutters: Map<string, number>,
): void {
  const nearby = targetPos.y - sourcePos.y < (DEFAULT_NODE_HEIGHT + VERTICAL_GAP) * 3;
  if (!nearby) {
    edgeRouteGutters.set(key, sourcePos.x + DEFAULT_NODE_WIDTH + GUTTER_PAD);
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
  edgeRouteGutters: Map<string, number>;
};

export function classifyComponentEdges(options: ClassifyComponentEdgesOptions): FactoryRunLayoutEdge[] {
  const {
    componentEdges,
    positions,
    layer,
    column,
    leafEdgeKeys,
    spineEdgeKeys,
    sideHandleNodeIds,
    sideTargetNodeIds,
    edgeRouteGutters,
  } = options;
  const trunkEdges: FactoryRunLayoutEdge[] = [];
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
    const route = longEdgeRoute({
      isSide,
      sourcePos,
      targetPos,
      sourceLayer: layer.get(edge.source) ?? 0,
      targetLayer: layer.get(edge.target) ?? 0,
      sourceCol: column.get(edge.source) ?? 0,
      targetCol: column.get(edge.target) ?? 0,
    });
    if (route === "trunk") {
      trunkEdges.push(edge);
    } else if (route === "leftward") {
      resolveLeftwardMergeGutter(key, sourcePos, targetPos, edgeRouteGutters);
    }
  }
  return trunkEdges;
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
