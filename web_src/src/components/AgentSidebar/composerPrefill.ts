/**
 * Window event + FIFO pending buffer so suggestions can send even if the Agent
 * composer mounts after the sidebar opens, including multiple rapid clicks.
 */
export const AGENT_SEND_COMPOSER_EVENT = "agent:send-composer";

const pendingSends: string[] = [];

export function requestAgentComposerSend(text: string) {
  const trimmed = text.trim();
  if (!trimmed) return;

  pendingSends.push(trimmed);
  if (typeof window === "undefined") return;
  window.dispatchEvent(new CustomEvent(AGENT_SEND_COMPOSER_EVENT));
}

/** Read the next pending text without clearing (composer may not be ready yet). */
export function peekAgentComposerSend(): string | null {
  return pendingSends[0] ?? null;
}

/** Read and clear the next send that was requested before the composer mounted. */
export function consumeAgentComposerSend(): string | null {
  return pendingSends.shift() ?? null;
}

/** Clear the entire pending queue (tests / abandoned flushes). */
export function clearAgentComposerSend() {
  pendingSends.length = 0;
}

export function subscribeAgentComposerSend(onFlush: () => void): () => void {
  if (typeof window === "undefined") return () => undefined;

  const listener = () => {
    onFlush();
  };

  window.addEventListener(AGENT_SEND_COMPOSER_EVENT, listener);
  return () => window.removeEventListener(AGENT_SEND_COMPOSER_EVENT, listener);
}
