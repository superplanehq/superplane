import type {
  BlueprintsBlueprint,
  CanvasesCanvas,
  ComponentsComponent,
  ComponentsEdge,
  ComponentsIntegrationRef,
  ComponentsNode,
  ComponentsNodeType,
  OrganizationsIntegration,
} from "@/api-client";
import type { LayoutEngine } from "@/lib/layout";
import type { BuildingBlock, BuildingBlockCategory } from "@/ui/BuildingBlocksSidebar";
import { generateNodeId, generateUniqueNodeName } from "@/pages/workflowv2/utils";
import type {
  CanvasOperation,
  NodeRef,
  SourceNodeRef,
  AddNodeOperation,
  ConnectionNodesOperation,
  UpdateNodeConfigOperation,
  DeleteNodeOperation,
} from "./types";

type ScoredChannel = {
  name: string;
  label: string;
  description: string;
};

export type CanvasBuilderOptions = {
  canvas: CanvasesCanvas;
  buildingBlocks: BuildingBlockCategory[];
  integrations?: OrganizationsIntegration[];
  components?: ComponentsComponent[];
  blueprints?: BlueprintsBlueprint[];
};

/*
 * Applies a list of AI-generated canvas operations to a canvas.
 * The operations are applied in the order they are provided.
 *
 * @param options - Builder configuration with canvas, building blocks and optional metadata.
 * @returns The updated canvas.
 */
export class CanvasBuilder {
  private readonly canvas: CanvasesCanvas;
  private readonly integrations: OrganizationsIntegration[];
  private readonly blockLookup: Map<string, BuildingBlockCategory["blocks"][number]>;
  private readonly layoutComponents: ComponentsComponent[];
  private readonly layoutBlueprints: BlueprintsBlueprint[];
  private createdNodeIdsByKey: Map<string, string>;
  private addedNodeBlockNameByKey: Map<string, string>;
  private addedNodeIds: Set<string>;
  private updatedNodes: ComponentsNode[];
  private updatedEdges: ComponentsEdge[];

  constructor(options: CanvasBuilderOptions) {
    const { canvas, buildingBlocks, integrations = [], components = [], blueprints = [] } = options;

    this.canvas = canvas;
    this.integrations = integrations;
    this.blockLookup = new Map(
      buildingBlocks.flatMap((category) => category.blocks.map((block) => [block.name, block])),
    );
    this.layoutComponents = components;
    this.layoutBlueprints = blueprints;
    this.createdNodeIdsByKey = new Map();
    this.addedNodeBlockNameByKey = new Map();
    this.addedNodeIds = new Set();
    this.updatedNodes = [];
    this.updatedEdges = [];
  }

  async apply(operations: CanvasOperation[], layout?: LayoutEngine): Promise<CanvasesCanvas> {
    this.initializeState(operations);

    for (const operation of operations) {
      switch (operation.type) {
        case "add_node":
          this.applyAddNodeOperation(operation);
          break;
        case "connect_nodes":
        case "disconnect_nodes":
          this.applyConnectionOperation(operation);
          break;
        case "update_node_config":
          this.applyUpdateNodeConfigOperation(operation);
          break;
        case "delete_node":
          this.applyDeleteNodeOperation(operation);
          break;
      }
    }

    const updatedCanvas = {
      ...this.canvas,
      spec: {
        ...this.canvas.spec,
        nodes: this.updatedNodes,
        edges: this.updatedEdges,
      },
    };

    return this.applyLayoutIfNewNodesWereAdded(updatedCanvas, layout);
  }

  private initializeState(operations: CanvasOperation[]): void {
    this.createdNodeIdsByKey = new Map();
    this.addedNodeBlockNameByKey = new Map();
    this.addedNodeIds = new Set();
    this.updatedNodes = [...(this.canvas.spec?.nodes || [])];
    this.updatedEdges = [...(this.canvas.spec?.edges || [])];

    for (const operation of operations) {
      if (operation.type === "add_node" && operation.nodeKey) {
        this.addedNodeBlockNameByKey.set(operation.nodeKey, operation.blockName);
      }
    }
  }

  private resolveNodeId(ref?: NodeRef): string | null {
    if (!ref) {
      return null;
    }

    if (ref.nodeKey && this.createdNodeIdsByKey.has(ref.nodeKey)) {
      return this.createdNodeIdsByKey.get(ref.nodeKey) || null;
    }
    if (ref.nodeId) {
      return ref.nodeId;
    }
    if (ref.nodeName) {
      const found = this.updatedNodes.find((node) => node.id === ref.nodeName || node.name === ref.nodeName);
      return found?.id || null;
    }

    return null;
  }

