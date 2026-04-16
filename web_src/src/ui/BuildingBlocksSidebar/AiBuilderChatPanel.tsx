import { AiBuilderConversationMessageList } from "@/components/AiBuilderConversationMessageList";
import { ConversationList } from "@/components/AiBuilderConversationList";
import { InputForm } from "@/components/AiBuilderInputForm";
import { ProposalsList } from "@/components/AiBuilderProposalsList";
import {
  type AiBuilderMessage,
  type AiBuilderProposal,
  type AiChatSession,
} from "@/ui/BuildingBlocksSidebar/agentChat";
import { useEffect, useRef, type RefObject } from "react";

type AiBuilderChatPanelProps = {
  chatSessions: AiChatSession[];
  currentChatId: string | null;
  isLoadingChatSessions: boolean;
  isLoadingChatMessages: boolean;
  aiMessages: AiBuilderMessage[];
  isGeneratingResponse: boolean;
  pendingProposal: AiBuilderProposal | null;
  pendingProposalSummaries: string[];
  applyShortcutHint: string;
  onApplyProposal: () => void;
  onDiscardProposal: () => void;
  isApplyingProposal: boolean;
  aiError: string | null;
  disabled: boolean;
  canvasId?: string;
  aiInput: string;
  onAiInputChange: (value: string) => void;
  onSelectChat: (chatId: string) => void;
  onDeleteChat?: (chatId: string) => void;
  onStartNewSession: () => void;
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
  pendingProposalSummaries,
  applyShortcutHint,
  onApplyProposal,
  onDiscardProposal,
  isApplyingProposal,
  aiError,
  disabled,
  canvasId,
  aiInput,
  onAiInputChange,
  onSelectChat,
  onDeleteChat,
  onStartNewSession,
  onSendPrompt,
  aiInputRef,
}: AiBuilderChatPanelProps) {
  const aiMessagesContainerRef = useRef<HTMLDivElement>(null);
  const hasConversationState =
    aiMessages.length > 0 || isGeneratingResponse || pendingProposal !== null || aiError !== null;
  const isNewChatView = currentChatId === null && !hasConversationState;
  const showConversationList = currentChatId !== null;
  const maxAiInputHeight = isNewChatView ? 240 : 160;

  useEffect(() => {
    const container = aiMessagesContainerRef.current;
    if (!container) {
      return;
    }

    container.scrollTop = container.scrollHeight;
  }, [aiMessages, pendingProposal, isGeneratingResponse, aiError]);

  useEffect(() => {
    if (!aiInputRef.current) {
      return;
    }

    aiInputRef.current.style.height = "auto";
    aiInputRef.current.style.height = `${Math.min(aiInputRef.current.scrollHeight, maxAiInputHeight)}px`;
  }, [aiInput, aiInputRef, maxAiInputHeight]);

  return (
    <div className="mt-0 flex flex-1 min-h-0 flex-col overflow-hidden px-5 pb-5">
      <div className="h-full min-h-0 rounded-md bg-slate-50/30 flex flex-col">
        {isNewChatView ? (
          <>
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
              onDeleteChat={onDeleteChat}
              onStartNewSession={onStartNewSession}
              title="Previous chats"
              className="flex-1 min-h-0 px-2 py-2 space-y-2"
              fillAvailable
            />
          </>
        ) : (
          <>
            {showConversationList ? (
              <ConversationList
                chatSessions={chatSessions}
                currentChatId={currentChatId}
                isLoadingChatSessions={isLoadingChatSessions}
                isGeneratingResponse={isGeneratingResponse}
                onSelectChat={onSelectChat}
                onDeleteChat={onDeleteChat}
                onStartNewSession={onStartNewSession}
              />
            ) : null}

            <ConversationContent
              aiMessagesContainerRef={aiMessagesContainerRef}
              isLoadingChatMessages={isLoadingChatMessages}
              aiMessages={aiMessages}
              isGeneratingResponse={isGeneratingResponse}
              pendingProposal={pendingProposal}
              pendingProposalSummaries={pendingProposalSummaries}
              applyShortcutHint={applyShortcutHint}
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
          </>
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
  pendingProposalSummaries: string[];
  applyShortcutHint: string;
  onApplyProposal: () => void;
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
  pendingProposalSummaries,
  applyShortcutHint,
  onApplyProposal,
  onDiscardProposal,
  isApplyingProposal,
  aiError,
  disabled,
}: ConversationContentProps) {
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
          applyShortcutHint={applyShortcutHint}
          pendingProposalSummaries={pendingProposalSummaries}
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
