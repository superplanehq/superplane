import type {
  CanvasesCanvas,
  ComponentsEdge,
  ComponentsIntegrationRef,
  ComponentsNode,
  ConfigurationField,
  OrganizationsIntegration,
} from "@/api-client";
import type { AiCanvasOperation, BuildingBlockCategory } from "@/ui/BuildingBlocksSidebar";
import { filterVisibleConfiguration } from "@/utils/components";
import { generateNodeId, generateUniqueNodeName } from "./utils";

type ApplyAiOperationsToWorkflowInput = {
  workflow: CanvasesCanvas;
  operations: AiCanvasOperation[];
  buildingBlocks: BuildingBlockCategory[];
  integrations?: OrganizationsIntegration[];
};

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

export function applyAiOperationsToWorkflow({
  workflow,
  operations,
  buildingBlocks,
  integrations = [],
}: ApplyAiOperationsToWorkflowInput): CanvasesCanvas {
  const blockLookup = new Map(
    buildingBlocks.flatMap((category) => category.blocks.map((block) => [block.name, block])),
  );
  const createdNodeIdsByKey = new Map<string, string>();
  const addedNodeBlockNameByKey = new Map<string, string>();
  for (const operation of operations) {
    if (operation.type === "add_node" && operation.nodeKey) {
      addedNodeBlockNameByKey.set(operation.nodeKey, operation.blockName);
    }
  }
  const updatedNodes: ComponentsNode[] = [...(workflow.spec?.nodes || [])];
  const updatedEdges: ComponentsEdge[] = [...(workflow.spec?.edges || [])];

  const resolveNodeId = (ref?: { nodeKey?: string; nodeId?: string; nodeName?: string }) => {
    if (!ref) return null;
    if (ref.nodeKey && createdNodeIdsByKey.has(ref.nodeKey)) {
      return createdNodeIdsByKey.get(ref.nodeKey) || null;
    }
    if (ref.nodeId) return ref.nodeId;
    if (ref.nodeName) {
      const found = updatedNodes.find((node) => node.id === ref.nodeName || node.name === ref.nodeName);
      return found?.id || null;
    }
    return null;
  };

  const getBlockNameForNodeRef = (ref?: { nodeKey?: string; nodeId?: string; nodeName?: string }) => {
    if (!ref) {
      return null;
    }

    if (ref.nodeKey) {
      const blockNameFromKey = addedNodeBlockNameByKey.get(ref.nodeKey);
      if (blockNameFromKey) {
        return blockNameFromKey;
      }
    }

    const nodeId = resolveNodeId(ref);
    if (!nodeId) {
      return null;
    }

    const node = updatedNodes.find((candidate) => candidate.id === nodeId);
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
  };

  const resolveOutputChannelsForSourceRef = (ref?: { nodeKey?: string; nodeId?: string; nodeName?: string }) => {
    const blockName = getBlockNameForNodeRef(ref);
    if (!blockName) {
      return [];
    }

    const block = blockLookup.get(blockName);
    if (!block?.outputChannels) {
      return [];
    }

    return block.outputChannels
      .map((channel) => ({
        name: channel?.name || "",
        label: "label" in channel ? channel.label || "" : "",
        description: "description" in channel ? channel.description || "" : "",
      }))
      .filter((channel): channel is { name: string; label: string; description: string } => channel.name.length > 0);
  };

  const normalizedTokens = (value: string) =>
    value
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, " ")
      .trim()
      .split(/\s+/)
      .filter(Boolean);

  const scoreChannelAsSuccessPath = (channel: { name: string; label: string; description: string }) => {
    const text = `${channel.name} ${channel.label} ${channel.description}`.toLowerCase();
    const tokens = new Set(normalizedTokens(text));

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
  };

  const resolveConnectionChannel = (source: {
    nodeKey?: string;
    nodeId?: string;
    nodeName?: string;
    handleId?: string | null;
  }) => {
    const explicitChannel = source.handleId?.trim();
    if (explicitChannel) {
      return explicitChannel;
    }

    const outputChannels = resolveOutputChannelsForSourceRef(source);
    const outputChannelNames = outputChannels.map((channel) => channel.name);
    if (outputChannelNames.includes("default")) {
      return "default";
    }

    if (outputChannels.length > 0) {
      const sorted = [...outputChannels].sort((left, right) => {
        return scoreChannelAsSuccessPath(right) - scoreChannelAsSuccessPath(left);
      });
      return sorted[0].name;
    }

    return "default";
  };

  for (const operation of operations) {
    if (operation.type === "add_node") {
      const block = blockLookup.get(operation.blockName);
      if (!block) {
        continue;
      }

      const schemaValidationErrors = validateConfigurationBySchema(
        operation.configuration || {},
        block.configuration || [],
      );
      if (schemaValidationErrors.length > 0) {
        throw new Error(`Invalid node configuration for '${block.name}': ${schemaValidationErrors.join("; ")}`);
      }

      const filteredConfiguration = filterVisibleConfiguration(
        operation.configuration || {},
        block.configuration || [],
      );

      const existingNodeNames = updatedNodes.map((node) => node.name || "").filter(Boolean);
      const uniqueNodeName = generateUniqueNodeName(operation.nodeName || block.name || "node", existingNodeNames);
      const newNodeId = generateNodeId(block.name || "node", uniqueNodeName);

      const newNode: ComponentsNode = {
        id: newNodeId,
        name: uniqueNodeName,
        type:
          block.type === "trigger"
            ? "TYPE_TRIGGER"
            : block.type === "blueprint"
              ? "TYPE_BLUEPRINT"
              : block.name === "annotation"
                ? "TYPE_WIDGET"
                : "TYPE_COMPONENT",
        configuration: filteredConfiguration,
        position: operation.position
          ? {
              x: Math.round(operation.position.x),
              y: Math.round(operation.position.y),
            }
          : {
              x: 0,
              y: 0,
            },
      };

      const integrationRef = resolveIntegrationRefForBlock(block.integrationName, integrations);
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

      updatedNodes.push(newNode);
      if (operation.nodeKey) {
        createdNodeIdsByKey.set(operation.nodeKey, newNodeId);
      }

      const sourceNodeId = resolveNodeId(operation.source);
      if (sourceNodeId) {
        updatedEdges.push({
          sourceId: sourceNodeId,
          targetId: newNodeId,
          channel: resolveConnectionChannel(operation.source || {}),
        });
      }
      continue;
    }

    if (operation.type === "connect_nodes") {
      const sourceId = resolveNodeId(operation.source);
      const targetId = resolveNodeId(operation.target);
      if (!sourceId || !targetId) {
        continue;
      }
      const channel = resolveConnectionChannel(operation.source);

      const edgeExists = updatedEdges.some(
        (edge) => edge.sourceId === sourceId && edge.targetId === targetId && edge.channel === channel,
      );
      if (!edgeExists) {
        updatedEdges.push({
          sourceId,
          targetId,
          channel,
        });
      }
      continue;
    }

    if (operation.type === "disconnect_nodes") {
      const sourceId = resolveNodeId(operation.source);
      const targetId = resolveNodeId(operation.target);
      if (!sourceId || !targetId) {
        continue;
      }

      const explicitChannel = operation.source.handleId?.trim();
      for (let edgeIndex = updatedEdges.length - 1; edgeIndex >= 0; edgeIndex -= 1) {
        const edge = updatedEdges[edgeIndex];
        if (edge.sourceId !== sourceId || edge.targetId !== targetId) {
          continue;
        }
        if (explicitChannel && edge.channel !== explicitChannel) {
          continue;
        }
        updatedEdges.splice(edgeIndex, 1);
      }
      continue;
    }

    if (operation.type === "update_node_config") {
      const targetId = resolveNodeId(operation.target);
      if (!targetId) {
        continue;
      }

      const nodeIndex = updatedNodes.findIndex((node) => node.id === targetId);
      if (nodeIndex === -1) {
        continue;
      }

      const targetNode = updatedNodes[nodeIndex];
      const targetBlockName = getBlockNameForNodeRef(operation.target);
      const targetBlock = targetBlockName ? blockLookup.get(targetBlockName) : undefined;
      const mergedConfiguration = {
        ...(targetNode.configuration || {}),
        ...(operation.configuration || {}),
      };
      const targetSchema = targetBlock?.configuration || [];
      if (targetSchema.length > 0) {
        const schemaValidationErrors = validateConfigurationBySchema(mergedConfiguration, targetSchema);
        if (schemaValidationErrors.length > 0) {
          throw new Error(`Invalid node configuration for '${targetBlockName}': ${schemaValidationErrors.join("; ")}`);
        }
      }
      const nextConfiguration =
        targetSchema.length > 0 ? filterVisibleConfiguration(mergedConfiguration, targetSchema) : mergedConfiguration;
      updatedNodes[nodeIndex] = {
        ...targetNode,
        name: operation.nodeName || targetNode.name,
        configuration: nextConfiguration,
      };
      continue;
    }

    if (operation.type === "delete_node") {
      const targetId = resolveNodeId(operation.target);
      if (!targetId) {
        continue;
      }

      const nodeIndex = updatedNodes.findIndex((node) => node.id === targetId);
      if (nodeIndex === -1) {
        continue;
      }

      updatedNodes.splice(nodeIndex, 1);

      for (let edgeIndex = updatedEdges.length - 1; edgeIndex >= 0; edgeIndex -= 1) {
        const edge = updatedEdges[edgeIndex];
        if (edge.sourceId === targetId || edge.targetId === targetId) {
          updatedEdges.splice(edgeIndex, 1);
        }
      }
    }
  }

  return {
    ...workflow,
    spec: {
      ...workflow.spec,
      nodes: updatedNodes,
      edges: updatedEdges,
    },
  };
}

