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

type LayoutPosition = {
  x: number;
  y: number;
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

function resolveFlowNodes(nodes: ComponentsNode[]): ComponentsNode[] {
  return nodes.filter((node) => !!node.id && node.type !== "TYPE_WIDGET");
}

function resolveLayoutNodes(flowNodes: ComponentsNode[], requestedNodeIDs: string[]): ComponentsNode[] {
  const normalizedRequestedNodeIDs = Array.from(
    new Set(requestedNodeIDs.map((nodeId) => nodeId.trim()).filter((nodeId) => nodeId.length > 0)),
  );
  const flowNodeIDs = new Set(flowNodes.map((node) => node.id as string));
  const scopedNodeIDs = normalizedRequestedNodeIDs.filter((nodeId) => flowNodeIDs.has(nodeId));
  const hasScopedSelection = scopedNodeIDs.length > 0;
  const scopedNodeIDSet = new Set(scopedNodeIDs);

  if (hasScopedSelection) {
    return flowNodes.filter((node) => scopedNodeIDSet.has(node.id as string));
  }

  return flowNodes;
}

function resolveLayoutEdges(workflow: CanvasesCanvas, layoutNodes: ComponentsNode[]) {
  const layoutNodeIDs = new Set(layoutNodes.map((node) => node.id as string));

  return (workflow.spec?.edges || []).filter(
    (edge) =>
      !!edge.sourceId && !!edge.targetId && layoutNodeIDs.has(edge.sourceId) && layoutNodeIDs.has(edge.targetId),
  );
}

function buildElkGraph(workflow: CanvasesCanvas, layoutNodes: ComponentsNode[]) {
  const layoutEdges = resolveLayoutEdges(workflow, layoutNodes);

  return {
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
}

function extractLayoutedPositions(layoutedGraph: { children?: Array<{ id: string; x?: number; y?: number }> }) {
  const layoutedPositions = new Map<string, LayoutPosition>();
  for (const child of layoutedGraph.children || []) {
    layoutedPositions.set(child.id, {
      x: child.x || 0,
      y: child.y || 0,
    });
  }

  return layoutedPositions;
}

function resolveMinPositionFromNodes(nodes: ComponentsNode[]): LayoutPosition {
  let minX = Number.POSITIVE_INFINITY;
  let minY = Number.POSITIVE_INFINITY;

  for (const node of nodes) {
    minX = Math.min(minX, node.position?.x || 0);
    minY = Math.min(minY, node.position?.y || 0);
  }

  if (!Number.isFinite(minX)) minX = 0;
  if (!Number.isFinite(minY)) minY = 0;

  return { x: minX, y: minY };
}

function resolveMinPositionFromLayout(layoutedPositions: Map<string, LayoutPosition>): LayoutPosition {
  let minX = Number.POSITIVE_INFINITY;
  let minY = Number.POSITIVE_INFINITY;

  layoutedPositions.forEach((position) => {
    minX = Math.min(minX, position.x);
    minY = Math.min(minY, position.y);
  });

  if (!Number.isFinite(minX)) minX = 0;
  if (!Number.isFinite(minY)) minY = 0;

  return { x: minX, y: minY };
}

function applyLayoutedPositions(
  nodes: ComponentsNode[],
  layoutedPositions: Map<string, LayoutPosition>,
  offset: LayoutPosition,
): ComponentsNode[] {
  return nodes.map((node) => {
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
        x: Math.round(position.x + offset.x),
        y: Math.round(position.y + offset.y),
      },
    };
  });
}

export async function applyHorizontalAutoLayout(
  workflow: CanvasesCanvas,
  options?: ApplyHorizontalAutoLayoutOptions,
): Promise<CanvasesCanvas> {
  const nodes = workflow.spec?.nodes || [];
  if (nodes.length === 0) {
    return workflow;
  }

  const flowNodes = resolveFlowNodes(nodes);
  if (flowNodes.length === 0) {
    return workflow;
  }

  const layoutNodes = resolveLayoutNodes(flowNodes, options?.nodeIds || []);
  if (layoutNodes.length === 0) {
    return workflow;
  }

  const graph = buildElkGraph(workflow, layoutNodes);
  const layoutedGraph = await elk.layout(graph);
  const layoutedPositions = extractLayoutedPositions(layoutedGraph);

  if (layoutedPositions.size === 0) {
    return workflow;
  }

  const minCurrentPosition = resolveMinPositionFromNodes(layoutNodes);
  const minLayoutPosition = resolveMinPositionFromLayout(layoutedPositions);
  const updatedNodes = applyLayoutedPositions(nodes, layoutedPositions, {
    x: minCurrentPosition.x - minLayoutPosition.x,
    y: minCurrentPosition.y - minLayoutPosition.y,
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