  private getBlockNameForNodeRef(ref?: NodeRef): string | null {
    if (!ref) {
      return null;
    }

    if (ref.nodeKey) {
      const blockNameFromKey = this.addedNodeBlockNameByKey.get(ref.nodeKey);
      if (blockNameFromKey) {
        return blockNameFromKey;
      }
    }

    const nodeId = this.resolveNodeId(ref);
    if (!nodeId) {
      return null;
    }

    const node = this.updatedNodes.find((candidate) => candidate.id === nodeId);
    if (!node) {
      return null;
    }

    if (node.type === "TYPE_TRIGGER") {
      return node.trigger?.name || null;
    }
    if (node.type === "TYPE_COMPONENT") {
      return node.component?.name || null;
    }

    return null;
  }

  private resolveOutputChannelsForSourceRef(ref?: NodeRef): ScoredChannel[] {
    const blockName = this.getBlockNameForNodeRef(ref);
    if (!blockName) {
      return [];
    }

    const block = this.blockLookup.get(blockName);
    if (!block?.outputChannels) {
      return [];
    }

    return block.outputChannels
      .map((channel) => ({
        name: channel?.name || "",
        label: "label" in channel ? channel.label || "" : "",
        description: "description" in channel ? channel.description || "" : "",
      }))
      .filter((channel): channel is ScoredChannel => channel.name.length > 0);
  }

