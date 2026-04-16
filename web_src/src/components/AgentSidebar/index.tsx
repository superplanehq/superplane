import type { CanvasChangesetChange } from "@/api-client";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { X } from "lucide-react";
import type { MouseEvent as ReactMouseEvent } from "react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { AiBuilderChatPanel } from "./AiBuilderChatPanel";
import type { AgentContext, AiBuilderMessage, AiBuilderProposal, AiChatSession } from "./agentChat";
import {
  deleteAgentChatSession,
  loadChatConversation,
  loadChatSessions,
  pushAiMessages,
  sendChatPrompt,
} from "./agentChat";
import type { AgentState } from "./useAgentState";
import { useApplyOnCmdEnter } from "./useApplyOnCmdEnter";
import { useApplyShortcutHint } from "./useApplyShortcutHint";
import { useFormatOperation } from "./useFormatOperation";
import { useSidebarWidth } from "./useSidebarWidth";

export interface AgentSidebarProps {
  agentState: AgentState;
}

export function AgentSidebar({ agentState }: AgentSidebarProps) {
  const {
    agentContext,
    isAgentSidebarOpen,
    handleAgentSidebarOpenChange,
    canvasId,
    organizationId,
    readOnly,
    onApplyAiOperations,
    showAgentSidebarToggle,
  } = agentState;

  if (!showAgentSidebarToggle) {
    return null;
  }

  if (!isAgentSidebarOpen || !agentContext.enabled) {
    return null;
  }

  return (
    <OpenAgentSidebar
      onToggle={handleAgentSidebarOpenChange}
      agentContext={agentContext}
      canvasId={canvasId}
      organizationId={organizationId}
      onApplyAiOperations={onApplyAiOperations}
      disabled={readOnly}
    />
  );
}

interface OpenAgentSidebarProps {
  onToggle: (open: boolean) => void;
  agentContext: AgentContext;
  canvasId?: string;
  organizationId?: string;
  onApplyAiOperations?: (changes: CanvasChangesetChange[]) => Promise<void>;
  disabled: boolean;
}

