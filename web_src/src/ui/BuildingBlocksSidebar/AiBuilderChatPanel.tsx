import { Button } from "@/components/ui/button";
import { TabsContent } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { AiBuilderMessage, AiBuilderProposal } from "@/ui/BuildingBlocksSidebar/agentChat";
import { ArrowUp } from "lucide-react";
import { useEffect, useRef } from "react";
import ReactMarkdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import { cn } from "../../lib/utils";

type AiBuilderChatPanelProps = {
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
  onSendPrompt: () => void;
  aiInputRef: React.RefObject<HTMLTextAreaElement | null>;
};

export function AiBuilderChatPanel({
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
  onSendPrompt,
  aiInputRef,
}: AiBuilderChatPanelProps) {
  const aiMessagesContainerRef = useRef<HTMLDivElement>(null);
  const maxAiInputHeight = 160;

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
  }, [aiInput, aiInputRef]);

  return (
    <TabsContent value="ai" className="mt-0 flex-1 overflow-hidden px-5 pb-5">
      <div className="h-full rounded-md border border-border bg-slate-50/30 flex flex-col">
        <div ref={aiMessagesContainerRef} className="flex-1 overflow-y-auto space-y-1 px-2 py-3">
          <AiMessages messages={aiMessages} />

          {isGeneratingResponse ? (
            <div className="sp-ai-thinking text-xs text-gray-500 px-1 py-1 rounded-sm">Planning next steps...</div>
          ) : null}

          {pendingProposal && (
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
          )}

          {!pendingProposal && aiError ? <p className="text-xs text-red-700">{aiError}</p> : null}
        </div>

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
    </TabsContent>
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

function AiMessage({ message }: { message: AiBuilderMessage }) {
  const isEmptyAssistantPlaceholder = message.role === "assistant" && message.content.trim().length === 0;
  if (isEmptyAssistantPlaceholder) {
    return null;
  }

  const isToolMessage = message.role === "tool";
  const isRunningToolMessage = isToolMessage && message.toolStatus === "running";

  let messageClassName = "";

  if (message.role === "user") {
    messageClassName = "w-full rounded-md bg-blue-600 text-white px-2 py-1.5 text-sm";
  } else if (isToolMessage) {
    messageClassName = `px-2 text-xs leading-relaxed text-gray-500 ${isRunningToolMessage ? "sp-ai-thinking" : ""}`;
  } else {
    messageClassName = "px-2 text-sm text-gray-800";
  }

  return (
    <div key={message.id} className="w-full">
      <div className={messageClassName}>
        {message.role === "assistant" ? <AiMessageMarkdown content={message.content} /> : message.content}
      </div>
    </div>
  );
}

function AiMessageMarkdown({ content }: { content: string }) {
  return (
    <div className="max-w-none [&_p]:mb-2 [&_p:last-child]:mb-0 [&_ol]:mb-2 [&_ol]:ml-5 [&_ol]:list-decimal [&_ul]:mb-2 [&_ul]:ml-5 [&_ul]:list-disc [&_li]:mb-1 [&_code]:rounded [&_code]:bg-slate-100 [&_code]:px-1 [&_code]:py-0.5 [&_code]:text-xs [&_pre]:my-2 [&_pre]:overflow-auto [&_pre]:rounded [&_pre]:bg-slate-100 [&_pre]:p-2 [&_pre_code]:bg-transparent [&_pre_code]:p-0 [&_strong]:font-semibold">
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
    <div className="m-1.5">
      <form onSubmit={submitHandler} className="rounded-md border border-slate-300 bg-white p-1.5">
        <Textarea
          ref={aiInputRef}
          value={aiInput}
          onChange={(e) => onAiInputChange(e.target.value)}
          onKeyDown={keyDownHandler}
          placeholder="What would you like to build?"
          disabled={disabled || !canvasId}
          rows={1}
          className={TEXT_AREA_CLASSNAME}
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
