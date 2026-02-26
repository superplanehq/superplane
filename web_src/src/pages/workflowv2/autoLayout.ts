import type { CanvasesCanvas, ComponentsNode } from "@/api-client";
import ELK from "elkjs/lib/elk.bundled.js";

const elk = new ELK();

const DEFAULT_NODE_WIDTH = 420;
const DEFAULT_NODE_HEIGHT = 180;
const ANNOTATION_NODE_WIDTH = 320;
const ANNOTATION_NODE_HEIGHT = 200;

type ApplyHorizontalAutoLayoutOptions = {
  nodeIds?: string[];
};

function estimateNodeSize(node: ComponentsNode): { width: number; height: number } {
  if (node.type === "TYPE_WIDGET") {
    return {
      width: Number(node.configuration?.width) || ANNOTATION_NODE_WIDTH,
      height: Number(node.configuration?.height) || ANNOTATION_NODE_HEIGHT,
    };
  }

  return {
    width: DEFAULT_NODE_WIDTH,
    height: DEFAULT_NODE_HEIGHT,
  };
}

export async function applyHorizontalAutoLayout(
  workflow: CanvasesCanvas,
  options?: ApplyHorizontalAutoLayoutOptions,
): Promise<CanvasesCanvas> {
  const nodes = workflow.spec?.nodes || [];
  if (nodes.length === 0) {
    return workflow;
  }

  const flowNodes = nodes.filter((node) => !!node.id && node.type !== "TYPE_WIDGET");
  if (flowNodes.length === 0) {
    return workflow;
  }

  const requestedNodeIDs = options?.nodeIds || [];
  const normalizedRequestedNodeIDs = Array.from(
    new Set(requestedNodeIDs.map((nodeId) => nodeId.trim()).filter((nodeId) => nodeId.length > 0)),
  );
  const flowNodeIDs = new Set(flowNodes.map((node) => node.id as string));
  const scopedNodeIDs = normalizedRequestedNodeIDs.filter((nodeId) => flowNodeIDs.has(nodeId));
  const hasScopedSelection = scopedNodeIDs.length > 0;
  const scopedNodeIDSet = new Set(scopedNodeIDs);

  const layoutNodes = hasScopedSelection
    ? flowNodes.filter((node) => scopedNodeIDSet.has(node.id as string))
    : flowNodes;
  if (layoutNodes.length === 0) {
    return workflow;
  }

  const layoutNodeIDs = new Set(layoutNodes.map((node) => node.id as string));
  const layoutEdges = (workflow.spec?.edges || []).filter(
    (edge) =>
      !!edge.sourceId && !!edge.targetId && layoutNodeIDs.has(edge.sourceId) && layoutNodeIDs.has(edge.targetId),
  );

  const graph = {
    id: "root",
    layoutOptions: {
      "elk.algorithm": "layered",
      "elk.direction": "RIGHT",
      "elk.spacing.nodeNode": "100",
      "elk.layered.spacing.nodeNodeBetweenLayers": "180",
      "elk.layered.nodePlacement.strategy": "NETWORK_SIMPLEX",
    },
    children: layoutNodes.map((node) => {
      const { width, height } = estimateNodeSize(node);
      return {
        id: node.id!,
        width,
        height,
      };
    }),
    edges: layoutEdges.map((edge) => ({
      id: `${edge.sourceId}->${edge.targetId}->${edge.channel || "default"}`,
      sources: [edge.sourceId!],
      targets: [edge.targetId!],
    })),
  };

  const layoutedGraph = await elk.layout(graph);
  const layoutedPositions = new Map<string, { x: number; y: number }>();
  for (const child of layoutedGraph.children || []) {
    layoutedPositions.set(child.id, {
      x: child.x || 0,
      y: child.y || 0,
    });
  }

  if (layoutedPositions.size === 0) {
    return workflow;
  }

  let minCurrentX = Number.POSITIVE_INFINITY;
  let minCurrentY = Number.POSITIVE_INFINITY;
  for (const node of layoutNodes) {
    minCurrentX = Math.min(minCurrentX, node.position?.x || 0);
    minCurrentY = Math.min(minCurrentY, node.position?.y || 0);
  }
  if (!Number.isFinite(minCurrentX)) minCurrentX = 0;
  if (!Number.isFinite(minCurrentY)) minCurrentY = 0;

  let minLayoutX = Number.POSITIVE_INFINITY;
  let minLayoutY = Number.POSITIVE_INFINITY;
  layoutedPositions.forEach((position) => {
    minLayoutX = Math.min(minLayoutX, position.x);
    minLayoutY = Math.min(minLayoutY, position.y);
  });
  if (!Number.isFinite(minLayoutX)) minLayoutX = 0;
  if (!Number.isFinite(minLayoutY)) minLayoutY = 0;

  const offsetX = minCurrentX - minLayoutX;
  const offsetY = minCurrentY - minLayoutY;

  const updatedNodes = nodes.map((node) => {
    const nodeID = node.id;
    if (!nodeID) {
      return node;
    }

    const position = layoutedPositions.get(nodeID);
    if (!position) {
      return node;
    }

    return {
      ...node,
      position: {
        x: Math.round(position.x + offsetX),
        y: Math.round(position.y + offsetY),
      },
    };
  });

  return {
    ...workflow,
    spec: {
      ...workflow.spec,
      nodes: updatedNodes,
      edges: workflow.spec?.edges || [],
    },
  };
}
