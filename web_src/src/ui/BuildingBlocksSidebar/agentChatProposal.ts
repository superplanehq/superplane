import type {
  CanvasChangesetChange,
  CanvasChangesetChangeEdge,
  CanvasChangesetChangeNode,
  CanvasChangesetChangeType,
  CanvasesCanvasChangeset,
} from "@/api-client";
import type { AiBuilderProposal } from "./agentChat";

type JsonObject = Record<string, unknown>;

const SUPPORTED_CHANGE_TYPES: CanvasChangesetChangeType[] = [
  "ADD_NODE",
  "DELETE_NODE",
  "UPDATE_NODE",
  "ADD_EDGE",
  "DELETE_EDGE",
];

function isRecord(value: unknown): value is JsonObject {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function normalizeNode(value: unknown): CanvasChangesetChangeNode | null {
  if (!isRecord(value)) {
    return null;
  }

  const id = typeof value.id === "string" ? value.id : "";
  if (!id) {
    return null;
  }

  const node: CanvasChangesetChangeNode = {
    id,
  };

  if (typeof value.name === "string" && value.name.length > 0) {
    node.name = value.name;
  }
  if (typeof value.block === "string" && value.block.length > 0) {
    node.block = value.block;
  }
  if (isRecord(value.configuration)) {
    node.configuration = value.configuration;
  }

  return node;
}

function normalizeEdge(value: unknown): CanvasChangesetChangeEdge | null {
  if (!isRecord(value)) {
    return null;
  }

  const sourceId = typeof value.sourceId === "string" ? value.sourceId : "";
  const targetId = typeof value.targetId === "string" ? value.targetId : "";
  if (!sourceId || !targetId) {
    return null;
  }

  const edge: CanvasChangesetChangeEdge = {
    sourceId,
    targetId,
  };
  if (typeof value.channel === "string" && value.channel.length > 0) {
    edge.channel = value.channel;
  }

  return edge;
}

function normalizeChange(value: unknown): CanvasChangesetChange | null {
  if (!isRecord(value) || typeof value.type !== "string") {
    return null;
  }

  const type = value.type as CanvasChangesetChangeType;
  if (!SUPPORTED_CHANGE_TYPES.includes(type)) {
    return null;
  }

  if (type === "ADD_NODE" || type === "DELETE_NODE" || type === "UPDATE_NODE") {
    const node = normalizeNode(value.node);
    if (!node) {
      return null;
    }
    if (type === "ADD_NODE" && !node.block) {
      return null;
    }

    return { type, node };
  }

  const edge = normalizeEdge(value.edge);
  if (!edge) {
    return null;
  }

  return { type, edge };
}

function normalizeChangeset(value: unknown): CanvasesCanvasChangeset | null {
  if (!isRecord(value)) {
    return null;
  }

  const changes = Array.isArray(value.changes)
    ? value.changes
        .map((change) => normalizeChange(change))
        .filter((change): change is CanvasChangesetChange => Boolean(change))
    : [];

  if (changes.length === 0) {
    return null;
  }

  return { changes };
}

export function normalizeAiProposal(value: unknown): AiBuilderProposal | null {
  if (!isRecord(value)) {
    return null;
  }

  const summary = typeof value.summary === "string" ? value.summary.trim() : "";
  if (!summary) {
    return null;
  }

  const changeset = normalizeChangeset(value.changeset);
  if (!changeset) {
    return null;
  }

  return {
    id: `proposal-${Date.now()}`,
    summary,
    changeset,
  };
}
