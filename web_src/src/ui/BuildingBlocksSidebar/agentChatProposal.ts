import type { AiCanvasOperation } from "./index";
import type { AiBuilderProposal } from "./agentChat";

type JsonObject = Record<string, unknown>;

function isRecord(value: unknown): value is JsonObject {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function normalizeNodeRef(
  value: unknown,
): { nodeKey?: string; nodeId?: string; nodeName?: string; handleId?: string | null } | null {
  if (!isRecord(value)) {
    return null;
  }

  const nodeKey = typeof value.nodeKey === "string" ? value.nodeKey : undefined;
  const nodeId = typeof value.nodeId === "string" ? value.nodeId : undefined;
  const nodeName = typeof value.nodeName === "string" ? value.nodeName : undefined;
  const handleId = typeof value.handleId === "string" ? value.handleId : value.handleId === null ? null : undefined;

  if (!nodeKey && !nodeId && !nodeName) {
    return null;
  }

  return { nodeKey, nodeId, nodeName, handleId };
}

function normalizeAddNodeOperation(value: JsonObject): AiCanvasOperation | null {
  const blockName = typeof value.blockName === "string" ? value.blockName : "";
  if (!blockName) {
    return null;
  }

  const operation: AiCanvasOperation = {
    type: "add_node",
    blockName,
    nodeKey: typeof value.nodeKey === "string" ? value.nodeKey : undefined,
    nodeName: typeof value.nodeName === "string" ? value.nodeName : undefined,
  };
  if (isRecord(value.configuration)) {
    operation.configuration = value.configuration;
  }
  if (isRecord(value.position) && typeof value.position.x === "number" && typeof value.position.y === "number") {
    operation.position = { x: value.position.x, y: value.position.y };
  }

  const source = normalizeNodeRef(value.source);
  if (source) {
    operation.source = source;
  }

  return operation;
}

function normalizeConnectionOperation(
  value: JsonObject,
  type: "connect_nodes" | "disconnect_nodes",
): AiCanvasOperation | null {
  const source = normalizeNodeRef(value.source);
  const target = normalizeNodeRef(value.target);
  if (!source || !target) {
    return null;
  }

  return { type, source, target };
}

function normalizeUpdateNodeConfigOperation(value: JsonObject): AiCanvasOperation | null {
  const target = normalizeNodeRef(value.target);
  if (!target) {
    return null;
  }

  return {
    type: "update_node_config",
    target,
    configuration: isRecord(value.configuration) ? value.configuration : {},
    nodeName: typeof value.nodeName === "string" ? value.nodeName : undefined,
  };
}

function normalizeDeleteNodeOperation(value: JsonObject): AiCanvasOperation | null {
  const target = normalizeNodeRef(value.target);
  if (!target) {
    return null;
  }

  return {
    type: "delete_node",
    target,
  };
}

function normalizeAiOperation(value: unknown): AiCanvasOperation | null {
  if (!isRecord(value) || typeof value.type !== "string") {
    return null;
  }

  switch (value.type) {
    case "add_node":
      return normalizeAddNodeOperation(value);
    case "connect_nodes":
      return normalizeConnectionOperation(value, "connect_nodes");
    case "disconnect_nodes":
      return normalizeConnectionOperation(value, "disconnect_nodes");
    case "update_node_config":
      return normalizeUpdateNodeConfigOperation(value);
    case "delete_node":
      return normalizeDeleteNodeOperation(value);
    default:
      return null;
  }
}

export function normalizeAiProposal(value: unknown): AiBuilderProposal | null {
  if (!isRecord(value)) {
    return null;
  }

  const summary = typeof value.summary === "string" ? value.summary.trim() : "";
  if (!summary) {
    return null;
  }

  const operationsRaw = Array.isArray(value.operations) ? value.operations : [];
  const operations = operationsRaw
    .map((operation) => normalizeAiOperation(operation))
    .filter((operation): operation is AiCanvasOperation => Boolean(operation));
  if (operations.length === 0) {
    return null;
  }

  return {
    id: `proposal-${Date.now()}`,
    summary,
    operations,
  };
}
