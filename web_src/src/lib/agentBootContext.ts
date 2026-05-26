export const AGENT_BOOT_CONTEXT_KEY = "agent-boot-context";
export const PLACEHOLDER_NODE_CONTEXT_KEY = "add-placeholder-node";
export const AGENT_BOOT_CONTEXT_READY_EVENT = "agent-boot-context-ready";

const DEFAULT_BOOT_MESSAGE =
  "Session ready. Read the current canvas state, check connected integrations, and greet the user.";

const BLANK_BOOT_MESSAGE =
  "The user just created a new blank app with a placeholder node on the canvas. Greet them briefly, then tell them to click on the 'New Component' node on the canvas and pick a component from the sidebar to get started. You can also ask what they want to build and help them choose the right component.";

export function setAgentBootContext(canvasId: string, message: string) {
  sessionStorage.setItem(AGENT_BOOT_CONTEXT_KEY, JSON.stringify({ canvasId, message }));
}

export function getAgentBootMessage(canvasId: string): string {
  if (typeof window === "undefined") return DEFAULT_BOOT_MESSAGE;
  const raw = sessionStorage.getItem(AGENT_BOOT_CONTEXT_KEY);
  if (!raw) return DEFAULT_BOOT_MESSAGE;

  try {
    const context = JSON.parse(raw) as { canvasId: string; message: string };
    if (context.canvasId !== canvasId) return DEFAULT_BOOT_MESSAGE;
    if (context.message === "blank") return BLANK_BOOT_MESSAGE;
    return context.message;
  } catch {
    return DEFAULT_BOOT_MESSAGE;
  }
}

export function isAgentBootReady(canvasId: string): boolean {
  if (typeof window === "undefined") return true;
  return sessionStorage.getItem(PLACEHOLDER_NODE_CONTEXT_KEY) !== canvasId;
}

export function markAgentBootReady(canvasId: string) {
  if (typeof window === "undefined") return;
  if (sessionStorage.getItem(PLACEHOLDER_NODE_CONTEXT_KEY) === canvasId) {
    sessionStorage.removeItem(PLACEHOLDER_NODE_CONTEXT_KEY);
  }
  window.dispatchEvent(new CustomEvent(AGENT_BOOT_CONTEXT_READY_EVENT, { detail: { canvasId } }));
}

export function clearAgentBootContext() {
  sessionStorage.removeItem(AGENT_BOOT_CONTEXT_KEY);
}
