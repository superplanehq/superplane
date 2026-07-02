import { getNodesInside, type EdgeBase, type InternalNodeBase, type NodeBase, type Transform } from "@xyflow/system";

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

export function getVisibleNodeIdsInPaddedViewport<NodeType extends NodeBase = NodeBase>(
  nodeLookup: Map<string, InternalNodeBase<NodeType>>,
  width: number,
  height: number,
  transform: Transform,
  paddingPx = CANVAS_VIEWPORT_CULL_PADDING_PX,
): Set<string> {
  if (width === 0 || height === 0) {
    return new Set(nodeLookup.keys());
  }

  const visibleNodes = getNodesInside(
    nodeLookup,
    getPaddedViewportScreenRect(width, height, paddingPx),
    transform,
    true,
  );

  return new Set(visibleNodes.map((node) => node.id));
}

export function getVisibleEdgeIdsInPaddedViewport<EdgeType extends EdgeBase = EdgeBase>(
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
