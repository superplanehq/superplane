import type { Dispatch, SetStateAction } from "react";
import { useEffect, useRef, type RefObject } from "react";
import { ConversationList } from "./AiBuilderConversationList";
import { AiBuilderConversationMessageList } from "./AiBuilderConversationMessageList";
import { InputForm } from "./AiBuilderInputForm";
import { ProposalsList } from "./AiBuilderProposalsList";
import { type AiBuilderMessage, type AiBuilderProposal, type AiChatSession } from "./agentChat";
import { useApplyOnCmdEnter } from "./useApplyOnCmdEnter";
import { useDeleteChatSession } from "./useDeleteChatSession";

type AiBuilderChatPanelProps = {
  chatSessions: AiChatSession[];
  currentChatId: string | null;
  isLoadingChatSessions: boolean;
  isLoadingChatMessages: boolean;
  aiMessages: AiBuilderMessage[];
  isGeneratingResponse: boolean;
  pendingProposal: AiBuilderProposal | null;
  onApplyProposal: () => void | Promise<void>;
  onDiscardProposal: () => void;
  isApplyingProposal: boolean;
  aiError: string | null;
  disabled: boolean;
  canvasId?: string;
  organizationId?: string;
  setChatSessions: Dispatch<SetStateAction<AiChatSession[]>>;
  setCurrentChatId: Dispatch<SetStateAction<string | null>>;
  setAiMessages: Dispatch<SetStateAction<AiBuilderMessage[]>>;
  setPendingProposal: Dispatch<SetStateAction<AiBuilderProposal | null>>;
  setAiError: Dispatch<SetStateAction<string | null>>;
  aiInput: string;
  onAiInputChange: (value: string) => void;
  onSelectChat: (chatId: string) => void;
  onSendPrompt: () => void;
  aiInputRef: RefObject<HTMLTextAreaElement | null>;
};

export function AiBuilderChatPanel({
  chatSessions,
  currentChatId,
  isLoadingChatSessions,
  isLoadingChatMessages,
  aiMessages,
  isGeneratingResponse,
  pendingProposal,
  onApplyProposal,
  onDiscardProposal,
  isApplyingProposal,
  aiError,
  disabled,
  canvasId,
  organizationId,
  setChatSessions,
  setCurrentChatId,
  setAiMessages,
  setPendingProposal,
  setAiError,
  aiInput,
  onAiInputChange,
  onSelectChat,
  onSendPrompt,
  aiInputRef,
}: AiBuilderChatPanelProps) {
  const currentChatIdRef = useRef(currentChatId);
  currentChatIdRef.current = currentChatId;

  const handleDeleteChatSession = useDeleteChatSession({
    canvasId,
    organizationId,
    currentChatIdRef,
    setChatSessions,
    setCurrentChatId,
    setAiMessages,
    setPendingProposal,
    setAiError,
  });

  const aiMessagesContainerRef = useRef<HTMLDivElement>(null);
  const hasConversationState =
    aiMessages.length > 0 || isGeneratingResponse || pendingProposal !== null || aiError !== null;
  const isNewChatView = currentChatId === null && !hasConversationState;
  const maxAiInputHeight = isNewChatView ? 240 : 160;

  useEffect(() => {
    const container = aiMessagesContainerRef.current;
    if (!container) {
      return;
    }

    container.scrollTop = container.scrollHeight;
  }, [aiMessages, pendingProposal, isGeneratingResponse, aiError]);

  useAutoInputHeight(aiInputRef, maxAiInputHeight, aiInput);

  return (
    <div className="flex flex-1 min-h-0 flex-col overflow-hidden">
      <div className="h-full min-h-0 flex flex-col">
        {isNewChatView ? (
          <div className="m-3">
            <InputForm
              aiInputRef={aiInputRef}
              aiInput={aiInput}
              onAiInputChange={onAiInputChange}
              onSendPrompt={onSendPrompt}
              disabled={disabled}
              canvasId={canvasId}
              isGeneratingResponse={isGeneratingResponse}
              maxAiInputHeight={maxAiInputHeight}
              expanded
            />

            <ConversationList
              chatSessions={chatSessions}
              currentChatId={currentChatId}
              isLoadingChatSessions={isLoadingChatSessions}
              isGeneratingResponse={isGeneratingResponse}
              onSelectChat={onSelectChat}
              onDeleteChat={handleDeleteChatSession}
              className="flex-1 min-h-0 px-2 py-2 space-y-2"
              fillAvailable
            />
          </div>
        ) : (
          <div className="mx-2 mb-2 h-full flex flex-col">
            <ConversationContent
              aiMessagesContainerRef={aiMessagesContainerRef}
              isLoadingChatMessages={isLoadingChatMessages}
              aiMessages={aiMessages}
              isGeneratingResponse={isGeneratingResponse}
              pendingProposal={pendingProposal}
              onApplyProposal={onApplyProposal}
              onDiscardProposal={onDiscardProposal}
              isApplyingProposal={isApplyingProposal}
              aiError={aiError}
              disabled={disabled}
            />

            <InputForm
              aiInputRef={aiInputRef}
              aiInput={aiInput}
              onAiInputChange={onAiInputChange}
              onSendPrompt={onSendPrompt}
              disabled={disabled}
              canvasId={canvasId}
              isGeneratingResponse={isGeneratingResponse}
              maxAiInputHeight={maxAiInputHeight}
            />
          </div>
        )}
      </div>
    </div>
  );
}

