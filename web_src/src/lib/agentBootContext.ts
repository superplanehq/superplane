export const AGENT_BOOT_CONTEXT_KEY = "agent-boot-context";
const AGENT_BOOT_INITIAL_MESSAGES_KEY = "agent-boot-initial-messages";
export const PLACEHOLDER_NODE_CONTEXT_KEY = "add-placeholder-node";
export const AGENT_BOOT_CONTEXT_READY_EVENT = "agent-boot-context-ready";

const DEFAULT_BOOT_MESSAGE =
  "Session ready. Read the current canvas state, check connected integrations, and greet the user.";

const BLANK_BOOT_MESSAGE =
  "The user just created a new blank app with a placeholder node on the canvas. Greet them briefly, then tell them to click on the 'New Component' node on the canvas and pick a component from the sidebar to get started. You can also ask what they want to build and help them choose the right component.";

const TEMPLATE_NEXT_STEP_MESSAGE = "What do you want to do next in the canvas?";

interface TemplateAgentBootContext {
  instructions?: string;
  initialMessage?: string;
}

interface AgentBootContext {
  canvasId: string;
  message: string;
  initialMessage?: string;
}

export function setAgentBootContext(canvasId: string, message: string | TemplateAgentBootContext) {
  if (typeof message !== "string" && message.initialMessage) {
    setAgentBootInitialMessage(canvasId, message.initialMessage);
  }

  const context = {
    canvasId,
    message: typeof message === "string" ? message : buildTemplateBootMessage(message),
  };

  sessionStorage.setItem(AGENT_BOOT_CONTEXT_KEY, JSON.stringify(context));
}

export function getAgentBootMessage(canvasId: string): string {
  if (typeof window === "undefined") return DEFAULT_BOOT_MESSAGE;
  const raw = sessionStorage.getItem(AGENT_BOOT_CONTEXT_KEY);
  if (!raw) return DEFAULT_BOOT_MESSAGE;

  try {
    const context = JSON.parse(raw) as AgentBootContext;
    if (context.canvasId !== canvasId) return DEFAULT_BOOT_MESSAGE;
    if (context.message === "blank") return BLANK_BOOT_MESSAGE;
    return context.message;
  } catch {
    return DEFAULT_BOOT_MESSAGE;
  }
}

export function getAgentBootInitialMessage(canvasId: string): string | null {
  if (typeof window === "undefined") return null;
  const raw = sessionStorage.getItem(AGENT_BOOT_CONTEXT_KEY);

  if (raw) {
    try {
      const context = JSON.parse(raw) as AgentBootContext;
      if (context.canvasId === canvasId && context.initialMessage) return context.initialMessage;
    } catch {
      return getStoredAgentBootInitialMessage(canvasId);
    }
  }

  return getStoredAgentBootInitialMessage(canvasId);
}

function buildTemplateBootMessage({ instructions, initialMessage }: TemplateAgentBootContext): string {
  if (!initialMessage) return instructions || DEFAULT_BOOT_MESSAGE;

  return [
    "The UI has already shown the user the template introduction.",
    "Do not inspect the canvas, integrations, files, or run any commands or tools.",
    `Reply only with: "${TEMPLATE_NEXT_STEP_MESSAGE}"`,
  ].join("\n\n");
}

function setAgentBootInitialMessage(canvasId: string, initialMessage: string) {
  const messages = getStoredAgentBootInitialMessages();
  messages[canvasId] = initialMessage;
  sessionStorage.setItem(AGENT_BOOT_INITIAL_MESSAGES_KEY, JSON.stringify(messages));
}

function getStoredAgentBootInitialMessage(canvasId: string): string | null {
  return getStoredAgentBootInitialMessages()[canvasId] ?? null;
}

function getStoredAgentBootInitialMessages(): Record<string, string> {
  const raw = sessionStorage.getItem(AGENT_BOOT_INITIAL_MESSAGES_KEY);
  if (!raw) return {};

  try {
    const parsed = JSON.parse(raw);
    return parsed && typeof parsed === "object" && !Array.isArray(parsed) ? (parsed as Record<string, string>) : {};
  } catch {
    return {};
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

export function abandonPendingPlaceholderBoot(canvasId: string) {
  if (typeof window === "undefined") return;
  if (sessionStorage.getItem(PLACEHOLDER_NODE_CONTEXT_KEY) === canvasId) {
    sessionStorage.removeItem(PLACEHOLDER_NODE_CONTEXT_KEY);
  }
  clearAgentBootContext(canvasId);
  window.dispatchEvent(new CustomEvent(AGENT_BOOT_CONTEXT_READY_EVENT, { detail: { canvasId } }));
}

export function clearAgentBootContext(canvasId?: string) {
  if (!canvasId) {
    sessionStorage.removeItem(AGENT_BOOT_CONTEXT_KEY);
    return;
  }

  const raw = sessionStorage.getItem(AGENT_BOOT_CONTEXT_KEY);
  if (!raw) return;

  try {
    const context = JSON.parse(raw) as { canvasId: string };
    if (context.canvasId !== canvasId) return;
  } catch {
    return;
  }

  sessionStorage.removeItem(AGENT_BOOT_CONTEXT_KEY);
}
