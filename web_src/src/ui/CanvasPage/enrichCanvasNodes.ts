import { Position, type Edge as ReactFlowEdge, type Node as ReactFlowNode } from "@xyflow/react";

import type { resolveCanvasFlowDirection } from "@/lib/canvasFlowDirection";
import type { FactoryRunLeafLayoutResult } from "@/lib/layout/factoryRunLeafLayout";
import { isCanvasNodeHighlighted, shouldBlankCanvasNodeBody } from "./nodeDimming";

type CanvasEdge = ReactFlowEdge;

type CanvasConnectionState = {
  nodeId: string;
  handleId: string | null;
  handleType: "source" | "target" | null;
};

type CanvasBlockNodeData = Record<string, unknown> & {
  _callbacksRef?: unknown;
  _hoveredEdge?: CanvasEdge;
  _connectingFrom?: unknown;
  _allEdges?: CanvasEdge[];
  _isHighlighted?: boolean;
  _hasHighlightedNodes?: boolean;
  _dimBodyBelowHeader?: boolean;
  _flowDirection?: ReturnType<typeof resolveCanvasFlowDirection>;
  _factorySideSource?: boolean;
  _factorySideTarget?: boolean;
  _factoryCompactFork?: boolean;
  _factorySpineSource?: boolean;
};

export type EnrichedCanvasNodeCacheEntry = {
  sourceNode: ReactFlowNode;
  sourceData: ReactFlowNode["data"];
  node: ReactFlowNode;
  data: CanvasBlockNodeData;
  hoveredEdge: CanvasEdge | null;
  connectingFrom: CanvasConnectionState | null;
  edges: CanvasEdge[];
  isHighlighted: boolean;
  hasHighlightedNodes: boolean;
  runParticipantKey: string;
  flowDirection: ReturnType<typeof resolveCanvasFlowDirection>;
  /** Ephemeral leaf layout active when this entry was built. */
  factoryLeafLayoutActive: boolean;
};

export function canReuseEnrichedNodeData({
  cachedNode,
  node,
  hoveredEdge,
  connectingFrom,
  edges,
  isHighlighted,
  hasHighlightedNodes,
  runParticipantKey,
  flowDirection,
  factoryLeafLayoutActive,
}: {
  cachedNode: EnrichedCanvasNodeCacheEntry | undefined;
  node: ReactFlowNode;
  hoveredEdge: CanvasEdge | null;
  connectingFrom: CanvasConnectionState | null;
  edges: CanvasEdge[];
  isHighlighted: boolean;
  hasHighlightedNodes: boolean;
  runParticipantKey: string;
  flowDirection: ReturnType<typeof resolveCanvasFlowDirection>;
  factoryLeafLayoutActive: boolean;
}) {
  return (
    cachedNode &&
    cachedNode.sourceData === node.data &&
    cachedNode.hoveredEdge === hoveredEdge &&
    cachedNode.connectingFrom === connectingFrom &&
    cachedNode.edges === edges &&
    cachedNode.isHighlighted === isHighlighted &&
    cachedNode.hasHighlightedNodes === hasHighlightedNodes &&
    cachedNode.runParticipantKey === runParticipantKey &&
    cachedNode.flowDirection === flowDirection &&
    cachedNode.factoryLeafLayoutActive === factoryLeafLayoutActive
  );
}

function resolveFactoryLayoutFlags(
  layout: FactoryRunLeafLayoutResult | null | undefined,
  nodeId: string,
): {
  sideSource: boolean;
  sideTarget: boolean;
  compactFork: boolean;
  spineSource: boolean;
  overlayPosition: { x: number; y: number } | undefined;
} {
  return {
    sideSource: layout?.sideHandleNodeIds.has(nodeId) ?? false,
    sideTarget: layout?.sideTargetNodeIds.has(nodeId) ?? false,
    compactFork: layout?.compactForkNodeIds.has(nodeId) ?? false,
    spineSource: layout?.spineSourceNodeIds.has(nodeId) ?? false,
    overlayPosition: layout?.positions.get(nodeId),
  };
}

