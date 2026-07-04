import { useCallback, useMemo, useRef } from "react";
import type { AgentMode } from "@/components/AgentSidebar/agentMode";
import type { RubricCategory } from "@/components/AgentSidebar/widgets/parser";
import type { OutcomeState } from "@/components/AgentSidebar/widgets/OutcomeProgressWidget";
import type {
  useDefineAgentOutcome,
  useInterruptAgentChat,
  useResetCanvasAgentChat,
  useSendAgentChatMessage,
} from "@/hooks/useAgentChats";
import { clearAgentBootContextForCanvas } from "@/lib/agentBootContext";
import { isSessionBusyError } from "./agentSetupStateModel";
import type { AgentOutgoingImage } from "./types";

export type ConversationHandlers = {
  handleSend: (content: string, images?: AgentOutgoingImage[]) => Promise<void>;
  handleStop: () => void;
  handleQuickAction: (action: string) => Promise<void>;
  handleStartBuilding: (rubric: { title: string; criteria: string[]; categories?: RubricCategory[] }) => Promise<void>;
};

export function useAgentConversationHandlers({
  agentMode,
  chatId,
  canvasId,
  isBusy,
  outcomeMutation,
  interruptMutation,
  resetMutation,
  sendMutation,
  setError,
  setNotice,
  setOutcomeState,
}: {
  agentMode: AgentMode;
  chatId: string;
  canvasId: string;
  isBusy: boolean;
  outcomeMutation: ReturnType<typeof useDefineAgentOutcome>;
  interruptMutation: ReturnType<typeof useInterruptAgentChat>;
  resetMutation: ReturnType<typeof useResetCanvasAgentChat>;
  sendMutation: ReturnType<typeof useSendAgentChatMessage>;
  setError: (value: string | null) => void;
  setNotice: (value: string | null) => void;
  setOutcomeState: (update: OutcomeState | null | ((prev: OutcomeState | null) => OutcomeState | null)) => void;
}): ConversationHandlers {
  const mutationsRef = useRef({ sendMutation, interruptMutation, outcomeMutation, resetMutation });
  mutationsRef.current = { sendMutation, interruptMutation, outcomeMutation, resetMutation };
  const isBusyRef = useRef(isBusy);
  isBusyRef.current = isBusy;

  const handleSend = useCallback(
    async (content: string, images?: AgentOutgoingImage[]) => {
      const trimmed = content.trim();
      const { sendMutation: send, resetMutation: reset } = mutationsRef.current;

      if (trimmed === "/clear") {
        if (reset.isPending) return;
        setError(null);
        setNotice(null);
        if (isBusyRef.current) {
          setNotice("Wait for the agent to finish before clearing the chat.");
          return;
        }
        if (send.isPending) {
          setNotice("Wait for the message to send before clearing the chat.");
          return;
        }
        await reset.mutateAsync().catch((error) => {
          if (isSessionBusyError(error)) {
            setNotice("Wait for the agent to finish before clearing the chat.");
          } else {
            setError(error instanceof Error ? error.message : "failed to clear chat");
          }
          throw error;
        });
        clearAgentBootContextForCanvas(canvasId);
        setOutcomeState(null);
        setNotice("Chat cleared. You’re in a fresh session.");
        return;
      }

      if ((!trimmed && (images?.length ?? 0) === 0) || send.isPending || reset.isPending) return;
      setError(null);
      setNotice(null);

      await send.mutateAsync({ chatId, content, mode: agentMode, images }).catch((error) => {
        setError(error instanceof Error ? error.message : "failed to send message");
        throw error;
      });
    },
    [agentMode, chatId, canvasId, setError, setNotice, setOutcomeState],
  );

  const handleStop = useCallback(() => {
    mutationsRef.current.interruptMutation.mutate({ chatId });
  }, [chatId]);

  const handleQuickAction = useCallback(
    async (action: string) => {
      const { sendMutation: send, resetMutation: reset } = mutationsRef.current;
      if (send.isPending || reset.isPending) return;
      try {
        await send.mutateAsync({ chatId, content: action, mode: agentMode });
      } catch {
        // Keep the current transcript unchanged when quick actions fail.
      }
    },
    [agentMode, chatId],
  );

  const handleStartBuilding = useCallback(
    async (_rubric: { title: string; criteria: string[]; categories?: RubricCategory[] }) => {
      const { sendMutation: send, resetMutation: reset } = mutationsRef.current;
      if (reset.isPending) return;

      try {
        await send.mutateAsync({
          chatId,
          content: "Specs approved. Start building.",
          mode: "builder",
        });
      } catch {
        setError("Failed to start building. Please try again.");
      }
    },
    [chatId, setError],
  );

  return useMemo(
    () => ({ handleSend, handleStop, handleQuickAction, handleStartBuilding }),
    [handleSend, handleStop, handleQuickAction, handleStartBuilding],
  );
}
