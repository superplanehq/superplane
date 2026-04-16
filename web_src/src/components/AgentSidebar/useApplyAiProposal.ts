import type { CanvasChangesetChange } from "@/api-client";
import { useCallback } from "react";
import type { AiBuilderMessage, AiBuilderProposal } from "./agentChat";
import { pushAiMessages } from "./agentChat";

export type UseApplyAiProposalParams = {
  onApplyAiOperations?: (changes: CanvasChangesetChange[]) => Promise<void>;
  pendingProposal: AiBuilderProposal | null;
  setAiError: (error: string | null) => void;
  setIsApplyingProposal: (isApplying: boolean) => void;
  setAiMessages: (messages: AiBuilderMessage[]) => void;
  setPendingProposal: (proposal: AiBuilderProposal | null) => void;
};

export function useApplyAiProposal({
  onApplyAiOperations,
  pendingProposal,
  setAiError,
  setIsApplyingProposal,
  setAiMessages,
  setPendingProposal,
}: UseApplyAiProposalParams): () => Promise<void> {
  return useCallback(async () => {
    if (!pendingProposal) {
      return;
    }

    if (!onApplyAiOperations) {
      setAiError("Canvas apply handlers are not available.");
      return;
    }

    setAiError(null);
    setIsApplyingProposal(true);
    try {
      await onApplyAiOperations(pendingProposal.changeset.changes || []);
      setAiMessages((prev) =>
        pushAiMessages(prev, {
          id: `assistant-${Date.now()}`,
          role: "assistant",
          content: "Applied the proposed changes to the canvas.",
        }),
      );
      setPendingProposal(null);
    } catch (error) {
      setAiError(error instanceof Error ? error.message : "Failed to apply AI proposal.");
    } finally {
      setIsApplyingProposal(false);
    }
  }, [onApplyAiOperations, pendingProposal, setAiError, setAiMessages, setIsApplyingProposal, setPendingProposal]);
}
