import { useCallback, useEffect } from "react";
import type { AgentOutgoingImage } from "@/components/CanvasToolSidebar/types";
import { consumeAgentComposerSend, peekAgentComposerSend, subscribeAgentComposerSend } from "./composerPrefill";

/**
 * Flushes suggestion/prefill sends into the composer. Keeps the pending buffer
 * while another mutation is in flight so install boot kickoff cannot drop it.
 */
export function useFlushAgentComposerSend(
  onSend: (content: string, images: AgentOutgoingImage[]) => Promise<void>,
  sendPending: boolean,
) {
  const sendExternalText = useCallback(
    (text: string) => {
      const trimmed = text.trim();
      if (!trimmed) return;
      if (sendPending) return;
      // Clear before send so Strict Mode remounts do not double-send.
      if (peekAgentComposerSend() === trimmed) {
        consumeAgentComposerSend();
      }
      void onSend(trimmed, []);
    },
    [onSend, sendPending],
  );

  useEffect(() => {
    let cancelled = false;
    const flushPending = () => {
      if (cancelled) return;
      const pending = peekAgentComposerSend();
      if (pending) sendExternalText(pending);
    };

    // Sidebar opens → chat query resolves → composer mounts. Flush now and once
    // more shortly after so a suggestion click is not lost to that race.
    flushPending();
    const retryId = window.setTimeout(flushPending, 300);
    const unsubscribe = subscribeAgentComposerSend(sendExternalText);
    return () => {
      cancelled = true;
      window.clearTimeout(retryId);
      unsubscribe();
    };
  }, [sendExternalText]);

  useEffect(() => {
    if (sendPending) return;
    const pending = peekAgentComposerSend();
    if (pending) sendExternalText(pending);
  }, [sendPending, sendExternalText]);
}
