/**
 * Window event + FIFO pending buffer so suggestions can send even if the Agent
 * composer mounts after the sidebar opens, including multiple rapid clicks.
 * Entries are canvas-scoped so SPA navigation cannot flush a prompt into the wrong chat.
 */
export const AGENT_SEND_COMPOSER_EVENT = "agent:send-composer";

type PendingSend = {
  canvasId: string;
  text: string;
};

const pendingSends: PendingSend[] = [];

export function requestAgentComposerSend(canvasId: string, text: string) {
  const trimmed = text.trim();
  if (!canvasId || !trimmed) return;

  pendingSends.push({ canvasId, text: trimmed });
  if (typeof window === "undefined") return;
  window.dispatchEvent(new CustomEvent(AGENT_SEND_COMPOSER_EVENT));
}

/** Read the next pending text for this canvas without clearing. */
export function peekAgentComposerSend(canvasId: string): string | null {
  if (!canvasId) return null;
  return pendingSends.find((item) => item.canvasId === canvasId)?.text ?? null;
}

/** Read and clear the next send for this canvas. */
export function consumeAgentComposerSend(canvasId: string): string | null {
  if (!canvasId) return null;
  const index = pendingSends.findIndex((item) => item.canvasId === canvasId);
  if (index < 0) return null;
  return pendingSends.splice(index, 1)[0]?.text ?? null;
}

/** Clear pending sends for one canvas, or the entire queue when omitted. */
export function clearAgentComposerSend(canvasId?: string) {
  if (!canvasId) {
    pendingSends.length = 0;
    return;
  }
  for (let index = pendingSends.length - 1; index >= 0; index -= 1) {
    if (pendingSends[index]?.canvasId === canvasId) {
      pendingSends.splice(index, 1);
    }
  }
}

export function subscribeAgentComposerSend(onFlush: () => void): () => void {
  if (typeof window === "undefined") return () => undefined;

  const listener = () => {
    onFlush();
  };

  window.addEventListener(AGENT_SEND_COMPOSER_EVENT, listener);
  return () => window.removeEventListener(AGENT_SEND_COMPOSER_EVENT, listener);
}
