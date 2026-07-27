import { useCallback, useEffect } from "react";
import type { AgentOutgoingImage } from "@/components/CanvasToolSidebar/types";
import {
  consumeAgentComposerSend,
  peekAgentComposerSend,
  requeueAgentComposerSend,
  subscribeAgentComposerSend,
} from "./composerPrefill";

/**
 * Flushes queued suggestion/prefill sends into the composer one at a time.
 * Keeps the queue while another mutation is in flight so install boot kickoff
 * or a prior suggestion send cannot drop later prompts.
 */
export function useFlushAgentComposerSend(
  onSend: (content: string, images: AgentOutgoingImage[]) => Promise<void>,
  sendPending: boolean,
) {
  const flushPending = useCallback(() => {
    if (sendPending) return;
    const pending = peekAgentComposerSend();
    if (!pending) return;
    // Clear before send so Strict Mode remounts do not double-send.
    consumeAgentComposerSend();
    void onSend(pending, []).catch(() => {
      // Restore the prompt so a failed agent send can be retried on the next flush.
      requeueAgentComposerSend(pending);
    });
  }, [onSend, sendPending]);

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
