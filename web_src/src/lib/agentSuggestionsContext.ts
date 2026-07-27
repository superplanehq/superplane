import type { AgentSuggestion } from "@/ui/CanvasPage";

export const AGENT_SUGGESTIONS_STORAGE_KEY = "agent-suggestions";
const AGENT_SUGGESTIONS_DISMISSED_KEY = "agent-suggestions-dismissed";

type SuggestionsByCanvas = Record<string, AgentSuggestion[]>;
type DismissedByCanvas = Record<string, string[]>;

function isAgentSuggestion(value: unknown): value is AgentSuggestion {
  if (!value || typeof value !== "object") return false;
  const item = value as Partial<AgentSuggestion>;
  return typeof item.id === "string" && typeof item.label === "string" && typeof item.prompt === "string";
}

function readSuggestionsByCanvas(): SuggestionsByCanvas {
  if (typeof window === "undefined") return {};
  const raw = sessionStorage.getItem(AGENT_SUGGESTIONS_STORAGE_KEY);
  if (!raw) return {};

  try {
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) return {};

    const result: SuggestionsByCanvas = {};
    for (const [canvasId, suggestions] of Object.entries(parsed as Record<string, unknown>)) {
      if (!Array.isArray(suggestions)) continue;
      const valid = suggestions.filter(isAgentSuggestion);
      if (valid.length > 0) result[canvasId] = valid;
    }
    return result;
  } catch {
    return {};
  }
}

function writeSuggestionsByCanvas(value: SuggestionsByCanvas) {
  if (typeof window === "undefined") return;
  sessionStorage.setItem(AGENT_SUGGESTIONS_STORAGE_KEY, JSON.stringify(value));
}

function readDismissedByCanvas(): DismissedByCanvas {
  if (typeof window === "undefined") return {};
  try {
    const raw = localStorage.getItem(AGENT_SUGGESTIONS_DISMISSED_KEY);
    if (!raw) return {};
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) return {};

    const result: DismissedByCanvas = {};
    for (const [canvasId, ids] of Object.entries(parsed as Record<string, unknown>)) {
      if (!Array.isArray(ids)) continue;
      const valid = ids.filter((id): id is string => typeof id === "string");
      if (valid.length > 0) result[canvasId] = valid;
    }
    return result;
  } catch {
    return {};
  }
}

function writeDismissedByCanvas(value: DismissedByCanvas) {
  if (typeof window === "undefined") return;
  try {
    localStorage.setItem(AGENT_SUGGESTIONS_DISMISSED_KEY, JSON.stringify(value));
  } catch {
    // Ignore storage failures (e.g. private mode); dismissals are best-effort.
  }
}

export function setAgentSuggestions(canvasId: string, suggestions: AgentSuggestion[]) {
  if (typeof window === "undefined") return;
  const byCanvas = readSuggestionsByCanvas();
  if (suggestions.length === 0) {
    delete byCanvas[canvasId];
  } else {
    byCanvas[canvasId] = suggestions;
  }
  writeSuggestionsByCanvas(byCanvas);
}

export function getAgentSuggestions(canvasId: string): AgentSuggestion[] {
  if (typeof window === "undefined" || !canvasId) return [];
  return readSuggestionsByCanvas()[canvasId] ?? [];
}

export function clearAgentSuggestions(canvasId: string) {
  if (typeof window === "undefined") return;
  const byCanvas = readSuggestionsByCanvas();
  if (!(canvasId in byCanvas)) return;
  delete byCanvas[canvasId];
  writeSuggestionsByCanvas(byCanvas);
}

export function getDismissedAgentSuggestionIds(canvasId: string): ReadonlySet<string> {
  if (typeof window === "undefined" || !canvasId) return new Set();
  return new Set(readDismissedByCanvas()[canvasId] ?? []);
}

export function dismissAgentSuggestion(canvasId: string, suggestionId: string) {
  if (typeof window === "undefined" || !canvasId || !suggestionId) return;
  const byCanvas = readDismissedByCanvas();
  const existing = new Set(byCanvas[canvasId] ?? []);
  existing.add(suggestionId);
  byCanvas[canvasId] = Array.from(existing);
  writeDismissedByCanvas(byCanvas);
}