// Validate configuration by schema

const LIST_LIKE_FIELD_TYPES = new Set(["multi-select", "list", "any-predicate-list", "days-of-week"]);

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function validateConfigurationBySchema(
  configuration: Record<string, unknown>,
  fields: ConfigurationField[],
  pathPrefix = "configuration",
): string[] {
  const errors: string[] = [];
  const fieldsByName = new Map(fields.map((field) => [field.name || "", field]));

  for (const [name, value] of Object.entries(configuration)) {
    const field = fieldsByName.get(name);
    if (!field?.type) {
      continue;
    }
    errors.push(...validateConfigurationFieldValue(field, value, `${pathPrefix}.${name}`));
  }

  return errors;
}

function validateConfigurationFieldValue(field: ConfigurationField, value: unknown, fieldPath: string): string[] {
  if (!field.type) {
    return [];
  }

  if (LIST_LIKE_FIELD_TYPES.has(field.type)) {
    return validateListLikeConfigurationField(field, value, fieldPath);
  }

  if (field.type === "object") {
    return validateObjectConfigurationField(field, value, fieldPath);
  }

  if (field.type === "number" && typeof value !== "number") {
    return [`${fieldPath} must be a number`];
  }

  if (field.type === "boolean" && typeof value !== "boolean") {
    return [`${fieldPath} must be a boolean`];
  }

  return [];
}

function validateListLikeConfigurationField(field: ConfigurationField, value: unknown, fieldPath: string): string[] {
  if (!Array.isArray(value)) {
    return [`${fieldPath} must be an array for field type '${field.type}'`];
  }

  const itemSchema = field.typeOptions?.list?.itemDefinition?.schema;
  if (!itemSchema) {
    return [];
  }

  const errors: string[] = [];
  for (let index = 0; index < value.length; index += 1) {
    const item = value[index];
    if (!isRecord(item)) {
      continue;
    }
    errors.push(...validateConfigurationBySchema(item, itemSchema, `${fieldPath}[${index}]`));
  }

  return errors;
}

function validateObjectConfigurationField(field: ConfigurationField, value: unknown, fieldPath: string): string[] {
  if (!isRecord(value)) {
    return [`${fieldPath} must be an object`];
  }

  const objectSchema = field.typeOptions?.object?.schema;
  if (!objectSchema) {
    return [];
  }

  return validateConfigurationBySchema(value, objectSchema, fieldPath);
}
