import type { Dispatch, MutableRefObject, SetStateAction } from "react";
import { useEffect } from "react";
import type { AiBuilderMessage, AiBuilderProposal, AiChatSession } from "./agentChat";
import { loadChatConversation, loadChatSessions } from "./agentChat";

const POLL_INTERVAL_MS = 2000;

export type UsePollForMessagesParams = {
  canvasId?: string;
  organizationId?: string;
  currentChatId: string | null;
  isGeneratingResponse: boolean;
  isStreamingRef: MutableRefObject<boolean>;
  setChatSessions: Dispatch<SetStateAction<AiChatSession[]>>;
  setAiMessages: Dispatch<SetStateAction<AiBuilderMessage[]>>;
  setIsGeneratingResponse: Dispatch<SetStateAction<boolean>>;
  setPendingProposal: Dispatch<SetStateAction<AiBuilderProposal | null>>;
};

export function usePollForMessages({
  canvasId,
  organizationId,
  currentChatId,
  isGeneratingResponse,
  isStreamingRef,
  setChatSessions,
  setAiMessages,
  setIsGeneratingResponse,
  setPendingProposal,
}: UsePollForMessagesParams): void {
  useEffect(() => {
    // Only poll when the spinner is showing and there's no active SSE stream.
    if (!isGeneratingResponse || !currentChatId || !canvasId || !organizationId) {
      return;
    }

    let cancelled = false;
    let timerId: number | null = null;

    const poll = async () => {
      if (cancelled) {
        return;
      }

      // Skip a tick if the SSE stream is actively delivering events.
      if (isStreamingRef.current) {
        schedule();
        return;
      }

      // Skip a tick if the tab is hidden — no point burning requests nobody sees.
      if (typeof document !== "undefined" && document.hidden) {
        schedule();
        return;
      }

      try {
        const [sessions, { messages, pendingProposal: polledProposal }] = await Promise.all([
          loadChatSessions({ canvasId, organizationId }),
          loadChatConversation({ chatId: currentChatId, canvasId, organizationId }),
        ]);

        if (cancelled) {
          return;
        }

        setChatSessions(sessions);
        setAiMessages(messages);

        const session = sessions.find((s) => s.id === currentChatId);
        if (session?.latestRunStatus !== "running") {
          setIsGeneratingResponse(false);
          setPendingProposal(polledProposal);
          return;
        }
      } catch (error) {
        console.warn("Polling for messages failed:", error);
      }

      schedule();
    };

    const schedule = () => {
      timerId = window.setTimeout(() => void poll(), POLL_INTERVAL_MS);
    };

    // When the user returns to the tab, fire immediately instead of waiting for
    // the next scheduled tick (which may have been skipped while hidden).
    const onVisibilityChange = () => {
      if (!document.hidden) {
        if (timerId !== null) {
          window.clearTimeout(timerId);
          timerId = null;
        }
        void poll();
      }
    };
    document.addEventListener("visibilitychange", onVisibilityChange);

    // useLoadChatConversation already fetched fresh messages and sessions when it
    // set isGeneratingResponse = true, so we don't need to poll immediately —
    // wait one interval to avoid a redundant double-fetch on chat open.
    schedule();

    return () => {
      cancelled = true;
      if (timerId !== null) {
        window.clearTimeout(timerId);
      }
      document.removeEventListener("visibilitychange", onVisibilityChange);
    };
  }, [
    isGeneratingResponse,
    currentChatId,
    canvasId,
    organizationId,
    // isStreamingRef is a stable object; included to satisfy exhaustive-deps.
    // .current is read synchronously inside poll() and never causes a re-run.
    isStreamingRef,
    setChatSessions,
    setAiMessages,
    setIsGeneratingResponse,
    setPendingProposal,
  ]);
}
