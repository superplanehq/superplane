import type { FactoryRunLayoutEdge, FactoryRunLayoutPosition } from "./factoryRunLeafLayout";
import {
  DEFAULT_NODE_HEIGHT,
  DEFAULT_NODE_WIDTH,
  GUTTER_PAD,
  SIDE_X_THRESHOLD,
  VERTICAL_GAP,
  factoryRunLeafEdgeKey,
} from "./factoryRunLeafLayoutHelpers";

type ResolveEdgeGutterOptions = {
  key: string;
  isSide: boolean;
  sourcePos: FactoryRunLayoutPosition;
  targetPos: FactoryRunLayoutPosition;
  sourceLayer: number;
  targetLayer: number;
  sourceCol: number;
  targetCol: number;
  graphRight: number;
  edgeRouteGutters: Map<string, number>;
};

function resolveEdgeGutter(options: ResolveEdgeGutterOptions): void {
  const {
    key,
    isSide,
    sourcePos,
    targetPos,
    sourceLayer,
    targetLayer,
    sourceCol,
    targetCol,
    graphRight,
    edgeRouteGutters,
  } = options;
  const layerSkip = targetLayer - sourceLayer > 1;
  const crossColumn = Math.abs(sourceCol - targetCol) > 0;
  if (isSide || (!layerSkip && !crossColumn)) return;

  const leftwardMerge = sourcePos.x > targetPos.x + SIDE_X_THRESHOLD;
  if (!leftwardMerge) {
    edgeRouteGutters.set(key, graphRight);
    return;
  }

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
  graphRight: number;
  leafEdgeKeys: Set<string>;
  spineEdgeKeys: Set<string>;
  sideHandleNodeIds: Set<string>;
  sideTargetNodeIds: Set<string>;
  edgeRouteGutters: Map<string, number>;
};

export function classifyComponentEdges(options: ClassifyComponentEdgesOptions): void {
  const {
    componentEdges,
    positions,
    layer,
    column,
    graphRight,
    leafEdgeKeys,
    spineEdgeKeys,
    sideHandleNodeIds,
    sideTargetNodeIds,
    edgeRouteGutters,
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
    resolveEdgeGutter({
      key,
      isSide,
      sourcePos,
      targetPos,
      graphRight,
      edgeRouteGutters,
      sourceLayer: layer.get(edge.source) ?? 0,
      targetLayer: layer.get(edge.target) ?? 0,
      sourceCol: column.get(edge.source) ?? 0,
      targetCol: column.get(edge.target) ?? 0,
    });
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
