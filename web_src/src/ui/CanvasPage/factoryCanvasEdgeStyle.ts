import { Position, type Edge as ReactFlowEdge, type Node as ReactFlowNode } from "@xyflow/react";

import {
  factoryEdgePalette,
  factoryEdgeToneClassName,
  primaryEventStateFromCanvasNodeData,
  resolveFactoryEdgeTone,
  type FactoryEdgeTone,
} from "@/lib/factoryCanvasChrome";
import { getDraftDiffEdgeStyle } from "@/lib/draftDiff";
import {
  FACTORY_SIDE_HANDLE_ID,
  FACTORY_SPINE_HANDLE_ID,
  factoryRunLeafEdgeKey,
  type FactoryRunLeafLayoutResult,
} from "@/lib/layout/factoryRunLeafLayout";
import {
  FACTORY_TOUCHING_EDGE_CLASS,
  findTouchContrastEdgeIds,
  getCanvasEdgePath,
  type TouchEdgePathEntry,
} from "@/ui/CanvasPage/edgePath";

/** Match factoryRunLeafLayout defaults when measuring handles for touch detection. */
const FACTORY_RUN_NODE_WIDTH = 280;
const FACTORY_RUN_NODE_HEIGHT = 104;

type CanvasEdge = ReactFlowEdge;
type EdgeDefaults = {
  type: "custom";
  style: { stroke: string; strokeWidth: number };
};

function factoryRunHandleAnchor(
  position: { x: number; y: number },
  width: number,
  height: number,
  side: Position,
): { x: number; y: number } {
  switch (side) {
    case Position.Top:
      return { x: position.x + width / 2, y: position.y };
    case Position.Bottom:
      return { x: position.x + width / 2, y: position.y + height };
    case Position.Left:
      return { x: position.x, y: position.y + height / 2 };
    case Position.Right:
    default:
      return { x: position.x + width, y: position.y + height / 2 };
  }
}

function nodeBoxSize(node: ReactFlowNode | undefined): { width: number; height: number } {
  const width = node?.width && node.width > 0 ? node.width : FACTORY_RUN_NODE_WIDTH;
  const height = node?.height && node.height > 0 ? node.height : FACTORY_RUN_NODE_HEIGHT;
  return { width, height };
}

function resolveEdgeEndpointPositions(
  edge: CanvasEdge,
  nodesById: Map<string, ReactFlowNode>,
  layout: FactoryRunLeafLayoutResult,
): { sourcePos: { x: number; y: number }; targetPos: { x: number; y: number } } | null {
  const sourcePos = layout.positions.get(edge.source) ?? nodesById.get(edge.source)?.position;
  const targetPos = layout.positions.get(edge.target) ?? nodesById.get(edge.target)?.position;
  if (!sourcePos || !targetPos) {
    return null;
  }
  return { sourcePos, targetPos };
}

function buildOneFactoryEdgePathEntry(
  edge: CanvasEdge,
  nodesById: Map<string, ReactFlowNode>,
  layout: FactoryRunLeafLayoutResult,
): TouchEdgePathEntry | null {
  const edgeKey = factoryRunLeafEdgeKey(edge.source, edge.target, edge.sourceHandle);
  const leafEdge = layout.leafEdgeKeys.has(edgeKey);
  const endpoints = resolveEdgeEndpointPositions(edge, nodesById, layout);
  if (!endpoints) {
    return null;
  }

  const sourceBox = nodeBoxSize(nodesById.get(edge.source));
  const targetBox = nodeBoxSize(nodesById.get(edge.target));
  const sourceSide = leafEdge ? Position.Right : Position.Bottom;
  const targetSide = leafEdge ? Position.Left : Position.Top;
  const sourceAnchor = factoryRunHandleAnchor(endpoints.sourcePos, sourceBox.width, sourceBox.height, sourceSide);
  const targetAnchor = factoryRunHandleAnchor(endpoints.targetPos, targetBox.width, targetBox.height, targetSide);
  const routeGutterX = layout.edgeRouteGutters.get(edgeKey);
  const [path] = getCanvasEdgePath({
    sourceX: sourceAnchor.x,
    sourceY: sourceAnchor.y,
    targetX: targetAnchor.x,
    targetY: targetAnchor.y,
    sourcePosition: sourceSide,
    targetPosition: targetSide,
    ...(typeof routeGutterX === "number" ? { routeGutterX } : {}),
  });
  return { id: edge.id, path, source: edge.source, target: edge.target };
}

function collectFactoryEdgePathEntries(
  edges: CanvasEdge[],
  nodesById: Map<string, ReactFlowNode>,
  layout: FactoryRunLeafLayoutResult,
): TouchEdgePathEntry[] {
  const pathEntries: TouchEdgePathEntry[] = [];
  for (const edge of edges) {
    const entry = buildOneFactoryEdgePathEntry(edge, nodesById, layout);
    if (entry) {
      pathEntries.push(entry);
    }
  }
  return pathEntries;
}