function OpenAgentSidebar({
  onToggle,
  agentContext,
  canvasId,
  organizationId,
  onApplyAiOperations,
  disabled,
}: OpenAgentSidebarProps) {
  const aiInputRef = useRef<HTMLTextAreaElement>(null);
  const { sidebarRef, sidebarWidth, isResizing, onResizeMouseDown } = useSidebarWidth();
  const [aiInput, setAiInput] = useState("");
  const [aiMessages, setAiMessages] = useState<AiBuilderMessage[]>([]);
  const [chatSessions, setChatSessions] = useState<AiChatSession[]>([]);
  const [currentChatId, setCurrentChatId] = useState<string | null>(null);
  const currentChatIdRef = useRef(currentChatId);
  currentChatIdRef.current = currentChatId;
  const [isLoadingChatSessions, setIsLoadingChatSessions] = useState(false);
  const [isLoadingChatMessages, setIsLoadingChatMessages] = useState(false);
  const [isGeneratingResponse, setIsGeneratingResponse] = useState(false);
  const [isApplyingProposal, setIsApplyingProposal] = useState(false);
  const [aiError, setAiError] = useState<string | null>(null);
  const [pendingProposal, setPendingProposal] = useState<AiBuilderProposal | null>(null);

  const applyShortcutHint = useApplyShortcutHint();

  const handleSendPrompt = useCallback(
    async (value?: string) => {
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
    },
    [agentContext, aiInput, canvasId, currentChatId, isGeneratingResponse, organizationId],
  );

  const handleStartNewChatSession = useCallback(() => {
    setCurrentChatId(null);
    setAiMessages([]);
    setPendingProposal(null);
    setAiError(null);
    requestAnimationFrame(() => {
      aiInputRef.current?.focus();
    });
  }, []);

  const handleSelectChatSession = useCallback((chatId: string) => {
    setCurrentChatId(chatId);
    setPendingProposal(null);
    setAiError(null);
  }, []);

  const handleDeleteChatSession = useCallback(
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
    [canvasId, organizationId],
  );

  const handleDiscardProposal = useCallback(() => {
    setPendingProposal(null);
  }, []);

  const formatOperation = useFormatOperation();

  const pendingProposalSummaries = useMemo(() => {
    if (!pendingProposal) {
      return [];
    }

    return (pendingProposal.changeset.changes || []).map((change) => formatOperation(change));
  }, [formatOperation, pendingProposal]);

  const handleApplyProposal = useCallback(async () => {
    if (!pendingProposal) return;

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
  }, [onApplyAiOperations, pendingProposal]);

  const canApplyProposalWithShortcut = !!pendingProposal && (pendingProposal.changeset.changes || []).length > 0;

  useApplyOnCmdEnter({
    enabled: canApplyProposalWithShortcut,
    disabled,
    isApplying: isApplyingProposal,
    onApply: handleApplyProposal,
  });

  useEffect(() => {
    setCurrentChatId(null);
    setAiMessages([]);
    setPendingProposal(null);
    setAiError(null);
    setAiInput("");
  }, [canvasId]);

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
  }, [canvasId, organizationId]);

  useEffect(() => {
    let cancelled = false;

    if (!canvasId || !organizationId || !currentChatId) {
      if (!currentChatId) {
        setAiMessages([]);
        setPendingProposal(null);
      }
      setIsLoadingChatMessages(false);
      return () => {
        cancelled = true;
      };
    }

    void (async () => {
      setIsLoadingChatMessages(true);
      try {
        const messages = await loadChatConversation({
          chatId: currentChatId,
          canvasId,
          organizationId,
        });
        if (cancelled) {
          return;
        }

        setAiMessages(messages);
        setAiError(null);
      } catch (error) {
        if (!cancelled) {
          console.warn("Failed to load chat conversation:", error);
          setAiError(error instanceof Error ? error.message : "Failed to load chat conversation.");
        }
      } finally {
        if (!cancelled) {
          setIsLoadingChatMessages(false);
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [canvasId, currentChatId, organizationId]);

  return (
    <div
      ref={sidebarRef}
      className="relative border-r border-border shrink-0 h-full z-21 flex flex-col overflow-hidden bg-white"
      style={{ width: `${sidebarWidth}px`, minWidth: `${sidebarWidth}px`, maxWidth: `${sidebarWidth}px` }}
      data-testid="agent-sidebar"
    >
      <div className="flex items-center justify-between gap-3 px-4 py-3 border-b border-border shrink-0">
        <h2 className="text-base font-medium">SuperPlane Agent</h2>
        <CloseButton onToggle={onToggle} />
      </div>

      <div className="flex flex-1 flex-col min-h-0">
        <AiBuilderChatPanel
          chatSessions={chatSessions}
          currentChatId={currentChatId}
          isLoadingChatSessions={isLoadingChatSessions}
          isLoadingChatMessages={isLoadingChatMessages}
          aiMessages={aiMessages}
          isGeneratingResponse={isGeneratingResponse}
          pendingProposal={pendingProposal}
          pendingProposalSummaries={pendingProposalSummaries}
          applyShortcutHint={applyShortcutHint}
          onApplyProposal={() => void handleApplyProposal()}
          onDiscardProposal={handleDiscardProposal}
          isApplyingProposal={isApplyingProposal}
          aiError={aiError}
          disabled={disabled}
          canvasId={canvasId}
          aiInput={aiInput}
          onAiInputChange={setAiInput}
          onSelectChat={handleSelectChatSession}
          onDeleteChat={handleDeleteChatSession}
          onStartNewSession={handleStartNewChatSession}
          onSendPrompt={() => void handleSendPrompt()}
          aiInputRef={aiInputRef}
        />
      </div>

      <AgentSidebarResizeHandle isResizing={isResizing} onMouseDown={onResizeMouseDown} />
    </div>
  );
}

type OnMouseDown = (event: ReactMouseEvent<HTMLDivElement>) => void;

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

function CloseButton({ onToggle }: { onToggle: (open: boolean) => void }) {
  return (
    <button
      type="button"
      onClick={() => onToggle(false)}
      data-testid="close-agent-sidebar-button"
      className="z-40 w-8 h-8 hover:bg-slate-950/5 rounded-md flex items-center justify-center cursor-pointer leading-none border border-transparent text-muted-foreground"
      aria-label="Close SuperPlane Agent"
    >
      <X size={16} />
    </button>
  );
}
