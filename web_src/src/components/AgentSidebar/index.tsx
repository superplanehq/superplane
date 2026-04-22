import type { CanvasChangesetChange } from "@/api-client";
import { ChevronLeft, X } from "lucide-react";
import type { Dispatch, MouseEvent as ReactMouseEvent, SetStateAction } from "react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { AiBuilderChatPanel } from "./AiBuilderChatPanel";
import type { AiBuilderMessage, AiBuilderProposal, AiChatSession } from "./agentChat";
import { sendChatPrompt } from "./agentChat";
import type { AgentState } from "./useAgentState";
import { useApplyAiProposal } from "./useApplyAiProposal";
import { useLoadChatConversation } from "./useLoadChatConversation";
import { useLoadChatSessions } from "./useLoadChatSessions";
import { usePollForMessages } from "./usePollForMessages";
import { useSidebarWidth } from "./useSidebarWidth";

export interface AgentSidebarProps {
  agentState: AgentState;
}

type UseChatHandlersParams = {
  aiInput: string;
  aiInputRef: React.RefObject<HTMLTextAreaElement | null>;
  canvasId?: string;
  organizationId?: string;
  agentContext: AgentState["agentContext"];
  currentChatId: string | null;
  isGeneratingResponse: boolean;
  isStreamingRef: React.MutableRefObject<boolean>;
  setChatSessions: Dispatch<SetStateAction<AiChatSession[]>>;
  setCurrentChatId: Dispatch<SetStateAction<string | null>>;
  setAiMessages: Dispatch<SetStateAction<AiBuilderMessage[]>>;
  setAiInput: Dispatch<SetStateAction<string>>;
  setAiError: Dispatch<SetStateAction<string | null>>;
  setIsGeneratingResponse: Dispatch<SetStateAction<boolean>>;
  setPendingProposal: Dispatch<SetStateAction<AiBuilderProposal | null>>;
};

export function AgentSidebar({ agentState }: AgentSidebarProps) {
  if (!agentState.showAgentSidebarToggle) {
    return null;
  }

  if (!agentState.isAgentSidebarOpen || !agentState.agentContext.enabled) {
    return null;
  }

  return <OpenAgentSidebar agentState={agentState} />;
}