  private normalizedTokens(value: string): string[] {
    return value
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, " ")
      .trim()
      .split(/\s+/)
      .filter(Boolean);
  }

  private scoreChannelAsSuccessPath(channel: ScoredChannel): number {
    const text = `${channel.name} ${channel.label} ${channel.description}`.toLowerCase();
    const tokens = new Set(this.normalizedTokens(text));
    const hasToken = (token: string) => tokens.has(token);

    const positiveTokens = [
      "success",
      "successful",
      "succeeded",
      "pass",
      "passed",
      "ok",
      "done",
      "complete",
      "completed",
      "finish",
      "finished",
      "next",
      "continue",
      "proceed",
      "then",
    ];
    const negativeTokens = [
      "fail",
      "failed",
      "failure",
      "error",
      "errored",
      "cancel",
      "cancelled",
      "canceled",
      "reject",
      "rejected",
      "deny",
      "denied",
      "abort",
      "aborted",
      "timeout",
      "timedout",
      "exception",
    ];

    let score = 0;
    if (channel.name === "default") {
      score += 100;
    }
    for (const token of positiveTokens) {
      if (hasToken(token)) {
        score += 20;
      }
    }
    for (const token of negativeTokens) {
      if (hasToken(token)) {
        score -= 40;
      }
    }
    if (text.includes("on success") || text.includes("when successful")) {
      score += 30;
    }
    if (text.includes("on failure") || text.includes("when failed") || text.includes("on error")) {
      score -= 60;
    }

    return score;
  }

  private resolveConnectionChannel(source?: SourceNodeRef): string {
    const explicitChannel = source?.handleId?.trim();
    if (explicitChannel) {
      return explicitChannel;
    }

    const outputChannels = this.resolveOutputChannelsForSourceRef(source);
    const outputChannelNames = outputChannels.map((channel) => channel.name);
    if (outputChannelNames.includes("default")) {
      return "default";
    }

    if (outputChannels.length > 0) {
      const sorted = [...outputChannels].sort((left, right) => {
        return this.scoreChannelAsSuccessPath(right) - this.scoreChannelAsSuccessPath(left);
      });
      return sorted[0].name;
    }

    return "default";
  }

  private blockTypeFromBlock(block: BuildingBlock): ComponentsNodeType {
    switch (block.type) {
      case "trigger":
        return "TYPE_TRIGGER";
      case "blueprint":
        return "TYPE_BLUEPRINT";
      default:
        return block.name === "annotation" ? "TYPE_WIDGET" : "TYPE_COMPONENT";
    }
  }

  private positionFromOperation(operation: AddNodeOperation): { x: number; y: number } {
    return operation.position
      ? {
          x: Math.round(operation.position.x),
          y: Math.round(operation.position.y),
        }
      : { x: 0, y: 0 };
  }

  private applyAddNodeOperation(operation: AddNodeOperation): void {
    const block = this.blockLookup.get(operation.blockName);
    if (!block) {
      return;
    }

    const existingNodeNames = this.updatedNodes.map((node) => node.name || "").filter(Boolean);
    const uniqueNodeName = generateUniqueNodeName(operation.nodeName || block.name || "node", existingNodeNames);
    const newNodeId = generateNodeId(block.name || "node", uniqueNodeName);

    const newNode: ComponentsNode = {
      id: newNodeId,
      name: uniqueNodeName,
      type: this.blockTypeFromBlock(block),
      configuration: operation.configuration || {},
      position: this.positionFromOperation(operation),
    };

    const integrationRef = resolveIntegrationRefForBlock(block.integrationName, this.integrations);
    if (integrationRef) {
      newNode.integration = integrationRef;
    }

    if (block.name === "annotation") {
      newNode.widget = { name: "annotation" };
      newNode.configuration = { text: "", color: "yellow" };
    } else if (block.type === "component") {
      newNode.component = { name: block.name };
    } else if (block.type === "trigger") {
      newNode.trigger = { name: block.name };
    } else if (block.type === "blueprint") {
      newNode.blueprint = { id: block.id };
    }

    this.updatedNodes.push(newNode);
    this.addedNodeIds.add(newNodeId);
    if (operation.nodeKey) {
      this.createdNodeIdsByKey.set(operation.nodeKey, newNodeId);
    }

    const sourceNodeId = this.resolveNodeId(operation.source);
    if (!sourceNodeId) {
      return;
    }

    this.updatedEdges.push({
      sourceId: sourceNodeId,
      targetId: newNodeId,
      channel: this.resolveConnectionChannel(operation.source),
    });
  }

  private applyConnectionOperation(operation: ConnectionNodesOperation): void {
    const sourceId = this.resolveNodeId(operation.source);
    const targetId = this.resolveNodeId(operation.target);
    if (!sourceId || !targetId) {
      return;
    }

    if (operation.type === "connect_nodes") {
      const channel = this.resolveConnectionChannel(operation.source);
      const edgeExists = this.updatedEdges.some(
        (edge) => edge.sourceId === sourceId && edge.targetId === targetId && edge.channel === channel,
      );
      if (!edgeExists) {
        this.updatedEdges.push({
          sourceId,
          targetId,
          channel,
        });
      }
      return;
    }

    const explicitChannel = operation.source.handleId?.trim();
    for (let edgeIndex = this.updatedEdges.length - 1; edgeIndex >= 0; edgeIndex -= 1) {
      const edge = this.updatedEdges[edgeIndex];
      if (edge.sourceId !== sourceId || edge.targetId !== targetId) {
        continue;
      }
      if (explicitChannel && edge.channel !== explicitChannel) {
        continue;
      }
      this.updatedEdges.splice(edgeIndex, 1);
    }
  }

  private applyUpdateNodeConfigOperation(operation: UpdateNodeConfigOperation): void {
    const targetId = this.resolveNodeId(operation.target);
    if (!targetId) {
      return;
    }

    const nodeIndex = this.updatedNodes.findIndex((node) => node.id === targetId);
    if (nodeIndex === -1) {
      return;
    }

    const targetNode = this.updatedNodes[nodeIndex];
    const configuration = {
      ...(targetNode.configuration || {}),
      ...(operation.configuration || {}),
    };

    this.updatedNodes[nodeIndex] = {
      ...targetNode,
      name: operation.nodeName || targetNode.name,
      configuration: configuration,
    };
  }

  private applyDeleteNodeOperation(operation: DeleteNodeOperation): void {
    const targetId = this.resolveNodeId(operation.target);
    if (!targetId) {
      return;
    }

    const nodeIndex = this.updatedNodes.findIndex((node) => node.id === targetId);
    if (nodeIndex === -1) {
      return;
    }

    const targetNodeId = this.updatedNodes[nodeIndex]?.id;
    this.updatedNodes.splice(nodeIndex, 1);
    if (typeof targetNodeId === "string") {
      this.addedNodeIds.delete(targetNodeId);
    }
    for (let edgeIndex = this.updatedEdges.length - 1; edgeIndex >= 0; edgeIndex -= 1) {
      const edge = this.updatedEdges[edgeIndex];
      if (edge.sourceId === targetId || edge.targetId === targetId) {
        this.updatedEdges.splice(edgeIndex, 1);
      }
    }
  }

  private async applyLayoutIfNewNodesWereAdded(
    updatedCanvas: CanvasesCanvas,
    layout?: LayoutEngine,
  ): Promise<CanvasesCanvas> {
    if (!layout) {
      return updatedCanvas;
    }

    if (this.addedNodeIds.size === 0) {
      return updatedCanvas;
    }

    return layout.apply(updatedCanvas, {
      components: this.layoutComponents,
      blueprints: this.layoutBlueprints,
    });
  }
}

function normalizeIntegrationName(name?: string): string {
  return (name || "")
    .trim()
    .toLowerCase()
    .replace(/[\s_-]+/g, "");
}

function resolveIntegrationRefForBlock(
  blockIntegrationName: string | undefined,
  integrations: OrganizationsIntegration[],
): ComponentsIntegrationRef | undefined {
  const normalized = normalizeIntegrationName(blockIntegrationName);
  if (!normalized) {
    return undefined;
  }

  const matchingIntegrations = integrations.filter((integration) => {
    return normalizeIntegrationName(integration.spec?.integrationName) === normalized;
  });
  if (matchingIntegrations.length === 0) {
    return undefined;
  }

  const selectedIntegration =
    matchingIntegrations.find((integration) => (integration.status?.state || "").toLowerCase() === "ready") ||
    matchingIntegrations[0];
  if (!selectedIntegration.metadata?.id && !selectedIntegration.metadata?.name) {
    return undefined;
  }

  return {
    id: selectedIntegration.metadata?.id,
    name: selectedIntegration.metadata?.name,
  };
}
