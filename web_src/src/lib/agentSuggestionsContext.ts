import type { AgentSuggestion } from "@/ui/CanvasPage";

export const AGENT_SUGGESTIONS_STORAGE_KEY = "agent-suggestions";

type SuggestionsByCanvas = Record<string, AgentSuggestion[]>;

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
