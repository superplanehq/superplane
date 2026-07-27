/**
 * Window event + pending buffer so a suggestion can send even if the Agent
 * composer mounts after the sidebar opens.
 */
export const AGENT_SEND_COMPOSER_EVENT = "agent:send-composer";

let pendingSend: string | null = null;

export function requestAgentComposerSend(text: string) {
  const trimmed = text.trim();
  if (!trimmed) return;

  pendingSend = trimmed;
  if (typeof window === "undefined") return;
  window.dispatchEvent(new CustomEvent(AGENT_SEND_COMPOSER_EVENT, { detail: { text: trimmed } }));
}

/** Read pending text without clearing (composer may not be ready yet). */
export function peekAgentComposerSend(): string | null {
  return pendingSend;
}

/** Read and clear a send that was requested before the composer mounted. */
export function consumeAgentComposerSend(): string | null {
  const text = pendingSend;
  pendingSend = null;
  return text;
}

export function subscribeAgentComposerSend(onSend: (text: string) => void): () => void {
  if (typeof window === "undefined") return () => undefined;

  const listener = (event: Event) => {
    const text = (event as CustomEvent<{ text?: string }>).detail?.text;
    if (typeof text !== "string" || text.trim().length === 0) return;
    onSend(text);
  };

  window.addEventListener(AGENT_SEND_COMPOSER_EVENT, listener);
  return () => window.removeEventListener(AGENT_SEND_COMPOSER_EVENT, listener);
}
