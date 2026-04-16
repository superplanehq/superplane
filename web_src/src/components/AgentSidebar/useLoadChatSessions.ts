import type { Dispatch, SetStateAction } from "react";
import { useEffect } from "react";
import type { AiBuilderMessage, AiChatSession } from "./agentChat";
import { loadChatSessions } from "./agentChat";

export type UseLoadChatSessionsParams = {
  canvasId?: string;
  organizationId?: string;
  setChatSessions: Dispatch<SetStateAction<AiChatSession[]>>;
  setCurrentChatId: Dispatch<SetStateAction<string | null>>;
  setAiMessages: Dispatch<SetStateAction<AiBuilderMessage[]>>;
  setIsLoadingChatSessions: Dispatch<SetStateAction<boolean>>;
};

export function useLoadChatSessions({
  canvasId,
  organizationId,
  setChatSessions,
  setCurrentChatId,
  setAiMessages,
  setIsLoadingChatSessions,
}: UseLoadChatSessionsParams): void {
  useEffect(() => {
    let cancelled = false;

    if (!canvasId || !organizationId) {
      setChatSessions([]);
      setCurrentChatId(null);
      setAiMessages([]);
      return () => {
        cancelled = true;
      };
    }

    void (async () => {
      setIsLoadingChatSessions(true);
      try {
        const sessions = await loadChatSessions({
          canvasId,
          organizationId,
        });
        if (cancelled) {
          return;
        }

        setChatSessions(sessions);
        setCurrentChatId((previousChatId) => {
          if (previousChatId && sessions.some((session) => session.id === previousChatId)) {
            return previousChatId;
          }

          return null;
        });
      } catch (error) {
        if (!cancelled) {
          console.warn("Failed to load chat sessions:", error);
        }
      } finally {
        if (!cancelled) {
          setIsLoadingChatSessions(false);
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [canvasId, organizationId, setAiMessages, setChatSessions, setCurrentChatId, setIsLoadingChatSessions]);
}
