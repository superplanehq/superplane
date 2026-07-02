import type { Edge, InternalNode, Node, Rect, Transform } from "@xyflow/react";

/**
 * Screen-space padding around the viewport when culling off-screen nodes.
 * Roughly one component node width so edge-adjacent nodes stay mounted while panning.
 */
export const CANVAS_VIEWPORT_CULL_PADDING_PX = 320;

export function getPaddedViewportScreenRect(
  width: number,
  height: number,
  paddingPx = CANVAS_VIEWPORT_CULL_PADDING_PX,
): { x: number; y: number; width: number; height: number } {
  return {
    x: -paddingPx,
    y: -paddingPx,
    width: width + paddingPx * 2,
    height: height + paddingPx * 2,
  };
}

export function getVisibleNodeIdsInPaddedViewport<NodeType extends Node = Node>(
  nodeLookup: ReadonlyMap<string, InternalNode<NodeType>>,
  width: number,
  height: number,
  transform: Transform,
  paddingPx = CANVAS_VIEWPORT_CULL_PADDING_PX,
): Set<string> {
  if (width === 0 || height === 0) {
    return new Set(nodeLookup.keys());
  }

  const viewportRect = getRendererRect(getPaddedViewportScreenRect(width, height, paddingPx), transform);
  const visibleNodeIds = new Set<string>();

  for (const node of nodeLookup.values()) {
    if (isNodeInsideRect(node, viewportRect)) {
      visibleNodeIds.add(node.id);
    }
  }

  return visibleNodeIds;
}

export function getVisibleEdgeIdsInPaddedViewport<EdgeType extends Edge = Edge>(
  edges: EdgeType[],
  visibleNodeIds: Set<string>,
): Set<string> {
  const visibleEdgeIds = new Set<string>();

  for (const edge of edges) {
    if (visibleNodeIds.has(edge.source) || visibleNodeIds.has(edge.target)) {
      visibleEdgeIds.add(edge.id);
    }
  }

  return visibleEdgeIds;
}

export function includeCanvasNodesThatMustStayMounted<NodeType extends Node>(
  visibleNodeIds: Set<string>,
  nodeLookup: ReadonlyMap<string, InternalNode<NodeType>>,
  nodes: NodeType[],
): Set<string> {
  const nextVisibleNodeIds = new Set(visibleNodeIds);

  for (const node of nodes) {
    if (!nodeLookup.has(node.id) || shouldKeepCanvasNodeVisible(node)) {
      nextVisibleNodeIds.add(node.id);
    }
  }

  return nextVisibleNodeIds;
}

export function shouldKeepCanvasNodeVisible(node: {
  id: string;
  dragging?: boolean;
  selected?: boolean;
  data?: { isTemplate?: boolean; isPendingConnection?: boolean };
}): boolean {
  if (node.dragging || node.selected) {
    return true;
  }

  const data = node.data;
  return Boolean(data?.isTemplate || data?.isPendingConnection);
}

function getRendererRect(screenRect: Rect, [tx, ty, scale]: Transform): Rect {
  return {
    x: (screenRect.x - tx) / scale,
    y: (screenRect.y - ty) / scale,
    width: screenRect.width / scale,
    height: screenRect.height / scale,
  };
}

function isNodeInsideRect(node: InternalNode<Node>, rect: Rect): boolean {
  if (!node.internals.handleBounds || node.dragging) {
    return true;
  }

  const nodeRect = getNodeRect(node);
  if (nodeRect.width <= 0 || nodeRect.height <= 0) {
    return true;
  }

  return getOverlappingArea(rect, nodeRect) > 0;
}

function getNodeRect(node: InternalNode<Node>): Rect {
  return {
    x: node.internals.positionAbsolute.x,
    y: node.internals.positionAbsolute.y,
    width: node.measured.width ?? node.width ?? node.initialWidth ?? 0,
    height: node.measured.height ?? node.height ?? node.initialHeight ?? 0,
  };
}

function getOverlappingArea(rectA: Rect, rectB: Rect): number {
  const xOverlap = Math.max(0, Math.min(rectA.x + rectA.width, rectB.x + rectB.width) - Math.max(rectA.x, rectB.x));
  const yOverlap = Math.max(0, Math.min(rectA.y + rectA.height, rectB.y + rectB.height) - Math.max(rectA.y, rectB.y));

  return Math.ceil(xOverlap * yOverlap);
}
