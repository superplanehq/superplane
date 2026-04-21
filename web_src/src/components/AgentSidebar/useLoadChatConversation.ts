import type { Dispatch, SetStateAction } from "react";
import { useEffect, useState } from "react";
import type { AiBuilderMessage, AiBuilderProposal, AiChatSession } from "./agentChat";
import { loadChatConversation, loadChatSessions } from "./agentChat";

export type UseLoadChatConversationParams = {
  canvasId?: string;
  organizationId?: string;
  currentChatId: string | null;
  setChatSessions: Dispatch<SetStateAction<AiChatSession[]>>;
  setAiMessages: Dispatch<SetStateAction<AiBuilderMessage[]>>;
  setPendingProposal: Dispatch<SetStateAction<AiBuilderProposal | null>>;
  setAiError: Dispatch<SetStateAction<string | null>>;
  setIsGeneratingResponse: Dispatch<SetStateAction<boolean>>;
};

export function useLoadChatConversation({
  canvasId,
  organizationId,
  currentChatId,
  setChatSessions,
  setAiMessages,
  setPendingProposal,
  setAiError,
  setIsGeneratingResponse,
}: UseLoadChatConversationParams): boolean {
  const [isLoadingChatMessages, setIsLoadingChatMessages] = useState(false);

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
        // Fetch messages and sessions in parallel so we have authoritative, fresh
        // latestRunStatus at the exact moment we decide whether to show the spinner.
        const [{ messages, pendingProposal: loadedProposal }, freshSessions] = await Promise.all([
          loadChatConversation({ chatId: currentChatId, canvasId, organizationId }),
          loadChatSessions({ canvasId, organizationId }),
        ]);

        if (cancelled) {
          return;
        }

        setChatSessions(freshSessions);
        setAiMessages(messages);
        setAiError(null);

        // If the run is still in progress, show the spinner so the user knows the
        // agent is working and the polling loop can take over.
        // Don't restore a proposal yet — wait for the run to finish.
        const session = freshSessions.find((s) => s.id === currentChatId);
        if (session?.latestRunStatus === "running") {
          setIsGeneratingResponse(true);
        } else {
          setPendingProposal(loadedProposal);
        }
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
    setChatSessions,
    setAiError,
    setAiMessages,
    setIsGeneratingResponse,
    setIsLoadingChatMessages,
    setPendingProposal,
  ]);

  return isLoadingChatMessages;
}