function buildFactoryLayoutDataFlags(flags: {
  sideSource: boolean;
  sideTarget: boolean;
  compactFork: boolean;
  spineSource: boolean;
}): Pick<
  CanvasBlockNodeData,
  "_factorySideSource" | "_factorySideTarget" | "_factoryCompactFork" | "_factorySpineSource"
> {
  return {
    _factorySideSource: flags.sideSource,
    _factorySideTarget: flags.sideTarget,
    _factoryCompactFork: flags.compactFork,
    _factorySpineSource: flags.spineSource,
  };
}

function buildEnrichedNodeData(args: {
  canReuseData: boolean;
  cachedNode: EnrichedCanvasNodeCacheEntry | undefined;
  sourceData: CanvasBlockNodeData;
  callbacksRef: unknown;
  hoveredEdge: CanvasEdge | null;
  blockConnectingFrom: unknown;
  edges: CanvasEdge[];
  isHighlighted: boolean;
  hasHighlightedNodes: boolean;
  shouldBlankBody: boolean;
  flowDirection: ReturnType<typeof resolveCanvasFlowDirection>;
  factoryLayoutFlags: ReturnType<typeof buildFactoryLayoutDataFlags>;
}): CanvasBlockNodeData {
  if (args.canReuseData && args.cachedNode) {
    return {
      ...args.cachedNode.data,
      ...args.factoryLayoutFlags,
    };
  }
  return {
    ...args.sourceData,
    _callbacksRef: args.callbacksRef,
    _hoveredEdge: args.hoveredEdge ?? undefined,
    _connectingFrom: args.blockConnectingFrom,
    _allEdges: args.edges,
    _isHighlighted: args.isHighlighted,
    _hasHighlightedNodes: args.hasHighlightedNodes,
    _dimBodyBelowHeader: args.shouldBlankBody,
    _flowDirection: args.flowDirection,
    ...args.factoryLayoutFlags,
  };
}

function buildEnrichedReactFlowNode(args: {
  node: ReactFlowNode;
  data: CanvasBlockNodeData;
  overlayPosition: { x: number; y: number } | undefined;
  runSelectableSet: Set<string> | null | undefined;
  sideSource: boolean;
  spineSource: boolean;
  sideTarget: boolean;
  isVerticalFlow: boolean;
}): ReactFlowNode {
  return {
    ...args.node,
    position: args.overlayPosition ?? args.node.position,
    selectable: args.runSelectableSet ? args.runSelectableSet.has(args.node.id) : (args.node.selectable ?? true),
    sourcePosition:
      args.sideSource && !args.spineSource ? Position.Right : args.isVerticalFlow ? Position.Bottom : Position.Right,
    targetPosition: args.sideTarget ? Position.Left : args.isVerticalFlow ? Position.Top : Position.Left,
    data: args.data as ReactFlowNode["data"],
  };
}

