import { useCallback, useEffect, useRef } from "react";
import type { AgentOutgoingImage } from "@/components/CanvasToolSidebar/types";
import { consumeAgentComposerSend, peekAgentComposerSend, subscribeAgentComposerSend } from "./composerPrefill";

/**
 * Flushes queued suggestion/prefill sends into the composer one at a time.
 * Keeps the queue while another mutation is in flight so install boot kickoff
 * or a prior suggestion send cannot drop later prompts.
 * Failed sends are dropped (not requeued) to avoid a tight retry loop when
 * sendPending clears after rejection.
 */
export function useFlushAgentComposerSend(
  canvasId: string,
  onSend: (content: string, images: AgentOutgoingImage[]) => Promise<void>,
  sendPending: boolean,
) {
  const sendInFlightRef = useRef(false);

  const flushPending = useCallback(() => {
    if (!canvasId || sendPending || sendInFlightRef.current) return;
    const pending = peekAgentComposerSend(canvasId);
    if (!pending) return;
    // Clear before send so Strict Mode remounts do not double-send.
    consumeAgentComposerSend(canvasId);
    sendInFlightRef.current = true;
    void onSend(pending, [])
      .catch(() => {
        // Do not requeue: sendPending→false would immediately flush again forever.
      })
      .finally(() => {
        sendInFlightRef.current = false;
      });
  }, [canvasId, onSend, sendPending]);

  useEffect(() => {
    let cancelled = false;
    const flush = () => {
      if (cancelled) return;
      flushPending();
    };

    // Sidebar opens → chat query resolves → composer mounts. Flush now and once
    // more shortly after so a suggestion click is not lost to that race.
    flush();
    const retryId = window.setTimeout(flush, 300);
    const unsubscribe = subscribeAgentComposerSend(flush);
    return () => {
      cancelled = true;
      window.clearTimeout(retryId);
      unsubscribe();
    };
  }, [flushPending]);

  useEffect(() => {
    if (sendPending) return;
    flushPending();
  }, [sendPending, flushPending]);
}