function resolveFactoryTouchContrast(
  edges: CanvasEdge[],
  nodesById: Map<string, ReactFlowNode>,
  layout: FactoryRunLeafLayoutResult | null | undefined,
): { contrastEdgeIds: Set<string>; involvedTouchEdgeIds: Set<string> } {
  if (!layout) {
    return { contrastEdgeIds: new Set(), involvedTouchEdgeIds: new Set() };
  }
  const touchContrast = findTouchContrastEdgeIds(collectFactoryEdgePathEntries(edges, nodesById, layout));
  return { contrastEdgeIds: touchContrast.contrastIds, involvedTouchEdgeIds: touchContrast.involvedIds };
}

type FactoryEdgeToneVisual = {
  factoryTone: FactoryEdgeTone;
  factoryToneClassName: string | undefined;
  factoryToneStyle: { stroke: string; strokeWidth: number } | undefined;
  animated: boolean | undefined;
};

function resolveFactoryEdgeToneVisual(
  edge: CanvasEdge,
  nodesById: Map<string, ReactFlowNode>,
  isVerticalFlow: boolean,
  palette: ReturnType<typeof factoryEdgePalette> | null,
): FactoryEdgeToneVisual {
  let factoryTone: FactoryEdgeTone = "default";
  let factoryToneClassName: string | undefined;
  let factoryToneStyle: { stroke: string; strokeWidth: number } | undefined;
  let animated = edge.animated;

  if (isVerticalFlow && palette) {
    const targetNode = nodesById.get(edge.target);
    factoryTone = resolveFactoryEdgeTone(primaryEventStateFromCanvasNodeData(targetNode?.data));
    factoryToneClassName = factoryEdgeToneClassName(factoryTone);
    factoryToneStyle = palette[factoryTone];
    animated = factoryTone === "running";
  }

  return { factoryTone, factoryToneClassName, factoryToneStyle, animated };
}

type FactoryEdgeRouting = {
  leafEdge: boolean;
  spineEdge: boolean;
  originalChannel: string | undefined;
  routeGutterX: number | undefined;
};

function resolveFactoryEdgeRouting(
  edge: CanvasEdge,
  layout: FactoryRunLeafLayoutResult | null | undefined,
): FactoryEdgeRouting {
  const edgeKey = factoryRunLeafEdgeKey(edge.source, edge.target, edge.sourceHandle);
  const leafEdge = layout != null && layout.leafEdgeKeys.has(edgeKey);
  const spineEdge = layout != null && layout.spineEdgeKeys.has(edgeKey) && layout.compactForkNodeIds.has(edge.source);
  const originalChannel =
    typeof edge.sourceHandle === "string" && edge.sourceHandle.length > 0 && edge.sourceHandle !== "default"
      ? edge.sourceHandle
      : undefined;
  const routeGutterX = layout?.edgeRouteGutters.get(edgeKey);
  return { leafEdge, spineEdge, originalChannel, routeGutterX };
}

function resolveFactoryEdgeContrastFlags(
  edgeId: string,
  factoryTone: FactoryEdgeTone,
  layout: FactoryRunLeafLayoutResult | null | undefined,
  contrastEdgeIds: Set<string>,
  involvedTouchEdgeIds: Set<string>,
): { touchesOtherEdge: boolean; contrastStroke: boolean } {
  const statusBlocksContrast = factoryTone === "running" || factoryTone === "failed";
  const touchesOtherEdge = layout != null && involvedTouchEdgeIds.has(edgeId) && !statusBlocksContrast;
  const contrastStroke = layout != null && contrastEdgeIds.has(edgeId) && !statusBlocksContrast;
  return { touchesOtherEdge, contrastStroke };
}

type FactoryEdgeHandlePatch = {
  sourceHandle?: string | null;
  sourcePosition?: Position;
  targetPosition?: Position;
};

function factoryLeafSpineHandlePatch(routing: FactoryEdgeRouting): FactoryEdgeHandlePatch {
  if (routing.leafEdge) {
    return {
      sourceHandle: FACTORY_SIDE_HANDLE_ID,
      sourcePosition: Position.Right,
      targetPosition: Position.Left,
    };
  }
  if (routing.spineEdge) {
    return {
      sourceHandle: FACTORY_SPINE_HANDLE_ID,
      sourcePosition: Position.Bottom,
      targetPosition: Position.Top,
    };
  }
  return {};
}

