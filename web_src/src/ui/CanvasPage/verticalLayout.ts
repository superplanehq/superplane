import { Position, type Edge as ReactFlowEdge, type Node as ReactFlowNode } from "@xyflow/react";
import type { CanvasesCanvas, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { DefaultLayoutEngine, type LayoutPosition } from "@/lib/layout";

type FlowNodeData = { type?: string } | undefined;

function isAnnotationNode(node: ReactFlowNode): boolean {
  return (node.data as FlowNodeData)?.type === "annotation";
}

function toComponentsNode(node: ReactFlowNode): ComponentsNode {
  return {
    id: node.id,
    // Annotations are laid out as widgets so the engine leaves them in place.
    type: isAnnotationNode(node) ? "TYPE_WIDGET" : "TYPE_ACTION",
    position: { x: node.position?.x ?? 0, y: node.position?.y ?? 0 },
  };
}

function toComponentsEdge(edge: ReactFlowEdge) {
  return {
    sourceId: edge.source,
    targetId: edge.target,
    channel: edge.sourceHandle ?? "default",
  };
}

/**
 * Builds a minimal CanvasesCanvas from the ReactFlow node/edge graph so the shared
 * ELK layout engine can compute positions without needing the full canvas spec.
 */
export function buildLayoutCanvasFromFlow(nodes: ReactFlowNode[], edges: ReactFlowEdge[]): CanvasesCanvas {
  return {
    metadata: { id: "canvas", name: "canvas" },
    spec: {
      nodes: nodes.map(toComponentsNode),
      edges: edges.map(toComponentsEdge),
    },
  };
}

/**
 * Computes the top-to-bottom (vertical) auto-layout positions for the given graph.
 * Returns a node-id -> position map; positions are anchored to the current graph's
 * top-left so the overlay stays near the user's freeform layout.
 */
export async function computeVerticalLayoutPositions(
  nodes: ReactFlowNode[],
  edges: ReactFlowEdge[],
): Promise<Map<string, LayoutPosition>> {
  if (nodes.length === 0) {
    return new Map();
  }

  const canvas = buildLayoutCanvasFromFlow(nodes, edges);
  return DefaultLayoutEngine.computeLayoutPositions(canvas, { direction: "DOWN", scope: "full-canvas" });
}

/**
 * Produces a render-only signature of the graph structure (node ids + edges).
 * Used to avoid recomputing the layout when only cosmetic/data fields change.
 */
export function getCanvasStructureSignature(nodes: ReactFlowNode[], edges: ReactFlowEdge[]): string {
  const nodeSignature = nodes
    .map((node) => node.id)
    .sort()
    .join(",");
  const edgeSignature = edges
    .map((edge) => `${edge.source}>${edge.target}:${edge.sourceHandle ?? "default"}`)
    .sort()
    .join(",");
  return `${nodeSignature}|${edgeSignature}`;
}

/**
 * Applies the vertical auto-layout overlay to nodes for rendering. Positions are
 * overridden from the engine result (falling back to the node's freeform position
 * while the layout computes), handles are re-oriented top/bottom, and nodes are made
 * non-draggable since positions are engine-controlled. The overlay never mutates the
 * saved freeform positions.
 */
export function applyVerticalOverlay<T extends ReactFlowNode>(nodes: T[], positions: Map<string, LayoutPosition>): T[] {
  return nodes.map((node) => {
    if (isAnnotationNode(node)) {
      // Annotations keep their freeform position but still adopt vertical handles.
      return {
        ...node,
        data: { ...(node.data as object), _orientation: "vertical" },
      };
    }

    const position = positions.get(node.id) ?? node.position;

    return {
      ...node,
      position,
      draggable: false,
      sourcePosition: Position.Bottom,
      targetPosition: Position.Top,
      data: { ...(node.data as object), _orientation: "vertical" },
    };
  });
}
