import type { CanvasesCanvas, ActionsAction, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import type { CanvasFlowDirection } from "@/lib/canvasFlowDirection";
import { agentRunnerStepTitles, AGENT_HARNESS_COMPONENTS } from "@/lib/agentRunnerSteps";
import { FACTORY_NODE_VERTICAL_GAP, factoryNodeCardSize } from "@/lib/factoryCanvasChrome";
import ELK from "elkjs/lib/elk.bundled.js";
import type { LayoutEngine, LayoutEngineApplyOptions } from "./types";
import { appendUniqueChannels, resolveForwardLayoutEdges } from "./layoutGraph";
import {
  applyLayoutedPositions,
  packComponentPositions,
  resolveLayoutBounds,
  resolveMinPositionFromLayout,
  resolveMinPositionFromNodes,
  sortComponentsByCurrentPosition,
  type LayoutPosition,
} from "./elkPacking";

const DEFAULT_NODE_WIDTH = 420;
const DEFAULT_NODE_HEIGHT = 180;
const ANNOTATION_NODE_WIDTH = 320;
const ANNOTATION_NODE_HEIGHT = 200;
const DISCONNECTED_COMPONENT_VERTICAL_GAP = 220;
/** Side-by-side packing gap for vertical (factory) canvases — tighter than horizontal. */
const DISCONNECTED_COMPONENT_HORIZONTAL_GAP_VERTICAL = 100;
/** Same-layer horizontal gap when flow is top→bottom. */
const VERTICAL_FLOW_NODE_NODE_SPACING = "48";
const HORIZONTAL_FLOW_NODE_NODE_SPACING = "100";
const NODE_NODE_BETWEEN_LAYERS = "180";
const VERTICAL_FLOW_NODE_NODE_BETWEEN_LAYERS = String(FACTORY_NODE_VERTICAL_GAP);

export class ElkLayoutEngine implements LayoutEngine {
  private readonly elk = new ELK();

  estimateNodeSize(
    node: ComponentsNode,
    direction: CanvasFlowDirection = "horizontal",
  ): { width: number; height: number } {
    if (node.type === "TYPE_WIDGET") {
      return {
        width: Number(node.configuration?.width) || ANNOTATION_NODE_WIDTH,
        height: Number(node.configuration?.height) || ANNOTATION_NODE_HEIGHT,
      };
    }

    if (direction === "vertical") {
      const stepCount = AGENT_HARNESS_COMPONENTS.has(node.component ?? "")
        ? agentRunnerStepTitles(node.configuration).length
        : 0;
      return factoryNodeCardSize(stepCount);
    }

    return {
      width: DEFAULT_NODE_WIDTH,
      height: DEFAULT_NODE_HEIGHT,
    };
  }

  async apply(workflow: CanvasesCanvas, options?: LayoutEngineApplyOptions): Promise<CanvasesCanvas> {
    const nodes = workflow.spec?.nodes || [];
    if (nodes.length === 0) {
      return workflow;
    }

    const flowNodes = this.resolveFlowNodes(nodes);
    if (flowNodes.length === 0) {
      return workflow;
    }

    const scopedNodeIDs = this.resolveScopedNodeIDs(workflow, flowNodes, options);
    const layoutNodes = this.resolveLayoutNodes(flowNodes, scopedNodeIDs);
    if (layoutNodes.length === 0) {
      return workflow;
    }

    const outputChannelsByNodeId = this.buildOutputChannelsByNodeId(workflow, options?.components || []);
    const direction = options?.direction ?? "horizontal";
    const layoutedPositions = await this.resolvePackedLayoutedPositions(
      workflow,
      layoutNodes,
      outputChannelsByNodeId,
      direction,
    );

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

  private normalizeChannel(channel?: string): string {
    const normalizedChannel = (channel || "").trim();
    return normalizedChannel.length > 0 ? normalizedChannel : "default";
  }

  private isAnnotationWidget(node: ComponentsNode): boolean {
    return node.type === "TYPE_WIDGET";
  }

  private resolveFlowNodes(nodes: ComponentsNode[]): ComponentsNode[] {
    return nodes.filter((node) => !!node.id && !this.isAnnotationWidget(node));
  }

  private normalizeRequestedNodeIDs(flowNodes: ComponentsNode[], requestedNodeIDs: string[]): string[] {
    const normalizedRequestedNodeIDs = Array.from(
      new Set(requestedNodeIDs.map((nodeId) => nodeId.trim()).filter((nodeId) => nodeId.length > 0)),
    );

    const flowNodeIDs = new Set(flowNodes.map((node) => node.id as string));
    return normalizedRequestedNodeIDs.filter((nodeID) => flowNodeIDs.has(nodeID));
  }

  private resolveLayoutScope(options: LayoutEngineApplyOptions | undefined, hasSeedNodeIDs: boolean) {
    if (options?.scope) {
      return options.scope;
    }

    return hasSeedNodeIDs ? "connected-component" : "full-canvas";
  }

  private buildFlowAdjacency(workflow: CanvasesCanvas, flowNodes: ComponentsNode[]) {
    const flowNodeIDSet = new Set(flowNodes.map((node) => node.id as string));
    const adjacencyByNodeID = new Map<string, string[]>();

    for (const node of flowNodes) {
      adjacencyByNodeID.set(node.id as string, []);
    }

    for (const edge of workflow.spec?.edges || []) {
      const sourceID = edge.sourceId;
      const targetID = edge.targetId;
      if (!sourceID || !targetID || sourceID === targetID) {
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

  private resolveConnectedComponentNodeIDs(
    workflow: CanvasesCanvas,
    flowNodes: ComponentsNode[],
    seedNodeIDs: string[],
  ): string[] {
    if (seedNodeIDs.length === 0) {
      return flowNodes.map((node) => node.id as string);
    }

    const adjacencyByNodeID = this.buildFlowAdjacency(workflow, flowNodes);
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

  private resolveScopedNodeIDs(
    workflow: CanvasesCanvas,
    flowNodes: ComponentsNode[],
    options: LayoutEngineApplyOptions | undefined,
  ): string[] {
    const seedNodeIDs = this.normalizeRequestedNodeIDs(flowNodes, options?.nodeIds || []);
    const scope = this.resolveLayoutScope(options, seedNodeIDs.length > 0);

    if (scope === "connected-component") {
      return this.resolveConnectedComponentNodeIDs(workflow, flowNodes, seedNodeIDs);
    }

    return flowNodes.map((node) => node.id as string);
  }

  private resolveLayoutNodes(flowNodes: ComponentsNode[], scopedNodeIDs: string[]): ComponentsNode[] {
    if (scopedNodeIDs.length === 0) {
      return [];
    }

    const scopedNodeIDSet = new Set(scopedNodeIDs);
    return flowNodes.filter((node) => scopedNodeIDSet.has(node.id as string));
  }

  private resolveLayoutEdges(workflow: CanvasesCanvas, layoutNodes: ComponentsNode[]) {
    const layoutNodeIDs = new Set(layoutNodes.map((node) => node.id as string));

    return (workflow.spec?.edges || []).filter(
      (edge) =>
        !!edge.sourceId &&
        !!edge.targetId &&
        edge.sourceId !== edge.targetId &&
        layoutNodeIDs.has(edge.sourceId) &&
        layoutNodeIDs.has(edge.targetId),
    );
  }

  private buildLayoutAdjacency(
    layoutNodes: ComponentsNode[],
    layoutEdges: Array<{ sourceId?: string; targetId?: string }>,
  ) {
    const nodeIDs = new Set(layoutNodes.map((node) => node.id as string));
    const adjacencyByNodeID = new Map<string, string[]>();

    for (const node of layoutNodes) {
      adjacencyByNodeID.set(node.id as string, []);
    }

    for (const edge of layoutEdges) {
      const sourceID = edge.sourceId;
      const targetID = edge.targetId;
      if (!sourceID || !targetID) {
        continue;
      }

      if (!nodeIDs.has(sourceID) || !nodeIDs.has(targetID)) {
        continue;
      }

      adjacencyByNodeID.get(sourceID)?.push(targetID);
      adjacencyByNodeID.get(targetID)?.push(sourceID);
    }

    return adjacencyByNodeID;
  }

  private bfsCollectComponent(
    seedNodeID: string,
    adjacencyByNodeID: Map<string, string[]>,
    nodesByID: Map<string, ComponentsNode>,
    visitedNodeIDs: Set<string>,
  ): ComponentsNode[] {
    const componentNodes: ComponentsNode[] = [];
    const queue = [seedNodeID];

    while (queue.length > 0) {
      const currentNodeID = queue.shift();
      if (!currentNodeID || visitedNodeIDs.has(currentNodeID)) {
        continue;
      }

      visitedNodeIDs.add(currentNodeID);
      const currentNode = nodesByID.get(currentNodeID);
      if (currentNode) {
        componentNodes.push(currentNode);
      }

      for (const neighborNodeID of adjacencyByNodeID.get(currentNodeID) || []) {
        if (!visitedNodeIDs.has(neighborNodeID)) {
          queue.push(neighborNodeID);
        }
      }
    }

    return componentNodes;
  }

  private resolveDisconnectedLayoutComponents(
    layoutNodes: ComponentsNode[],
    layoutEdges: Array<{ sourceId?: string; targetId?: string }>,
  ): ComponentsNode[][] {
    if (layoutNodes.length === 0) {
      return [];
    }

    const adjacencyByNodeID = this.buildLayoutAdjacency(layoutNodes, layoutEdges);
    const nodesByID = new Map(layoutNodes.map((node) => [node.id as string, node]));
    const visitedNodeIDs = new Set<string>();
    const components: ComponentsNode[][] = [];

    for (const node of layoutNodes) {
      const seedNodeID = node.id as string;
      if (visitedNodeIDs.has(seedNodeID)) {
        continue;
      }

      const collected = this.bfsCollectComponent(seedNodeID, adjacencyByNodeID, nodesByID, visitedNodeIDs);
      if (collected.length > 0) {
        components.push(collected);
      }
    }

    return components;
  }

  private deduplicateEdges<T extends { id: string }>(edges: T[]): T[] {
    const seen = new Set<string>();
    return edges.filter((edge) => {
      if (seen.has(edge.id)) {
        return false;
      }

      seen.add(edge.id);
      return true;
    });
  }

  private buildElkGraph(
    workflow: CanvasesCanvas,
    layoutNodes: ComponentsNode[],
    outputChannelsByNodeId: Map<string, string[]>,
    positioningEdges?: Array<{ sourceId?: string; targetId?: string; channel?: string }>,
    direction: CanvasFlowDirection = "horizontal",
  ) {
    const layoutEdges = this.resolveLayoutEdges(workflow, layoutNodes);
    const graphEdges = positioningEdges ?? layoutEdges;
    const edgeChannelsBySourceNodeID = new Map<string, Set<string>>();
    const isVertical = direction === "vertical";
    const inputPortSide = isVertical ? "NORTH" : "WEST";
    const outputPortSide = isVertical ? "SOUTH" : "EAST";

    for (const edge of layoutEdges) {
      if (!edge.sourceId) {
        continue;
      }

      const sourceChannels = edgeChannelsBySourceNodeID.get(edge.sourceId) || new Set<string>();
      sourceChannels.add(this.normalizeChannel(edge.channel));
      edgeChannelsBySourceNodeID.set(edge.sourceId, sourceChannels);
    }

    return {
      id: "root",
      layoutOptions: {
        "elk.algorithm": "layered",
        "elk.direction": isVertical ? "DOWN" : "RIGHT",
        "elk.spacing.nodeNode": isVertical ? VERTICAL_FLOW_NODE_NODE_SPACING : HORIZONTAL_FLOW_NODE_NODE_SPACING,
        "elk.layered.spacing.nodeNodeBetweenLayers": isVertical
          ? VERTICAL_FLOW_NODE_NODE_BETWEEN_LAYERS
          : NODE_NODE_BETWEEN_LAYERS,
        "elk.layered.nodePlacement.strategy": "NETWORK_SIMPLEX",
        "elk.contentAlignment": isVertical ? "H_CENTER" : "V_CENTER",
      },
      children: layoutNodes.map((node) => {
        const { width, height } = this.estimateNodeSize(node, direction);
        const nodeId = node.id!;
        const metadataOutputChannels = (outputChannelsByNodeId.get(nodeId) || [])
          .map((channel) => this.normalizeChannel(channel))
          .filter((channel, index, channels) => channels.indexOf(channel) === index);
        const edgeOutputChannels = Array.from(edgeChannelsBySourceNodeID.get(nodeId) || []);
        const outputChannels = appendUniqueChannels(metadataOutputChannels, edgeOutputChannels);
        if (outputChannels.length === 0) {
          outputChannels.push("default");
        }

        // ELK port indexes run clockwise from the top-left. On SOUTH that means
        // right-to-left, while multi-bottom handles render left-to-right. Reverse
        // indexes on vertical canvases so true/false (etc.) children do not cross.
        const outputPortCount = outputChannels.length;
        const ports = [
          {
            id: `${nodeId}__input`,
            properties: {
              "elk.port.side": inputPortSide,
            },
          },
          ...outputChannels.map((channel, index) => ({
            id: `${nodeId}__${channel}`,
            properties: {
              "elk.port.side": outputPortSide,
              "elk.port.index": `${isVertical ? outputPortCount - 1 - index : index}`,
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
      edges: this.deduplicateEdges(
        graphEdges.map((edge) => ({
          id: `${edge.sourceId}->${edge.targetId}->${this.normalizeChannel(edge.channel)}`,
          sources: [`${edge.sourceId}__${this.normalizeChannel(edge.channel)}`],
          targets: [`${edge.targetId}__input`],
        })),
      ),
    };
  }

  private extractLayoutedPositions(layoutedGraph: { children?: Array<{ id: string; x?: number; y?: number }> }) {
    const layoutedPositions = new Map<string, LayoutPosition>();
    for (const child of layoutedGraph.children || []) {
      layoutedPositions.set(child.id, {
        x: child.x || 0,
        y: child.y || 0,
      });
    }

    return layoutedPositions;
  }

  private async resolvePackedLayoutedPositions(
    workflow: CanvasesCanvas,
    layoutNodes: ComponentsNode[],
    outputChannelsByNodeId: Map<string, string[]>,
    direction: CanvasFlowDirection,
  ): Promise<Map<string, LayoutPosition>> {
    const layoutEdges = this.resolveLayoutEdges(workflow, layoutNodes);
    const components = this.resolveDisconnectedLayoutComponents(layoutNodes, layoutEdges);
    if (components.length <= 1) {
      const graph = this.buildElkGraph(
        workflow,
        layoutNodes,
        outputChannelsByNodeId,
        resolveForwardLayoutEdges(layoutNodes, layoutEdges),
        direction,
      );
      const layoutedGraph = await this.elk.layout(graph);
      return this.extractLayoutedPositions(layoutedGraph);
    }

    const sortedComponents = sortComponentsByCurrentPosition(components, direction);
    const packedLayoutedPositions = new Map<string, LayoutPosition>();
    let currentPackOffset = 0;
    const packAlongCrossAxis = direction === "vertical";

    for (const componentNodes of sortedComponents) {
      const componentEdges = this.resolveLayoutEdges(workflow, componentNodes);
      const graph = this.buildElkGraph(
        workflow,
        componentNodes,
        outputChannelsByNodeId,
        resolveForwardLayoutEdges(componentNodes, componentEdges),
        direction,
      );
      const layoutedGraph = await this.elk.layout(graph);
      const componentPositions = this.extractLayoutedPositions(layoutedGraph);
      if (componentPositions.size === 0) {
        continue;
      }

      const bounds = resolveLayoutBounds(componentNodes, componentPositions, (node) =>
        this.estimateNodeSize(node, direction),
      );
      for (const [nodeID, position] of packComponentPositions(
        componentPositions,
        bounds,
        currentPackOffset,
        packAlongCrossAxis,
      ).entries()) {
        packedLayoutedPositions.set(nodeID, position);
      }

      currentPackOffset += packAlongCrossAxis
        ? bounds.width + DISCONNECTED_COMPONENT_HORIZONTAL_GAP_VERTICAL
        : bounds.height + DISCONNECTED_COMPONENT_VERTICAL_GAP;
    }

    return packedLayoutedPositions;
  }

  private resolveNodeOutputChannels(node: ComponentsNode, components: ActionsAction[]): string[] {
    const defaultChannels = ["default"];

    if (node.type === "TYPE_ACTION" && node.component) {
      const meta = components.find((component) => component.name === node.component);
      return meta?.outputChannels?.map((channel) => channel.name!).filter(Boolean) || defaultChannels;
    }

    return defaultChannels;
  }

  private buildOutputChannelsByNodeId(workflow: CanvasesCanvas, components: ActionsAction[]): Map<string, string[]> {
    const map = new Map<string, string[]>();
    for (const node of workflow.spec?.nodes || []) {
      if (!node.id) {
        continue;
      }

      map.set(node.id, this.resolveNodeOutputChannels(node, components));
    }

    return map;
  }
}