function OpenAgentSidebar({ agentState }: AgentSidebarProps) {
  const { canvasId, organizationId, agentContext, onApplyAiOperations, currentChatId, setCurrentChatId } = agentState;

  const aiInputRef = useRef<HTMLTextAreaElement>(null);
  const [aiInput, setAiInput] = useState("");
  const [aiMessages, setAiMessages] = useState<AiBuilderMessage[]>([]);
  const [chatSessions, setChatSessions] = useState<AiChatSession[]>([]);
  const [isGeneratingResponse, setIsGeneratingResponse] = useState(false);
  const [aiError, setAiError] = useState<string | null>(null);

  // Track whether an SSE stream is actively delivering events so the polling
  // hook knows to stand down.
  const isStreamingRef = useRef(false);

  const { isApplyingProposal, setPendingProposal, pendingProposal, handleDiscardProposal, onApplyProposal } =
    useProposalState({
      setAiError,
      setAiMessages,
      onApplyAiOperations,
    });

  const { handleSendPrompt, handleStartNewChatSession, handleSelectChatSession } = useChatHandlers({
    aiInput,
    aiInputRef,
    canvasId,
    organizationId,
    agentContext,
    currentChatId,
    isGeneratingResponse,
    isStreamingRef,
    setChatSessions,
    setCurrentChatId,
    setAiMessages,
    setAiInput,
    setAiError,
    setIsGeneratingResponse,
    setPendingProposal,
  });

  // load previous chat sessions
  const isLoadingChatSessions = useLoadChatSessions({
    canvasId,
    organizationId,
    setChatSessions,
    setCurrentChatId,
    setAiMessages,
  });

  // load chat conversation when currentChatId changes
  const isLoadingChatMessages = useLoadChatConversation({
    canvasId,
    organizationId,
    currentChatId,
    setChatSessions,
    setAiMessages,
    setPendingProposal,
    setAiError,
    setIsGeneratingResponse,
  });

  // poll for updates when a run is in progress but no SSE stream is active
  usePollForMessages({
    canvasId,
    organizationId,
    currentChatId,
    isGeneratingResponse,
    isStreamingRef,
    setChatSessions,
    setAiMessages,
    setIsGeneratingResponse,
    setPendingProposal,
  });

  const sidebarTitle = useMemo(() => {
    if (!currentChatId) {
      return "Agent";
    }

    const session = chatSessions.find((s) => s.id === currentChatId);
    return session?.title ?? "Agent";
  }, [chatSessions, currentChatId]);

  return (
    <AgentSidebarContainer
      onClose={agentState.closeSidebar}
      title={sidebarTitle}
      showBack={currentChatId !== null}
      onBack={handleStartNewChatSession}
    >
      <AiBuilderChatPanel
        chatSessions={chatSessions}
        currentChatId={currentChatId}
        isLoadingChatSessions={isLoadingChatSessions}
        isLoadingChatMessages={isLoadingChatMessages}
        aiMessages={aiMessages}
        isGeneratingResponse={isGeneratingResponse}
        pendingProposal={pendingProposal}
        onApplyProposal={onApplyProposal}
        onDiscardProposal={handleDiscardProposal}
        isApplyingProposal={isApplyingProposal}
        aiError={aiError}
        disabled={agentState.readOnly}
        canvasId={canvasId}
        organizationId={organizationId}
        setChatSessions={setChatSessions}
        setCurrentChatId={setCurrentChatId}
        setAiMessages={setAiMessages}
        setPendingProposal={setPendingProposal}
        setAiError={setAiError}
        aiInput={aiInput}
        onAiInputChange={setAiInput}
        onSelectChat={handleSelectChatSession}
        onSendPrompt={() => void handleSendPrompt()}
        aiInputRef={aiInputRef}
      />
    </AgentSidebarContainer>
  );
}

type OnMouseDown = (event: ReactMouseEvent<HTMLDivElement>) => void;

function AgentSidebarContainer({
  children,
  onClose,
  title,
  showBack,
  onBack,
}: {
  children: React.ReactNode;
  onClose: () => void;
  title: string;
  showBack: boolean;
  onBack: () => void;
}) {
  const { sidebarRef, isResizing, onResizeMouseDown, sidebarStyle } = useSidebarWidth();

  return (
    <div
      ref={sidebarRef}
      className="relative border-r border-border shrink-0 h-full z-21 flex flex-col overflow-hidden bg-white"
      style={sidebarStyle}
      data-testid="agent-sidebar"
    >
      <div className="flex items-center justify-between gap-3 px-4 py-2.5 border-b border-border shrink-0 min-w-0">
        <div className="flex min-w-0 flex-1 items-center gap-1">
          {showBack ? <BackButton onBack={onBack} /> : null}
          <h2 className="text-base font-medium min-w-0 flex-1 truncate" title={title}>
            {title}
          </h2>
        </div>
        <div className="flex shrink-0 items-center gap-1">
          <CloseButton onClose={onClose} />
        </div>
      </div>

      <div className="flex flex-1 flex-col min-h-0">{children}</div>

      <AgentSidebarResizeHandle isResizing={isResizing} onMouseDown={onResizeMouseDown} />
    </div>
  );
}

function AgentSidebarResizeHandle({ isResizing, onMouseDown }: { isResizing: boolean; onMouseDown: OnMouseDown }) {
  return (
    <div
      onMouseDown={onMouseDown}
      className={`absolute right-0 top-0 bottom-0 w-4 cursor-ew-resize hover:bg-gray-100 transition-colors flex items-center justify-center group ${
        isResizing ? "bg-blue-50" : ""
      }`}
      style={{ marginRight: "-8px" }}
      aria-hidden
    >
      <div
        className={`w-2 h-14 rounded-full bg-gray-300 group-hover:bg-gray-800 transition-colors ${
          isResizing ? "bg-blue-500" : ""
        }`}
      />
    </div>
  );
}

