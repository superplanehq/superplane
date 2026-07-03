export const AGENT_BOOT_CONTEXT_KEY = "agent-boot-context";
const AGENT_BOOT_INITIAL_MESSAGES_KEY = "agent-boot-initial-messages";
export const PLACEHOLDER_NODE_CONTEXT_KEY = "add-placeholder-node";
export const AGENT_BOOT_CONTEXT_READY_EVENT = "agent-boot-context-ready";

const BLANK_INITIAL_MESSAGE =
  "You can describe the workflow you want to build, or click on the 'New Component' node on the canvas to get started. I'm here to help!";

const TEMPLATE_NEXT_STEP_MESSAGE = "Tell me what you would like to do next in the canvas.";

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
  if (message === "blank") {
    setAgentBootInitialMessage(canvasId, BLANK_INITIAL_MESSAGE);
  }

  if (typeof message !== "string" && message.initialMessage) {
    setAgentBootInitialMessage(canvasId, message.initialMessage);
  }

  const context = {
    canvasId,
    message: typeof message === "string" ? message : buildTemplateBootMessage(message),
  };

  sessionStorage.setItem(AGENT_BOOT_CONTEXT_KEY, JSON.stringify(context));
}

// Returns the message to auto-send to the agent on canvas boot, or "" to send nothing.
// Opening or refreshing a canvas must never invoke the agent: without an explicit boot
// context (a template install) we return "", so the agent stays idle and spends no tokens
// until the user asks for something. A static greeting (rendered client-side in
// AgentTabPanel) welcomes the user instead.
export function getAgentBootMessage(canvasId: string): string {
  if (typeof window === "undefined") return "";
  const raw = sessionStorage.getItem(AGENT_BOOT_CONTEXT_KEY);
  if (!raw) return "";

  try {
    const context = JSON.parse(raw) as AgentBootContext;
    if (context.canvasId !== canvasId) return "";
    if (context.message === "blank") return ""; // Blank canvas — static greeting only.
    return context.message;
  } catch {
    return "";
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
  if (!initialMessage) return instructions ?? "";

  return [
    "The UI has already shown the user the template introduction.",
    "Do not run commands or tools to inspect the canvas, integrations, or files.",
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

// Clear every trace of boot state for a single canvas: the (shared) boot
// context entry if it belongs to this canvas, plus the persisted template
// intro. Used by /clear so a reset session doesn't auto-boot or keep showing
// the old template introduction, while leaving other canvases untouched.
export function clearAgentBootContextForCanvas(canvasId: string) {
  if (typeof window === "undefined") return;

  clearAgentBootContext(canvasId);

  const messages = getStoredAgentBootInitialMessages();
  if (canvasId in messages) {
    delete messages[canvasId];
    sessionStorage.setItem(AGENT_BOOT_INITIAL_MESSAGES_KEY, JSON.stringify(messages));
  }
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