type ConversationContentProps = {
  aiMessagesContainerRef: RefObject<HTMLDivElement | null>;
  isLoadingChatMessages: boolean;
  aiMessages: AiBuilderMessage[];
  isGeneratingResponse: boolean;
  pendingProposal: AiBuilderProposal | null;
  onApplyProposal: () => void | Promise<void>;
  onDiscardProposal: () => void;
  isApplyingProposal: boolean;
  aiError: string | null;
  disabled: boolean;
};

function ConversationContent({
  aiMessagesContainerRef,
  isLoadingChatMessages,
  aiMessages,
  isGeneratingResponse,
  pendingProposal,
  onApplyProposal,
  onDiscardProposal,
  isApplyingProposal,
  aiError,
  disabled,
}: ConversationContentProps) {
  const canApplyProposalWithShortcut = !!pendingProposal && (pendingProposal.changeset.changes || []).length > 0;

  useApplyOnCmdEnter({
    enabled: canApplyProposalWithShortcut,
    disabled,
    isApplying: isApplyingProposal,
    onApply: onApplyProposal,
  });

  return (
    <div ref={aiMessagesContainerRef} className="flex-1 overflow-y-auto space-y-1 px-2 py-3">
      {isLoadingChatMessages ? <div className="text-xs text-gray-500 px-1 py-1">Loading conversation...</div> : null}
      <AiBuilderConversationMessageList messages={aiMessages} isGeneratingResponse={isGeneratingResponse} />

      {isGeneratingResponse ? (
        <div className="sp-ai-thinking text-xs text-gray-500 px-1 py-1 rounded-sm">Planning next steps...</div>
      ) : null}

      {pendingProposal ? (
        <ProposalsList
          disabled={disabled}
          pendingProposal={pendingProposal}
          onApplyProposal={onApplyProposal}
          onDiscardProposal={onDiscardProposal}
          isApplyingProposal={isApplyingProposal}
          aiError={aiError}
        />
      ) : null}

      {!pendingProposal && aiError ? <p className="text-xs text-red-700 px-1">{aiError}</p> : null}
    </div>
  );
}

function useAutoInputHeight(
  aiInputRef: RefObject<HTMLTextAreaElement | null>,
  maxAiInputHeight: number,
  aiInput: string,
): void {
  useEffect(() => {
    const textarea = aiInputRef.current;
    if (!textarea) {
      return;
    }

    textarea.style.height = "auto";
    textarea.style.height = `${Math.min(textarea.scrollHeight, maxAiInputHeight)}px`;
  }, [aiInput, aiInputRef, maxAiInputHeight]);
}