function BackButton({ onBack }: { onBack: () => void }) {
  return (
    <button
      type="button"
      onClick={onBack}
      data-testid="agent-sidebar-back-button"
      className="z-40 shrink-0 w-6 h-6 hover:bg-slate-950/5 rounded-md flex items-center justify-center cursor-pointer leading-none border border-transparent text-muted-foreground"
      aria-label="Leave conversation"
    >
      <ChevronLeft size={18} />
    </button>
  );
}

function CloseButton({ onClose }: { onClose: () => void }) {
  return (
    <button
      type="button"
      onClick={onClose}
      data-testid="close-agent-sidebar-button"
      className="z-40 w-6 h-6 hover:bg-slate-950/5 rounded-md flex items-center justify-center cursor-pointer leading-none border border-transparent text-muted-foreground"
      aria-label="Close SuperPlane Agent"
    >
      <X size={16} />
    </button>
  );
}

function useChatHandlers(p: UseChatHandlersParams) {
  // reset local state when canvasId changes (currentChatId reset is handled in useAgentState)
  useEffect(() => {
    p.setAiMessages([]);
    p.setPendingProposal(null);
    p.setAiError(null);
    p.setAiInput("");
  }, [p.canvasId, p.setAiMessages, p.setPendingProposal, p.setAiError, p.setAiInput]); // eslint-disable-line react-hooks/exhaustive-deps

  const handleSendPrompt = async (value?: string) => {
    p.isStreamingRef.current = true;
    try {
      await sendChatPrompt({
        value,
        aiInput: p.aiInput,
        canvasId: p.canvasId,
        organizationId: p.organizationId,
        agentContext: p.agentContext,
        currentChatId: p.currentChatId,
        isGeneratingResponse: p.isGeneratingResponse,
        setChatSessions: p.setChatSessions,
        setCurrentChatId: p.setCurrentChatId,
        setAiMessages: p.setAiMessages,
        setAiInput: p.setAiInput,
        setAiError: p.setAiError,
        setIsGeneratingResponse: p.setIsGeneratingResponse,
        setPendingProposal: p.setPendingProposal,
        focusInput: () => p.aiInputRef.current?.focus(),
      });
    } finally {
      p.isStreamingRef.current = false;
    }
  };

  const handleStartNewChatSession = () => {
    p.setCurrentChatId(null);
    p.setAiMessages([]);
    p.setPendingProposal(null);
    p.setAiError(null);
    requestAnimationFrame(() => p.aiInputRef.current?.focus());
  };

  const handleSelectChatSession = (chatId: string) => {
    p.setCurrentChatId(chatId);
    p.setPendingProposal(null);
    p.setAiError(null);
  };

  return { handleSendPrompt, handleStartNewChatSession, handleSelectChatSession };
}

function useProposalState({
  setAiError,
  setAiMessages,
  onApplyAiOperations,
}: {
  setAiError: Dispatch<SetStateAction<string | null>>;
  setAiMessages: Dispatch<SetStateAction<AiBuilderMessage[]>>;
  onApplyAiOperations: (changes: CanvasChangesetChange[]) => Promise<void>;
}) {
  const [isApplyingProposal, setIsApplyingProposal] = useState(false);
  const [pendingProposal, setPendingProposal] = useState<AiBuilderProposal | null>(null);

  const handleDiscardProposal = useCallback(() => {
    setPendingProposal(null);
  }, []);

  const handleApplyProposal = useApplyAiProposal({
    onApplyAiOperations,
    pendingProposal,
    setAiError,
    setIsApplyingProposal,
    setAiMessages,
    setPendingProposal,
  });

  return {
    isApplyingProposal,
    setPendingProposal,
    pendingProposal,
    handleDiscardProposal,
    onApplyProposal: handleApplyProposal,
  };
}
