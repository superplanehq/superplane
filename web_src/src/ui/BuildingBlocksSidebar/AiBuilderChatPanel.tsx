import { TimeAgo } from "@/components/TimeAgo";
import { Button } from "@/components/ui/button";
import { TabsContent } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { Activity, ArrowLeft, ArrowUp, User } from "lucide-react";
import { useEffect, useRef } from "react";
import ReactMarkdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import type { AiBuilderMessage, AiBuilderProposal, AiChatSession } from "@/ui/BuildingBlocksSidebar/agentChat";
import { cn } from "../../lib/utils";

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
  onStartNewSession: () => void;
  onSendPrompt: () => void;
  aiInputRef: React.RefObject<HTMLTextAreaElement | null>;
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
    <TabsContent value="ai" className="mt-0 flex-1 overflow-hidden px-5 pb-5">
      <div className="h-full rounded-md bg-slate-50/30 flex flex-col">
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
    </TabsContent>
  );
}

type ConversationListProps = {
  chatSessions: AiChatSession[];
  currentChatId: string | null;
  isLoadingChatSessions: boolean;
  isGeneratingResponse: boolean;
  onSelectChat: (chatId: string) => void;
  onStartNewSession: () => void;
  title?: string;
  className?: string;
  fillAvailable?: boolean;
};

function ConversationList({
  chatSessions,
  currentChatId,
  isLoadingChatSessions,
  isGeneratingResponse,
  onSelectChat,
  onStartNewSession,
  title,
  className,
  fillAvailable = false,
}: ConversationListProps) {
  const currentSession = currentChatId ? chatSessions.find((s) => s.id === currentChatId) : undefined;
  const showCurrentSessionHeader = Boolean(currentChatId);

  const currentSessionHeader = () => {
    if (isLoadingChatSessions) {
      return <span className="text-xs text-slate-500">Loading…</span>;
    }

    if (currentSession) {
      return (
        <div
          className="flex min-w-0 flex-1 items-center justify-between gap-2"
          title={currentSession.createdAt ? formatSessionDate(currentSession.createdAt) : undefined}
        >
          <div className="min-w-0 truncate text-sm font-medium text-slate-800">{currentSession.title}</div>
          {currentSession.createdAt ? (
            <TimeAgo date={currentSession.createdAt} className="shrink-0 text-[11px] tabular-nums text-slate-500" />
          ) : null}
        </div>
      );
    }

    return <span className="text-sm text-slate-600">Conversation</span>;
  };

  return (
    <div
      className={cn("border-b border-border px-2 py-2 space-y-2", fillAvailable && "flex min-h-0 flex-col", className)}
    >
      <div className="flex min-w-0 items-center gap-2">
        {showCurrentSessionHeader ? (
          <>
            <Button
              size="icon-xs"
              variant="ghost"
              onClick={onStartNewSession}
              disabled={isGeneratingResponse}
              aria-label="Back to new chat"
              title="Back"
              className="shrink-0"
            >
              <ArrowLeft className="h-4 w-4" />
            </Button>
            {currentSessionHeader()}
          </>
        ) : (
          <p className="text-[11px] font-medium uppercase tracking-[0.08em] text-slate-500">
            {title ?? "Conversations"}
          </p>
        )}
      </div>

      {!currentChatId ? (
        <div
          className={cn(
            fillAvailable ? "min-h-0 flex-1 overflow-y-auto" : "max-h-28 overflow-y-auto",
            "bg-transparent",
            fillAvailable ? "space-y-2" : "space-y-1",
            "[scrollbar-width:thin] [scrollbar-color:rgb(203_213_225)_transparent]",
            "[&::-webkit-scrollbar]:w-1.5 [&::-webkit-scrollbar-thumb]:rounded-full [&::-webkit-scrollbar-thumb]:bg-slate-300/70 [&::-webkit-scrollbar-track]:bg-transparent",
          )}
        >
          {isLoadingChatSessions ? (
            <div className="text-xs text-gray-500 px-2 py-2">Loading conversations...</div>
          ) : null}
          {!isLoadingChatSessions && chatSessions.length === 0 ? (
            <div className="text-xs text-gray-500 px-2 py-2">No conversations yet.</div>
          ) : null}

          {chatSessions.map((session) => {
            return (
              <button
                key={session.id}
                type="button"
                onClick={() => onSelectChat(session.id)}
                disabled={isGeneratingResponse}
                title={session.createdAt ? formatSessionDate(session.createdAt) : undefined}
                className="w-full rounded-md border border-slate-200 bg-white px-2 py-2 text-left text-slate-700 transition-colors hover:bg-slate-50"
              >
                <div className="flex min-w-0 items-center justify-between gap-2">
                  <div className="min-w-0 truncate text-sm font-medium">{session.title}</div>
                  {session.createdAt ? (
                    <TimeAgo date={session.createdAt} className="shrink-0 text-[11px] tabular-nums text-slate-500" />
                  ) : null}
                </div>
              </button>
            );
          })}
        </div>
      ) : null}
    </div>
  );
}

