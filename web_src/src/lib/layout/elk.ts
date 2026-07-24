import type { CanvasesCanvas, ActionsAction, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import ELK from "elkjs/lib/elk.bundled.js";
import {
  DEFAULT_LAYOUT_DIRECTION,
  type LayoutDirection,
  type LayoutEngine,
  type LayoutEngineApplyOptions,
} from "./types";
import {
  appendUniqueChannels,
  resolveConnectedComponentNodeIds,
  resolveDisconnectedComponents,
  resolveForwardLayoutEdges,
} from "./layoutGraph";

const DEFAULT_NODE_WIDTH = 420;
const DEFAULT_NODE_HEIGHT = 180;
const ANNOTATION_NODE_WIDTH = 320;
const ANNOTATION_NODE_HEIGHT = 200;
// Gap kept between disconnected components along the packing axis (perpendicular
// to the flow direction) so independent sub-graphs never overlap.
const DISCONNECTED_COMPONENT_GAP = 220;

type LayoutPosition = {
  x: number;
  y: number;
};

export class ElkLayoutEngine implements LayoutEngine {
  private readonly elk = new ELK();

  estimateNodeSize(node: ComponentsNode): { width: number; height: number } {
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

    const direction = options?.direction ?? DEFAULT_LAYOUT_DIRECTION;
    const outputChannelsByNodeId = this.buildOutputChannelsByNodeId(workflow, options?.components || []);
    const layoutedPositions = await this.resolvePackedLayoutedPositions(
      workflow,
      layoutNodes,
      outputChannelsByNodeId,
      direction,
    );

    if (layoutedPositions.size === 0) {
      return workflow;
    }

    const minCurrentPosition = this.resolveMinPositionFromNodes(layoutNodes);
    const minLayoutPosition = this.resolveMinPositionFromLayout(layoutedPositions);
    const updatedNodes = this.applyLayoutedPositions(nodes, layoutedPositions, {
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

  private resolveScopedNodeIDs(
    workflow: CanvasesCanvas,
    flowNodes: ComponentsNode[],
    options: LayoutEngineApplyOptions | undefined,
  ): string[] {
    const seedNodeIDs = this.normalizeRequestedNodeIDs(flowNodes, options?.nodeIds || []);
    const scope = this.resolveLayoutScope(options, seedNodeIDs.length > 0);

    if (scope === "connected-component") {
      return resolveConnectedComponentNodeIds(flowNodes, workflow.spec?.edges || [], seedNodeIDs);
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

  private resolvePortSides(direction: LayoutDirection): { input: string; output: string } {
    if (direction === "vertical") {
      return { input: "NORTH", output: "SOUTH" };
    }

    return { input: "WEST", output: "EAST" };
  }

  private buildElkGraph(
    workflow: CanvasesCanvas,
    layoutNodes: ComponentsNode[],
    outputChannelsByNodeId: Map<string, string[]>,
    positioningEdges?: Array<{ sourceId?: string; targetId?: string; channel?: string }>,
    direction: LayoutDirection = DEFAULT_LAYOUT_DIRECTION,
  ) {
    const layoutEdges = this.resolveLayoutEdges(workflow, layoutNodes);
    const graphEdges = positioningEdges ?? layoutEdges;
    const edgeChannelsBySourceNodeID = new Map<string, Set<string>>();
    const portSides = this.resolvePortSides(direction);

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
        "elk.direction": direction === "vertical" ? "DOWN" : "RIGHT",
        "elk.spacing.nodeNode": "100",
        "elk.layered.spacing.nodeNodeBetweenLayers": "180",
        "elk.layered.nodePlacement.strategy": "NETWORK_SIMPLEX",
        "elk.contentAlignment": direction === "vertical" ? "H_CENTER" : "V_CENTER",
      },
      children: layoutNodes.map((node) => {
        const { width, height } = this.estimateNodeSize(node);
        const nodeId = node.id!;
        const metadataOutputChannels = (outputChannelsByNodeId.get(nodeId) || [])
          .map((channel) => this.normalizeChannel(channel))
          .filter((channel, index, channels) => channels.indexOf(channel) === index);
        const edgeOutputChannels = Array.from(edgeChannelsBySourceNodeID.get(nodeId) || []);
        const outputChannels = appendUniqueChannels(metadataOutputChannels, edgeOutputChannels);
        if (outputChannels.length === 0) {
          outputChannels.push("default");
        }

        const ports = [
          {
            id: `${nodeId}__input`,
            properties: {
              "elk.port.side": portSides.input,
            },
          },
          ...outputChannels.map((channel, index) => ({
            id: `${nodeId}__${channel}`,
            properties: {
              "elk.port.side": portSides.output,
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

  private resolveLayoutBounds(layoutNodes: ComponentsNode[], layoutedPositions: Map<string, LayoutPosition>) {
    let minX = Number.POSITIVE_INFINITY;
    let minY = Number.POSITIVE_INFINITY;
    let maxX = Number.NEGATIVE_INFINITY;
    let maxY = Number.NEGATIVE_INFINITY;

    for (const node of layoutNodes) {
      const nodeID = node.id;
      if (!nodeID) {
        continue;
      }

      const position = layoutedPositions.get(nodeID);
      if (!position) {
        continue;
      }

      const { width, height } = this.estimateNodeSize(node);
      minX = Math.min(minX, position.x);
      minY = Math.min(minY, position.y);
      maxX = Math.max(maxX, position.x + width);
      maxY = Math.max(maxY, position.y + height);
    }

    if (!Number.isFinite(minX) || !Number.isFinite(minY) || !Number.isFinite(maxX) || !Number.isFinite(maxY)) {
      return {
        minX: 0,
        minY: 0,
        maxX: 0,
        maxY: 0,
        width: 0,
        height: 0,
      };
    }

    return {
      minX,
      minY,
      maxX,
      maxY,
      width: maxX - minX,
      height: maxY - minY,
    };
  }

  private sortComponentsByCurrentPosition(
    components: ComponentsNode[][],
    direction: LayoutDirection = DEFAULT_LAYOUT_DIRECTION,
  ): ComponentsNode[][] {
    const isVertical = direction === "vertical";
    return [...components].sort((componentA, componentB) => {
      const a = this.resolveMinPositionFromNodes(componentA);
      const b = this.resolveMinPositionFromNodes(componentB);

      // Order components along the packing axis (x for vertical, y for horizontal)
      // so their on-screen arrangement matches the user's existing layout.
      const primaryA = isVertical ? a.x : a.y;
      const primaryB = isVertical ? b.x : b.y;
      if (primaryA !== primaryB) {
        return primaryA - primaryB;
      }

      const secondaryA = isVertical ? a.y : a.x;
      const secondaryB = isVertical ? b.y : b.x;
      return secondaryA - secondaryB;
    });
  }

  private async resolvePackedLayoutedPositions(
    workflow: CanvasesCanvas,
    layoutNodes: ComponentsNode[],
    outputChannelsByNodeId: Map<string, string[]>,
    direction: LayoutDirection = DEFAULT_LAYOUT_DIRECTION,
  ): Promise<Map<string, LayoutPosition>> {
    const layoutEdges = this.resolveLayoutEdges(workflow, layoutNodes);
    const components = resolveDisconnectedComponents(layoutNodes, layoutEdges);
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

    const sortedComponents = this.sortComponentsByCurrentPosition(components, direction);
    const packedLayoutedPositions = new Map<string, LayoutPosition>();
    // Disconnected components stack perpendicular to the flow so they never
    // overlap: below one another for horizontal flows, side-by-side for vertical.
    const isVertical = direction === "vertical";
    let currentOffset = 0;

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

      const bounds = this.resolveLayoutBounds(componentNodes, componentPositions);
      for (const [nodeID, position] of componentPositions.entries()) {
        packedLayoutedPositions.set(nodeID, {
          x: position.x - bounds.minX + (isVertical ? currentOffset : 0),
          y: position.y - bounds.minY + (isVertical ? 0 : currentOffset),
        });
      }

      currentOffset += (isVertical ? bounds.width : bounds.height) + DISCONNECTED_COMPONENT_GAP;
    }

    return packedLayoutedPositions;
  }

  private resolveMinPositionFromNodes(nodes: ComponentsNode[]): LayoutPosition {
    let minX = Number.POSITIVE_INFINITY;
    let minY = Number.POSITIVE_INFINITY;

    for (const node of nodes) {
      minX = Math.min(minX, node.position?.x || 0);
      minY = Math.min(minY, node.position?.y || 0);
    }

    if (!Number.isFinite(minX)) {
      minX = 0;
    }

    if (!Number.isFinite(minY)) {
      minY = 0;
    }

    return { x: minX, y: minY };
  }

  private resolveMinPositionFromLayout(layoutedPositions: Map<string, LayoutPosition>): LayoutPosition {
    let minX = Number.POSITIVE_INFINITY;
    let minY = Number.POSITIVE_INFINITY;

    layoutedPositions.forEach((position) => {
      minX = Math.min(minX, position.x);
      minY = Math.min(minY, position.y);
    });

    if (!Number.isFinite(minX)) {
      minX = 0;
    }

    if (!Number.isFinite(minY)) {
      minY = 0;
    }

    return { x: minX, y: minY };
  }

  private applyLayoutedPositions(
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
