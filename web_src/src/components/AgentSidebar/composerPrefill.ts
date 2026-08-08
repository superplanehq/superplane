import type { MentionItem } from "./useMentions";

/**
 * Window event + FIFO pending buffer so suggestions can send even if the Agent
 * composer mounts after the sidebar opens, including multiple rapid clicks.
 * Entries are canvas-scoped so SPA navigation cannot flush a prompt into the wrong chat.
 */
export const AGENT_SEND_COMPOSER_EVENT = "agent:send-composer";
export const AGENT_PREFILL_COMPOSER_EVENT = "agent:prefill-composer";

type PendingSend = {
  canvasId: string;
  text: string;
};

const pendingSends: PendingSend[] = [];

export type AgentComposerPrefill = {
  text: string;
  mentions: MentionItem[];
};

type PendingPrefill = AgentComposerPrefill & { canvasId: string; requestedAt: number };

const pendingPrefills: PendingPrefill[] = [];
const PREFILL_TTL_MS = 10_000;

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

export function requestAgentComposerPrefill(canvasId: string, prefill: AgentComposerPrefill) {
  const text = prefill.text.trim();
  if (!canvasId || !text) return;

  pendingPrefills.push({ canvasId, text, mentions: prefill.mentions, requestedAt: Date.now() });
  if (typeof window === "undefined") return;
  window.dispatchEvent(new CustomEvent(AGENT_PREFILL_COMPOSER_EVENT));
}

export function consumeAgentComposerPrefill(canvasId: string): AgentComposerPrefill | null {
  if (!canvasId) return null;
  while (true) {
    const index = pendingPrefills.findIndex((item) => item.canvasId === canvasId);
    if (index < 0) return null;
    const pending = pendingPrefills.splice(index, 1)[0];
    if (pending && Date.now() - pending.requestedAt <= PREFILL_TTL_MS) {
      return { text: pending.text, mentions: pending.mentions };
    }
  }
}

export function clearAgentComposerPrefill(canvasId?: string) {
  if (!canvasId) {
    pendingPrefills.length = 0;
    return;
  }
  for (let index = pendingPrefills.length - 1; index >= 0; index -= 1) {
    if (pendingPrefills[index]?.canvasId === canvasId) pendingPrefills.splice(index, 1);
  }
}

export function subscribeAgentComposerPrefill(onPrefill: () => void): () => void {
  if (typeof window === "undefined") return () => undefined;

  window.addEventListener(AGENT_PREFILL_COMPOSER_EVENT, onPrefill);
  return () => window.removeEventListener(AGENT_PREFILL_COMPOSER_EVENT, onPrefill);
}
