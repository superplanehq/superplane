import type { BlueprintsBlueprint, CanvasesCanvas, ComponentsComponent, ComponentsNode } from "@/api-client";
import ELK from "elkjs/lib/elk.bundled.js";

const elk = new ELK();

const DEFAULT_NODE_WIDTH = 420;
const DEFAULT_NODE_HEIGHT = 180;
const ANNOTATION_NODE_WIDTH = 320;
const ANNOTATION_NODE_HEIGHT = 200;

type ApplyHorizontalAutoLayoutOptions = {
  nodeIds?: string[];
  scope?: "full-canvas" | "connected-component";
  channelsByNodeId?: Map<string, string[]>;
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

function normalizeRequestedNodeIDs(flowNodes: ComponentsNode[], requestedNodeIDs: string[]): string[] {
  const normalizedRequestedNodeIDs = Array.from(
    new Set(requestedNodeIDs.map((nodeId) => nodeId.trim()).filter((nodeId) => nodeId.length > 0)),
  );

  const flowNodeIDs = new Set(flowNodes.map((node) => node.id as string));
  return normalizedRequestedNodeIDs.filter((nodeID) => flowNodeIDs.has(nodeID));
}

function resolveLayoutScope(options: ApplyHorizontalAutoLayoutOptions | undefined, hasSeedNodeIDs: boolean) {
  if (options?.scope) {
    return options.scope;
  }

  return hasSeedNodeIDs ? "connected-component" : "full-canvas";
}

function buildFlowAdjacency(workflow: CanvasesCanvas, flowNodes: ComponentsNode[]) {
  const flowNodeIDSet = new Set(flowNodes.map((node) => node.id as string));
  const adjacencyByNodeID = new Map<string, string[]>();

  for (const node of flowNodes) {
    adjacencyByNodeID.set(node.id as string, []);
  }

  for (const edge of workflow.spec?.edges || []) {
    const sourceID = edge.sourceId;
    const targetID = edge.targetId;
    if (!sourceID || !targetID) {
      continue;
    }

    if (!flowNodeIDSet.has(sourceID) || !flowNodeIDSet.has(targetID)) {
      continue;
    }

    adjacencyByNodeID.get(sourceID)?.push(targetID);
    adjacencyByNodeID.get(targetID)?.push(sourceID);
  }

  return adjacencyByNodeID;
}

function resolveConnectedComponentNodeIDs(
  workflow: CanvasesCanvas,
  flowNodes: ComponentsNode[],
  seedNodeIDs: string[],
): string[] {
  if (seedNodeIDs.length === 0) {
    return flowNodes.map((node) => node.id as string);
  }

  const adjacencyByNodeID = buildFlowAdjacency(workflow, flowNodes);
  const visitedNodeIDs = new Set<string>();
  const queue = [...seedNodeIDs];

  while (queue.length > 0) {
    const currentNodeID = queue.shift();
    if (!currentNodeID || visitedNodeIDs.has(currentNodeID)) {
      continue;
    }

    visitedNodeIDs.add(currentNodeID);
    const neighbors = adjacencyByNodeID.get(currentNodeID) || [];
    for (const neighborNodeID of neighbors) {
      if (!visitedNodeIDs.has(neighborNodeID)) {
        queue.push(neighborNodeID);
      }
    }
  }

  return flowNodes.map((node) => node.id as string).filter((nodeID) => visitedNodeIDs.has(nodeID));
}

function resolveScopedNodeIDs(
  workflow: CanvasesCanvas,
  flowNodes: ComponentsNode[],
  options?: ApplyHorizontalAutoLayoutOptions,
): string[] {
  const seedNodeIDs = normalizeRequestedNodeIDs(flowNodes, options?.nodeIds || []);
  const scope = resolveLayoutScope(options, seedNodeIDs.length > 0);

  if (scope === "connected-component") {
    return resolveConnectedComponentNodeIDs(workflow, flowNodes, seedNodeIDs);
  }

  return flowNodes.map((node) => node.id as string);
}

function resolveLayoutNodes(flowNodes: ComponentsNode[], scopedNodeIDs: string[]): ComponentsNode[] {
  if (scopedNodeIDs.length === 0) {
    return [];
  }

  const scopedNodeIDSet = new Set(scopedNodeIDs);
  return flowNodes.filter((node) => scopedNodeIDSet.has(node.id as string));
}

function resolveLayoutEdges(workflow: CanvasesCanvas, layoutNodes: ComponentsNode[]) {
  const layoutNodeIDs = new Set(layoutNodes.map((node) => node.id as string));

  return (workflow.spec?.edges || []).filter(
    (edge) =>
      !!edge.sourceId && !!edge.targetId && layoutNodeIDs.has(edge.sourceId) && layoutNodeIDs.has(edge.targetId),
  );
}

function buildElkGraph(
  workflow: CanvasesCanvas,
  layoutNodes: ComponentsNode[],
  channelsByNodeId?: Map<string, string[]>,
) {
  const layoutEdges = resolveLayoutEdges(workflow, layoutNodes);

  return {
    id: "root",
    layoutOptions: {
      "elk.algorithm": "layered",
      "elk.direction": "RIGHT",
      "elk.spacing.nodeNode": "100",
      "elk.layered.spacing.nodeNodeBetweenLayers": "180",
      "elk.layered.nodePlacement.strategy": "NETWORK_SIMPLEX",
      "elk.contentAlignment": "V_CENTER",
    },
    children: layoutNodes.map((node) => {
      const { width, height } = estimateNodeSize(node);
      const nodeId = node.id!;
      const outputChannels = channelsByNodeId?.get(nodeId) || ["default"];

      const ports = [
        {
          id: `${nodeId}__input`,
          properties: {
            "elk.port.side": "WEST",
          },
        },
        ...outputChannels.map((channel, index) => ({
          id: `${nodeId}__${channel}`,
          properties: {
            "elk.port.side": "EAST",
            "elk.port.index": `${index}`,
          },
        })),
      ];

      return {
        id: nodeId,
        width,
        height,
        properties: {
          "elk.portConstraints": "FIXED_ORDER",
        },
        ports,
      };
    }),
    edges: layoutEdges.map((edge) => ({
      id: `${edge.sourceId}->${edge.targetId}->${edge.channel || "default"}`,
      sources: [`${edge.sourceId}__${edge.channel || "default"}`],
      targets: [`${edge.targetId}__input`],
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

export function buildChannelsByNodeId(
  workflow: CanvasesCanvas,
  components: ComponentsComponent[],
  blueprints: BlueprintsBlueprint[],
): Map<string, string[]> {
  const map = new Map<string, string[]>();

  for (const node of workflow.spec?.nodes || []) {
    if (!node.id) continue;

    let channels: string[] = ["default"];

    if (node.type === "TYPE_TRIGGER") {
      channels = ["default"];
    } else if (node.type === "TYPE_BLUEPRINT") {
      const componentMeta = components.find((c) => c.name === node.component?.name);
      const bp = blueprints.find((b) => b.id === node.blueprint?.id);
      channels = componentMeta?.outputChannels?.map((c) => c.name!).filter(Boolean) ||
        bp?.outputChannels?.map((c) => c.name!).filter(Boolean) || ["default"];
    } else if (node.type === "TYPE_COMPONENT" && node.component?.name) {
      const meta = components.find((c) => c.name === node.component?.name);
      channels = meta?.outputChannels?.map((c) => c.name!).filter(Boolean) || ["default"];
    }

    map.set(node.id, channels);
  }

  return map;
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

  const scopedNodeIDs = resolveScopedNodeIDs(workflow, flowNodes, options);
  const layoutNodes = resolveLayoutNodes(flowNodes, scopedNodeIDs);
  if (layoutNodes.length === 0) {
    return workflow;
  }

  const graph = buildElkGraph(workflow, layoutNodes, options?.channelsByNodeId);
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