type ConversationContentProps = {
  aiMessagesContainerRef: React.RefObject<HTMLDivElement | null>;
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
      <AiMessages messages={aiMessages} />

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

function AiMessages({ messages }: { messages: AiBuilderMessage[] }) {
  return (
    <>
      {messages.map((message) => (
        <AiMessage key={message.id} message={message} />
      ))}
    </>
  );
}

function formatSessionDate(value: string): string {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }

  return parsed.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

function AiMessage({ message }: { message: AiBuilderMessage }) {
  const isEmptyAssistantPlaceholder = message.role === "assistant" && message.content.trim().length === 0;
  if (isEmptyAssistantPlaceholder) {
    return null;
  }

  const isToolMessage = message.role === "tool";
  const isRunningToolMessage = isToolMessage && message.toolStatus === "running";

  let messageClassName = "";
  let wrapperClassName = "w-full";

  if (message.role === "user") {
    messageClassName =
      "flex w-full items-start gap-2 rounded-md border border-slate-200/90 bg-slate-100 px-3 py-2.5 text-sm text-slate-800";
    wrapperClassName = "w-full py-1";
  } else if (isToolMessage) {
    messageClassName = `flex items-start gap-2 px-2 text-xs leading-relaxed text-gray-500 ${isRunningToolMessage ? "sp-ai-thinking" : ""}`;
  } else {
    messageClassName = "px-2 text-sm text-gray-800";
  }

  return (
    <div key={message.id} className={wrapperClassName}>
      {message.role === "user" ? (
        <div className={messageClassName}>
          <User className="mt-0.5 h-3.5 w-3.5 shrink-0 text-slate-500" aria-hidden="true" />
          <span className="min-w-0 whitespace-pre-wrap break-words">{message.content}</span>
        </div>
      ) : isToolMessage ? (
        <div className={messageClassName}>
          <Activity className="mt-0.5 h-3.5 w-3.5 shrink-0 text-gray-400" aria-hidden="true" />
          <span className="min-w-0 whitespace-pre-wrap break-words">{message.content}</span>
        </div>
      ) : (
        <div className={messageClassName}>
          {message.role === "assistant" ? <AiMessageMarkdown content={message.content} /> : message.content}
        </div>
      )}
    </div>
  );
}

function AiMessageMarkdown({ content }: { content: string }) {
  return (
    <div className="max-w-none text-slate-800 [&_h1]:mb-1.5 [&_h1]:mt-1 [&_h1]:text-lg [&_h1]:font-semibold [&_h1]:leading-tight [&_h1:first-child]:mt-0 [&_h2]:mb-1 [&_h2]:mt-1 [&_h2]:text-base [&_h2]:font-semibold [&_h2]:leading-tight [&_h2:first-child]:mt-0 [&_h3]:mb-1.5 [&_h3]:mt-2 [&_h3]:text-sm [&_h3]:font-semibold [&_h3]:leading-tight [&_h3:first-child]:mt-0 [&_h4]:mb-0.5 [&_h4]:mt-1 [&_h4]:text-sm [&_h4]:font-medium [&_h4]:leading-tight [&_h4:first-child]:mt-0 [&_p]:mb-2 [&_p]:leading-relaxed [&_p:last-child]:mb-0 [&_ol]:mb-2 [&_ol]:ml-5 [&_ol]:list-decimal [&_ul]:mb-2 [&_ul]:ml-5 [&_ul]:list-disc [&_li]:mb-1 [&_blockquote]:my-2 [&_blockquote]:border-l-2 [&_blockquote]:border-slate-300 [&_blockquote]:pl-3 [&_hr]:my-6 [&_hr]:border-slate-300 [&_code]:rounded [&_code]:bg-slate-100 [&_code]:px-1 [&_code]:py-0.5 [&_code]:text-xs [&_pre]:my-2 [&_pre]:overflow-auto [&_pre]:rounded [&_pre]:bg-slate-100 [&_pre]:p-2 [&_pre_code]:bg-transparent [&_pre_code]:p-0 [&_a]:underline [&_a]:underline-offset-2 [&_a]:decoration-current">
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkBreaks]}
        components={{
          a: ({ children, href }) => (
            <a href={href} target="_blank" rel="noopener noreferrer" className="underline">
              {children}
            </a>
          ),
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  );
}

type ProposalsListProps = {
  pendingProposal: AiBuilderProposal;
  applyShortcutHint: string;
  pendingProposalSummaries: string[];
  onApplyProposal: () => void;
  onDiscardProposal: () => void;
  isApplyingProposal: boolean;
  aiError: string | null;
  disabled: boolean;
};

function ProposalsList({
  pendingProposal,
  applyShortcutHint,
  pendingProposalSummaries,
  onApplyProposal,
  onDiscardProposal,
  isApplyingProposal,
  aiError,
  disabled,
}: ProposalsListProps) {
  const isDisabled = disabled || isApplyingProposal || pendingProposal.operations.length === 0;

  return (
    <div className="relative rounded-md border border-blue-200 bg-blue-50 px-3 py-3 space-y-2">
      <span className="absolute right-2 top-2 text-[10px] text-blue-800">{`${applyShortcutHint} to accept`}</span>
      <ul className="text-sm text-blue-900 list-disc pl-5 space-y-1">
        {pendingProposalSummaries.map((summary, index) => (
          <li key={`${pendingProposal.id}-${index}`}>{summary}</li>
        ))}
      </ul>

      <div className="flex items-center gap-2 pt-1">
        <Button size="sm" onClick={onApplyProposal} disabled={isDisabled}>
          Apply changes
        </Button>
        <Button size="sm" variant="outline" onClick={onDiscardProposal} disabled={isDisabled}>
          Discard
        </Button>
      </div>

      {aiError ? <p className="text-xs text-red-700">{aiError}</p> : null}
    </div>
  );
}

type InputFormProps = {
  aiInputRef: React.RefObject<HTMLTextAreaElement | null>;
  aiInput: string;
  onAiInputChange: (value: string) => void;
  onSendPrompt: () => void;
  disabled: boolean;
  canvasId?: string;
  isGeneratingResponse: boolean;
  maxAiInputHeight: number;
  expanded?: boolean;
};

const TEXT_AREA_CLASSNAME = cn(
  "min-h-[20px] flex-1 resize-none border-0",
  "rounded-sm bg-transparent px-0.5 py-0.5 shadow-none",
  "focus-visible:ring-0 focus-visible:border-transparent",
);

function InputForm({
  aiInputRef,
  aiInput,
  onAiInputChange,
  onSendPrompt,
  disabled,
  canvasId,
  isGeneratingResponse,
  maxAiInputHeight,
  expanded = false,
}: InputFormProps) {
  const isDisabled = disabled || isGeneratingResponse || !canvasId || !aiInput.trim();

  const keyDownHandler = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      onSendPrompt();
    }
  };

  const submitHandler = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    onSendPrompt();
  };

  return (
    <div className={cn("m-1.5", expanded && "mb-3")}>
      <form
        onSubmit={submitHandler}
        className={cn("rounded-md border border-slate-300 bg-white p-1.5", expanded && "p-3 shadow-sm")}
      >
        <Textarea
          ref={aiInputRef}
          value={aiInput}
          onChange={(e) => onAiInputChange(e.target.value)}
          onKeyDown={keyDownHandler}
          placeholder="What would you like to build?"
          disabled={disabled || !canvasId}
          rows={expanded ? 4 : 1}
          className={cn(TEXT_AREA_CLASSNAME, expanded && "min-h-[112px] text-[15px] leading-6")}
          style={{ maxHeight: `${maxAiInputHeight}px` }}
        />

        <div className="flex items-center justify-end">
          <SubmitButton disabled={isDisabled} />
        </div>
      </form>
    </div>
  );
}

const SUBMIT_BUTTON_CLASSNAME = cn(
  "p-1 rounded-full bg-slate-600 text-white hover:bg-slate-700",
  "cursor-pointer",
  "disabled:opacity-50 disabled:cursor-not-allowed",
  "flex items-center justify-center",
);

function SubmitButton({ disabled }: { disabled: boolean }) {
  return (
    <button type="submit" className={SUBMIT_BUTTON_CLASSNAME} disabled={disabled} aria-label="Send prompt">
      <ArrowUp size={14} />
    </button>
  );
}
