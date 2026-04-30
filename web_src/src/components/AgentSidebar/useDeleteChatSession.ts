import { showErrorToast, showSuccessToast } from "@/lib/toast";
import type { Dispatch, MutableRefObject, SetStateAction } from "react";
import { useCallback } from "react";
import type { AiBuilderMessage, AiBuilderProposal, AiChatSession } from "./agentChat";
import { deleteAgentChatSession, loadChatSessions } from "./agentChat";

export type UseDeleteChatSessionParams = {
  canvasId?: string;
  organizationId?: string;
  currentChatIdRef: MutableRefObject<string | null>;
  setChatSessions: Dispatch<SetStateAction<AiChatSession[]>>;
  setCurrentChatId: Dispatch<SetStateAction<string | null>>;
  setAiMessages: Dispatch<SetStateAction<AiBuilderMessage[]>>;
  setPendingProposal: Dispatch<SetStateAction<AiBuilderProposal | null>>;
  setAiError: Dispatch<SetStateAction<string | null>>;
};

export function useDeleteChatSession({
  canvasId,
  organizationId,
  currentChatIdRef,
  setChatSessions,
  setCurrentChatId,
  setAiMessages,
  setPendingProposal,
  setAiError,
}: UseDeleteChatSessionParams): (chatId: string) => void {
  return useCallback(
    (chatId: string) => {
      if (!canvasId || !organizationId) {
        return;
      }

      setChatSessions((previous) => previous.filter((s) => s.id !== chatId));
      if (currentChatIdRef.current === chatId) {
        setCurrentChatId(null);
        setAiMessages([]);
        setPendingProposal(null);
        setAiError(null);
      }

      void deleteAgentChatSession({ chatId, canvasId, organizationId }).then(
        () => showSuccessToast("Conversation deleted"),
        () => {
          showErrorToast("Failed to delete conversation");
          void loadChatSessions({ canvasId, organizationId }).then(
            (sessions) => setChatSessions(sessions),
            () => {},
          );
        },
      );
    },
    [
      canvasId,
      organizationId,
      currentChatIdRef,
      setAiError,
      setAiMessages,
      setChatSessions,
      setCurrentChatId,
      setPendingProposal,
    ],
  );
}