export function enrichOneCanvasNode(args: {
  node: ReactFlowNode;
  cache: Map<string, EnrichedCanvasNodeCacheEntry>;
  hoveredEdge: CanvasEdge | null;
  connectingFrom: CanvasConnectionState | null;
  edges: CanvasEdge[];
  blockConnectingFrom: unknown;
  callbacksRef: unknown;
  isHighlighted: boolean;
  hasHighlightedNodes: boolean;
  shouldBlankBody: boolean;
  runParticipantKey: string;
  flowDirection: ReturnType<typeof resolveCanvasFlowDirection>;
  isVerticalFlow: boolean;
  runSelectableSet: Set<string> | null | undefined;
  factoryRunLeafLayout: FactoryRunLeafLayoutResult | null | undefined;
}): ReactFlowNode {
  const { node, cache } = args;
  const factoryLeafLayoutActive = Boolean(args.factoryRunLeafLayout);
  const cachedNode = cache.get(node.id);
  const canReuseData = Boolean(
    canReuseEnrichedNodeData({
      cachedNode,
      node,
      hoveredEdge: args.hoveredEdge,
      connectingFrom: args.connectingFrom,
      edges: args.edges,
      isHighlighted: args.isHighlighted,
      hasHighlightedNodes: args.hasHighlightedNodes,
      runParticipantKey: args.runParticipantKey,
      flowDirection: args.flowDirection,
      factoryLeafLayoutActive,
    }),
  );

  const layoutFlags = resolveFactoryLayoutFlags(args.factoryRunLeafLayout, node.id);

  // Full node reuse only when leaf layout is off (no overlay positions / side handles).
  if (canReuseData && cachedNode && cachedNode.sourceNode === node && !factoryLeafLayoutActive) {
    return cachedNode.node;
  }

  const data = buildEnrichedNodeData({
    canReuseData,
    cachedNode,
    sourceData: node.data as CanvasBlockNodeData,
    callbacksRef: args.callbacksRef,
    hoveredEdge: args.hoveredEdge,
    blockConnectingFrom: args.blockConnectingFrom,
    edges: args.edges,
    isHighlighted: args.isHighlighted,
    hasHighlightedNodes: args.hasHighlightedNodes,
    shouldBlankBody: args.shouldBlankBody,
    flowDirection: args.flowDirection,
    factoryLayoutFlags: buildFactoryLayoutDataFlags(layoutFlags),
  });

  const enrichedNode = buildEnrichedReactFlowNode({
    node,
    data,
    overlayPosition: layoutFlags.overlayPosition,
    runSelectableSet: args.runSelectableSet,
    sideSource: layoutFlags.sideSource,
    spineSource: layoutFlags.spineSource,
    sideTarget: layoutFlags.sideTarget,
    isVerticalFlow: args.isVerticalFlow,
  });

  cache.set(node.id, {
    sourceNode: node,
    sourceData: node.data,
    node: enrichedNode,
    data,
    hoveredEdge: args.hoveredEdge,
    connectingFrom: args.connectingFrom,
    edges: args.edges,
    isHighlighted: args.isHighlighted,
    hasHighlightedNodes: args.hasHighlightedNodes,
    runParticipantKey: args.runParticipantKey,
    flowDirection: args.flowDirection,
    factoryLeafLayoutActive,
  });

  return enrichedNode;
}

export function enrichCanvasNodes(args: {
  nodes: ReactFlowNode[];
  cache: Map<string, EnrichedCanvasNodeCacheEntry>;
  hoveredEdge: CanvasEdge | null;
  connectingFrom: CanvasConnectionState | null;
  edges: CanvasEdge[];
  blockConnectingFrom: unknown;
  callbacksRef: unknown;
  edgeHoverActive: boolean;
  runParticipantSet: Set<string> | null;
  runParticipantKey: string;
  flowDirection: ReturnType<typeof resolveCanvasFlowDirection>;
  isVerticalFlow: boolean;
  runSelectableSet: Set<string> | null | undefined;
  factoryRunLeafLayout: FactoryRunLeafLayoutResult | null | undefined;
}): ReactFlowNode[] {
  const runDimActive = args.runParticipantSet !== null;
  const hasHighlightedNodes = args.edgeHoverActive || runDimActive;
  const visibleNodeIds = new Set<string>();

  const enrichedNodes = args.nodes.map((node) => {
    visibleNodeIds.add(node.id);
    const isHighlighted = isCanvasNodeHighlighted({
      nodeId: node.id,
      edgeHoverActive: args.edgeHoverActive,
      highlightedNodeIds: new Set<string>(),
      runDimActive,
      runParticipantSet: args.runParticipantSet,
    });
    const shouldBlankBody = shouldBlankCanvasNodeBody({
      nodeId: node.id,
      edgeHoverActive: args.edgeHoverActive,
      runDimActive,
      runParticipantSet: args.runParticipantSet,
    });
    return enrichOneCanvasNode({
      node,
      cache: args.cache,
      hoveredEdge: args.hoveredEdge,
      connectingFrom: args.connectingFrom,
      edges: args.edges,
      blockConnectingFrom: args.blockConnectingFrom,
      callbacksRef: args.callbacksRef,
      isHighlighted,
      hasHighlightedNodes,
      shouldBlankBody,
      runParticipantKey: args.runParticipantKey,
      flowDirection: args.flowDirection,
      isVerticalFlow: args.isVerticalFlow,
      runSelectableSet: args.runSelectableSet,
      factoryRunLeafLayout: args.factoryRunLeafLayout,
    });
  });

  for (const nodeId of args.cache.keys()) {
    if (!visibleNodeIds.has(nodeId)) {
      args.cache.delete(nodeId);
    }
  }

  return enrichedNodes;
}
