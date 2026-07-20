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

/**
 * Renderer-space rect (canvas coordinates) of the padded viewport for the current
 * transform. Shared by node and edge culling so both use the same visible region.
 */
export function getPaddedViewportRendererRect(
  width: number,
  height: number,
  transform: Transform,
  paddingPx = CANVAS_VIEWPORT_CULL_PADDING_PX,
): Rect {
  return getRendererRect(getPaddedViewportScreenRect(width, height, paddingPx), transform);
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

  const viewportRect = getPaddedViewportRendererRect(width, height, transform, paddingPx);
  const visibleNodeIds = new Set<string>();

  for (const node of nodeLookup.values()) {
    if (isNodeInsideRect(node, viewportRect)) {
      visibleNodeIds.add(node.id);
    }
  }

  return visibleNodeIds;
}

export function getVisibleEdgeIdsInPaddedViewport<EdgeType extends Edge = Edge, NodeType extends Node = Node>(
  edges: EdgeType[],
  visibleNodeIds: Set<string>,
  nodeLookup?: ReadonlyMap<string, InternalNode<NodeType>>,
  viewportRect?: Rect,
): Set<string> {
  const visibleEdgeIds = new Set<string>();

  for (const edge of edges) {
    if (visibleNodeIds.has(edge.source) || visibleNodeIds.has(edge.target)) {
      visibleEdgeIds.add(edge.id);
      continue;
    }

    // Both endpoints are off-screen, but the edge itself can still cross the
    // visible viewport (a long connection between two distant nodes). Keep it
    // mounted when the box spanning both endpoints overlaps the viewport so the
    // edge does not disappear while panning/zooming.
    if (nodeLookup && viewportRect && edgeSpanOverlapsRect(edge, nodeLookup, viewportRect)) {
      visibleEdgeIds.add(edge.id);
    }
  }

  return visibleEdgeIds;
}

function edgeSpanOverlapsRect<EdgeType extends Edge, NodeType extends Node>(
  edge: EdgeType,
  nodeLookup: ReadonlyMap<string, InternalNode<NodeType>>,
  viewportRect: Rect,
): boolean {
  const source = nodeLookup.get(edge.source);
  const target = nodeLookup.get(edge.target);
  if (!source || !target) {
    return false;
  }

  return getOverlappingArea(viewportRect, getEdgeSpanRect(source, target)) > 0;
}

function getEdgeSpanRect(source: InternalNode<Node>, target: InternalNode<Node>): Rect {
  const sourceRect = getNodeRect(source);
  const targetRect = getNodeRect(target);
  const x = Math.min(sourceRect.x, targetRect.x);
  const y = Math.min(sourceRect.y, targetRect.y);

  return {
    x,
    y,
    width: Math.max(sourceRect.x + sourceRect.width, targetRect.x + targetRect.width) - x,
    height: Math.max(sourceRect.y + sourceRect.height, targetRect.y + targetRect.height) - y,
  };
}

/**
 * React Flow only renders an edge while both of its endpoint nodes are mounted.
 * Any edge we keep visible therefore needs its source and target nodes mounted too,
 * even when they sit outside the culled viewport — otherwise the edge disappears as
 * soon as an endpoint scrolls off-screen (which happens sooner the more you zoom in).
 */
export function includeEndpointsOfVisibleEdges<EdgeType extends Edge>(
  visibleNodeIds: Set<string>,
  edges: EdgeType[],
  visibleEdgeIds: Set<string>,
): Set<string> {
  const nextVisibleNodeIds = new Set(visibleNodeIds);

  for (const edge of edges) {
    if (visibleEdgeIds.has(edge.id)) {
      nextVisibleNodeIds.add(edge.source);
      nextVisibleNodeIds.add(edge.target);
    }
  }

  return nextVisibleNodeIds;
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
