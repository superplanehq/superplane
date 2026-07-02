import { useStore } from "@xyflow/react";
import type { Edge, Node } from "@xyflow/react";
import { useCallback, useMemo } from "react";
import { shallow } from "zustand/shallow";
import {
  getVisibleEdgeIdsInPaddedViewport,
  getVisibleNodeIdsInPaddedViewport,
  shouldKeepCanvasNodeVisible,
} from "./canvasViewportCulling";

type CanvasViewportCullingResult = {
  visibleNodeIds: Set<string> | null;
  visibleEdgeIds: Set<string> | null;
};

export function useCanvasViewportCulling(nodes: Node[], edges: Edge[], enabled: boolean): CanvasViewportCullingResult {
  const { nodeLookup, width, height, transform } = useStore(
    useCallback(
      (state) => ({
        nodeLookup: state.nodeLookup,
        width: state.width,
        height: state.height,
        transform: state.transform,
      }),
      [],
    ),
    shallow,
  );

  return useMemo(() => {
    if (!enabled || !nodeLookup || width == null || height == null || !transform) {
      return { visibleNodeIds: null, visibleEdgeIds: null };
    }

    const visibleNodeIds = getVisibleNodeIdsInPaddedViewport(nodeLookup, width, height, transform);

    for (const node of nodes) {
      if (shouldKeepCanvasNodeVisible(node)) {
        visibleNodeIds.add(node.id);
      }
    }

    const visibleEdgeIds = getVisibleEdgeIdsInPaddedViewport(edges, visibleNodeIds);

    return { visibleNodeIds, visibleEdgeIds };
  }, [enabled, nodeLookup, width, height, transform, nodes, edges]);
}

export function applyCanvasViewportCulling<NodeType extends Node, EdgeType extends Edge>(
  nodes: NodeType[],
  edges: EdgeType[],
  visibleNodeIds: Set<string> | null,
  visibleEdgeIds: Set<string> | null,
): { nodes: NodeType[]; edges: EdgeType[] } {
  if (!visibleNodeIds) {
    return { nodes, edges };
  }

  return {
    nodes: nodes.map((node) => ({
      ...node,
      hidden: shouldKeepCanvasNodeVisible(node) ? false : !visibleNodeIds.has(node.id),
    })),
    edges: visibleEdgeIds
      ? edges.map((edge) => ({
          ...edge,
          hidden: !visibleEdgeIds.has(edge.id),
        }))
      : edges,
  };
}
