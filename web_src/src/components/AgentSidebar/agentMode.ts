export type AgentMode = "builder" | "operator";

const CANVAS_AGENT_MODE_STORAGE_KEY = "canvasAgentMode";

export const VISIBLE_AGENT_MODES: AgentMode[] = ["operator", "builder"];
const DEFAULT_AGENT_MODE: AgentMode = "operator";

function isVisibleMode(value: unknown): value is AgentMode {
  return typeof value === "string" && (VISIBLE_AGENT_MODES as string[]).includes(value);
}

export function readInitialAgentMode(): AgentMode {
  if (typeof window === "undefined") return DEFAULT_AGENT_MODE;
  try {
    const stored = window.localStorage.getItem(CANVAS_AGENT_MODE_STORAGE_KEY);
    return isVisibleMode(stored) ? stored : DEFAULT_AGENT_MODE;
  } catch {
    return DEFAULT_AGENT_MODE;
  }
}

export function persistAgentMode(mode: AgentMode): void {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(CANVAS_AGENT_MODE_STORAGE_KEY, mode);
}
