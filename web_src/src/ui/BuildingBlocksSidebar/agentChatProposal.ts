import type { AiBuilderProposal } from "./agentChat";

type JsonObject = Record<string, unknown>;

function isRecord(value: unknown): value is JsonObject {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function hasNonEmptyChangeset(value: unknown): value is { summary?: unknown; changeset: { changes: unknown[] } } {
  if (!isRecord(value) || !isRecord(value.changeset)) {
    return false;
  }

  return Array.isArray(value.changeset.changes) && value.changeset.changes.length > 0;
}

export function normalizeAiProposal(value: unknown): AiBuilderProposal | null {
  if (!hasNonEmptyChangeset(value)) {
    return null;
  }

  const summary = typeof value.summary === "string" ? value.summary.trim() : "";
  if (!summary) {
    return null;
  }

  return {
    id: `proposal-${Date.now()}`,
    summary,
    changeset: value.changeset as AiBuilderProposal["changeset"],
  };
}
