import type { CanvasChangesetChange } from "@/api-client";
import { X } from "lucide-react";
import type { Dispatch, MouseEvent as ReactMouseEvent, SetStateAction } from "react";
import { useCallback, useEffect, useRef, useState } from "react";
import { AiBuilderChatPanel } from "./AiBuilderChatPanel";
import type { AiBuilderMessage, AiBuilderProposal, AiChatSession } from "./agentChat";
import { sendChatPrompt } from "./agentChat";
import type { AgentState } from "./useAgentState";
import { useApplyAiProposal } from "./useApplyAiProposal";
import { useLoadChatConversation } from "./useLoadChatConversation";
import { useLoadChatSessions } from "./useLoadChatSessions";
import { useSidebarWidth } from "./useSidebarWidth";

export interface AgentSidebarProps {
  agentState: AgentState;
}

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
  const { canvasId, organizationId, agentContext, onApplyAiOperations } = agentState;

  const aiInputRef = useRef<HTMLTextAreaElement>(null);
  const [aiInput, setAiInput] = useState("");
  const [aiMessages, setAiMessages] = useState<AiBuilderMessage[]>([]);
  const [chatSessions, setChatSessions] = useState<AiChatSession[]>([]);
  const [currentChatId, setCurrentChatId] = useState<string | null>(null);
  const [isGeneratingResponse, setIsGeneratingResponse] = useState(false);
  const [aiError, setAiError] = useState<string | null>(null);

  const { isApplyingProposal, setPendingProposal, pendingProposal, handleDiscardProposal, onApplyProposal } =
    useProposalState({
      setAiError,
      setAiMessages,
      onApplyAiOperations,
    });

  const handleSendPrompt = async (value?: string) =>
    await sendChatPrompt({
      value,
      aiInput,
      canvasId,
      organizationId,
      agentContext,
      currentChatId,
      isGeneratingResponse,
      setChatSessions,
      setCurrentChatId,
      setAiMessages,
      setAiInput,
      setAiError,
      setIsGeneratingResponse,
      setPendingProposal,
      focusInput: () => aiInputRef.current?.focus(),
    });

  const handleStartNewChatSession = () => {
    setCurrentChatId(null);
    setAiMessages([]);
    setPendingProposal(null);
    setAiError(null);
    requestAnimationFrame(() => {
      aiInputRef.current?.focus();
    });
  };

  const handleSelectChatSession = (chatId: string) => {
    setCurrentChatId(chatId);
    setPendingProposal(null);
    setAiError(null);
  };

  // reset state when canvasId changes
  useEffect(() => {
    setCurrentChatId(null);
    setAiMessages([]);
    setPendingProposal(null);
    setAiError(null);
    setAiInput("");
  }, [canvasId, setCurrentChatId, setAiMessages, setPendingProposal, setAiError, setAiInput]);

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
    setAiMessages,
    setPendingProposal,
    setAiError,
  });

  return (
    <AgentSidebarContainer onClose={agentState.closeSidebar}>
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
        onStartNewSession={handleStartNewChatSession}
        onSendPrompt={() => void handleSendPrompt()}
        aiInputRef={aiInputRef}
      />
    </AgentSidebarContainer>
  );
}

type OnMouseDown = (event: ReactMouseEvent<HTMLDivElement>) => void;

function AgentSidebarContainer({ children, onClose }: { children: React.ReactNode; onClose: () => void }) {
  const { sidebarRef, isResizing, onResizeMouseDown, sidebarStyle } = useSidebarWidth();

  return (
    <div
      ref={sidebarRef}
      className="relative border-r border-border shrink-0 h-full z-21 flex flex-col overflow-hidden bg-white"
      style={sidebarStyle}
      data-testid="agent-sidebar"
    >
      <div className="flex items-center justify-between gap-3 px-4 py-3 border-b border-border shrink-0">
        <h2 className="text-base font-medium">SuperPlane Agent</h2>
        <CloseButton onClose={onClose} />
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

function CloseButton({ onClose }: { onClose: () => void }) {
  return (
    <button
      type="button"
      onClick={onClose}
      data-testid="close-agent-sidebar-button"
      className="z-40 w-8 h-8 hover:bg-slate-950/5 rounded-md flex items-center justify-center cursor-pointer leading-none border border-transparent text-muted-foreground"
      aria-label="Close SuperPlane Agent"
    >
      <X size={16} />
    </button>
  );
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
