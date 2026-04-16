import type { Dispatch, SetStateAction } from "react";
import { useEffect } from "react";
import type { AiBuilderMessage, AiBuilderProposal } from "./agentChat";
import { loadChatConversation } from "./agentChat";

export type UseLoadChatConversationParams = {
  canvasId?: string;
  organizationId?: string;
  currentChatId: string | null;
  setAiMessages: Dispatch<SetStateAction<AiBuilderMessage[]>>;
  setPendingProposal: Dispatch<SetStateAction<AiBuilderProposal | null>>;
  setIsLoadingChatMessages: Dispatch<SetStateAction<boolean>>;
  setAiError: Dispatch<SetStateAction<string | null>>;
};

export function useLoadChatConversation({
  canvasId,
  organizationId,
  currentChatId,
  setAiMessages,
  setPendingProposal,
  setIsLoadingChatMessages,
  setAiError,
}: UseLoadChatConversationParams): void {
  useEffect(() => {
    let cancelled = false;

    if (!canvasId || !organizationId || !currentChatId) {
      if (!currentChatId) {
        setAiMessages([]);
        setPendingProposal(null);
      }
      setIsLoadingChatMessages(false);
      return () => {
        cancelled = true;
      };
    }

    void (async () => {
      setIsLoadingChatMessages(true);
      try {
        const messages = await loadChatConversation({
          chatId: currentChatId,
          canvasId,
          organizationId,
        });
        if (cancelled) {
          return;
        }

        setAiMessages(messages);
        setAiError(null);
      } catch (error) {
        if (!cancelled) {
          console.warn("Failed to load chat conversation:", error);
          setAiError(error instanceof Error ? error.message : "Failed to load chat conversation.");
        }
      } finally {
        if (!cancelled) {
          setIsLoadingChatMessages(false);
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [
    canvasId,
    currentChatId,
    organizationId,
    setAiError,
    setAiMessages,
    setIsLoadingChatMessages,
    setPendingProposal,
  ]);
}