function buildStyledEdgeData(args: {
  edge: CanvasEdge;
  hoveredEdgeId: string | null | undefined;
  isEditMode: boolean;
  isReadOnly: boolean;
  diffStatus: unknown;
  stableEdgeDelete: (edgeId: string) => void;
  layout: FactoryRunLeafLayoutResult | null | undefined;
  originalChannel: string | undefined;
  routeGutterX: number | undefined;
  touchesOtherEdge: boolean;
  contrastStroke: boolean;
}): CanvasEdge["data"] {
  const canMutate = args.isEditMode && !args.isReadOnly && args.diffStatus !== "removed";
  return {
    ...args.edge.data,
    isHovered: args.edge.id === args.hoveredEdgeId,
    canDelete: canMutate,
    onDelete: canMutate ? args.stableEdgeDelete : undefined,
    ...(args.layout && args.originalChannel ? { channelLabel: args.originalChannel } : {}),
    ...(typeof args.routeGutterX === "number" ? { routeGutterX: args.routeGutterX } : {}),
    ...(args.touchesOtherEdge ? { touchesOtherEdge: true } : {}),
    ...(args.contrastStroke ? { contrastStroke: true } : {}),
  };
}

function styleOneCanvasEdge(args: {
  edge: CanvasEdge;
  nodesById: Map<string, ReactFlowNode>;
  isVerticalFlow: boolean;
  palette: ReturnType<typeof factoryEdgePalette> | null;
  edgeDefaults: EdgeDefaults | { type?: string; style?: { stroke: string; strokeWidth: number } };
  hoveredEdgeId: string | null | undefined;
  isEditMode: boolean;
  isReadOnly: boolean;
  stableEdgeDelete: (edgeId: string) => void;
  layout: FactoryRunLeafLayoutResult | null | undefined;
  contrastEdgeIds: Set<string>;
  involvedTouchEdgeIds: Set<string>;
}): CanvasEdge {
  const { edge } = args;
  const diffStatus = (edge.data as Record<string, unknown> | undefined)?._draftDiffStatus;
  const diffStyle = getDraftDiffEdgeStyle(diffStatus) ?? {};
  const toneVisual = resolveFactoryEdgeToneVisual(edge, args.nodesById, args.isVerticalFlow, args.palette);
  const routing = resolveFactoryEdgeRouting(edge, args.layout);
  const contrast = resolveFactoryEdgeContrastFlags(
    edge.id,
    toneVisual.factoryTone,
    args.layout,
    args.contrastEdgeIds,
    args.involvedTouchEdgeIds,
  );
  const className =
    [edge.className, toneVisual.factoryToneClassName, contrast.contrastStroke ? FACTORY_TOUCHING_EDGE_CLASS : undefined]
      .filter(Boolean)
      .join(" ") || undefined;

  return {
    ...edge,
    ...args.edgeDefaults,
    ...factoryLeafSpineHandlePatch(routing),
    animated: toneVisual.animated,
    className,
    style: { ...args.edgeDefaults.style, ...toneVisual.factoryToneStyle, ...diffStyle },
    data: buildStyledEdgeData({
      edge,
      hoveredEdgeId: args.hoveredEdgeId,
      isEditMode: args.isEditMode,
      isReadOnly: args.isReadOnly,
      diffStatus,
      stableEdgeDelete: args.stableEdgeDelete,
      layout: args.layout,
      originalChannel: routing.originalChannel,
      routeGutterX: routing.routeGutterX,
      touchesOtherEdge: contrast.touchesOtherEdge,
      contrastStroke: contrast.contrastStroke,
    }),
    zIndex: edge.id === args.hoveredEdgeId ? 1000 : 0,
  };
}

export function buildStyledCanvasEdges(args: {
  edges: CanvasEdge[] | undefined;
  nodes: ReactFlowNode[];
  isVerticalFlow: boolean;
  resolvedThemeIsDark: boolean;
  edgeDefaults: EdgeDefaults | { type?: string; style?: { stroke: string; strokeWidth: number } };
  hoveredEdgeId: string | null | undefined;
  isEditMode: boolean;
  isReadOnly: boolean;
  stableEdgeDelete: (edgeId: string) => void;
  factoryRunLeafLayout: FactoryRunLeafLayoutResult | null | undefined;
}): CanvasEdge[] | undefined {
  const edges = args.edges ?? [];
  const nodesById = new Map(args.nodes.map((node) => [node.id, node]));
  const palette = args.isVerticalFlow ? factoryEdgePalette(args.resolvedThemeIsDark) : null;
  const { contrastEdgeIds, involvedTouchEdgeIds } = resolveFactoryTouchContrast(
    edges,
    nodesById,
    args.factoryRunLeafLayout,
  );

  return edges.map((edge) =>
    styleOneCanvasEdge({
      edge,
      nodesById,
      isVerticalFlow: args.isVerticalFlow,
      palette,
      edgeDefaults: args.edgeDefaults,
      hoveredEdgeId: args.hoveredEdgeId,
      isEditMode: args.isEditMode,
      isReadOnly: args.isReadOnly,
      stableEdgeDelete: args.stableEdgeDelete,
      layout: args.factoryRunLeafLayout,
      contrastEdgeIds,
      involvedTouchEdgeIds,
    }),
  );
}
