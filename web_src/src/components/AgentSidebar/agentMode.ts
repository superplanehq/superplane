export type AgentMode = "builder" | "operator" | "architect";

const CANVAS_AGENT_MODE_STORAGE_KEY = "canvasAgentMode";

export function readInitialAgentMode(): AgentMode {
  if (typeof window === "undefined") return "operator";
  try {
    const stored = window.localStorage.getItem(CANVAS_AGENT_MODE_STORAGE_KEY);
    if (stored === "builder" || stored === "architect") return stored;
    return "operator";
  } catch {
    return "operator";
  }
}

export function persistAgentMode(mode: AgentMode): void {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(CANVAS_AGENT_MODE_STORAGE_KEY, mode);
}
