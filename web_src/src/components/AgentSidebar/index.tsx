import type { CanvasChangesetChange } from "@/api-client";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { X } from "lucide-react";
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

export const AGENT_SIDEBAR_WIDTH_STORAGE_KEY = "agentSidebarWidth";

const AI_BUILDER_STORAGE_KEY_PREFIX = "sp:canvas-ai-builder:";

export interface AgentSidebarProps {
  isOpen: boolean;
  onToggle: (open: boolean) => void;
  agentContext: AgentContext;
  canvasId?: string;
  organizationId?: string;
  onApplyAiOperations?: (changes: CanvasChangesetChange[]) => Promise<void>;
  disabled?: boolean;
  disabledMessage?: string;
}

export function AgentSidebar({
  isOpen,
  onToggle,
  agentContext,
  canvasId,
  organizationId,
  onApplyAiOperations,
  disabled = false,
  disabledMessage,
}: AgentSidebarProps) {
  if (!isOpen || !agentContext.enabled) {
    return null;
  }

  return (
    <OpenAgentSidebar
      onToggle={onToggle}
      agentContext={agentContext}
      canvasId={canvasId}
      organizationId={organizationId}
      onApplyAiOperations={onApplyAiOperations}
      disabled={disabled}
      disabledMessage={disabledMessage}
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
  disabledMessage?: string;
}

function OpenAgentSidebar({
  onToggle,
  agentContext,
  canvasId,
  organizationId,
  onApplyAiOperations,
  disabled,
  disabledMessage: _disabledMessage,
}: OpenAgentSidebarProps) {
  const sidebarRef = useRef<HTMLDivElement>(null);
  const aiInputRef = useRef<HTMLTextAreaElement>(null);
  const [sidebarWidth, setSidebarWidth] = useState(() => {
    if (typeof window === "undefined") {
      return 400;
    }

    const saved = window.localStorage.getItem(AGENT_SIDEBAR_WIDTH_STORAGE_KEY);
    return saved ? parseInt(saved, 10) : 400;
  });
  const [isResizing, setIsResizing] = useState(false);
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

  const applyShortcutHint = useMemo(() => {
    if (typeof navigator === "undefined") {
      return "Ctrl+Enter";
    }

    const isMacPlatform = /Mac|iPhone|iPad|iPod/i.test(`${navigator.platform} ${navigator.userAgent}`);
    return isMacPlatform ? "Cmd+Enter" : "Ctrl+Enter";
  }, []);

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

  const formatOperation = useCallback((change: CanvasChangesetChange) => {
    const getNodeId = (nodeId?: string) => nodeId || "node";

    switch (change.type) {
      case "ADD_NODE":
        return `Add node ${getNodeId(change.node?.id)} (${change.node?.block || "unknown"})`;
      case "UPDATE_NODE":
        return `Update node ${getNodeId(change.node?.id)}`;
      case "DELETE_NODE":
        return `Delete node ${getNodeId(change.node?.id)}`;
      case "ADD_EDGE":
        return `Connect ${getNodeId(change.edge?.sourceId)} -> ${getNodeId(change.edge?.targetId)}`;
      case "DELETE_EDGE":
        return `Disconnect ${getNodeId(change.edge?.sourceId)} -> ${getNodeId(change.edge?.targetId)}`;
      default:
        return "Update canvas";
    }
  }, []);

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

  useEffect(() => {
    if (!pendingProposal || (pendingProposal.changeset.changes || []).length === 0) {
      return;
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.isComposing || event.key !== "Enter") {
        return;
      }

      if (!(event.metaKey || event.ctrlKey)) {
        return;
      }

      if (disabled || isApplyingProposal) {
        return;
      }

      event.preventDefault();
      void handleApplyProposal();
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => {
      window.removeEventListener("keydown", handleKeyDown);
    };
  }, [disabled, handleApplyProposal, isApplyingProposal, pendingProposal]);

  useEffect(() => {
    localStorage.setItem(AGENT_SIDEBAR_WIDTH_STORAGE_KEY, String(sidebarWidth));
  }, [sidebarWidth]);

  useEffect(() => {
    setCurrentChatId(null);
    setAiMessages([]);
    setPendingProposal(null);
    setAiError(null);
    setAiInput("");
  }, [canvasId]);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    const keysToRemove: string[] = [];
    for (let index = 0; index < window.localStorage.length; index += 1) {
      const key = window.localStorage.key(index);
      if (key?.startsWith(AI_BUILDER_STORAGE_KEY_PREFIX)) {
        keysToRemove.push(key);
      }
    }

    for (const key of keysToRemove) {
      window.localStorage.removeItem(key);
    }
  }, []);

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

  const handleMouseDownResize = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    setIsResizing(true);
  }, []);

  const handleMouseMoveResize = useCallback(
    (e: MouseEvent) => {
      if (!isResizing || !sidebarRef.current) return;

      const rect = sidebarRef.current.getBoundingClientRect();
      const newWidth = e.clientX - rect.left;
      const clampedWidth = Math.max(280, Math.min(560, newWidth));
      setSidebarWidth(clampedWidth);
    },
    [isResizing],
  );

  const handleMouseUpResize = useCallback(() => {
    setIsResizing(false);
  }, []);

  useEffect(() => {
    if (isResizing) {
      document.addEventListener("mousemove", handleMouseMoveResize);
      document.addEventListener("mouseup", handleMouseUpResize);
      document.body.style.cursor = "ew-resize";
      document.body.style.userSelect = "none";

      return () => {
        document.removeEventListener("mousemove", handleMouseMoveResize);
        document.removeEventListener("mouseup", handleMouseUpResize);
        document.body.style.cursor = "";
        document.body.style.userSelect = "";
      };
    }
  }, [isResizing, handleMouseMoveResize, handleMouseUpResize]);

  return (
    <div
      ref={sidebarRef}
      className="relative border-r border-border shrink-0 h-full z-21 flex flex-col overflow-hidden bg-white"
      style={{ width: `${sidebarWidth}px`, minWidth: `${sidebarWidth}px`, maxWidth: `${sidebarWidth}px` }}
      data-testid="agent-sidebar"
    >
      <div className="flex items-center justify-between gap-3 px-4 py-3 border-b border-border shrink-0">
        <h2 className="text-base font-medium">SuperPlane Agent</h2>
        <button
          type="button"
          onClick={() => onToggle(false)}
          data-testid="close-agent-sidebar-button"
          className="z-40 w-8 h-8 hover:bg-slate-950/5 rounded-md flex items-center justify-center cursor-pointer leading-none border border-transparent text-muted-foreground"
          aria-label="Close SuperPlane Agent"
        >
          <X size={16} />
        </button>
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

      <div
        onMouseDown={handleMouseDownResize}
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
    </div>
  );
}

export type { UseAgentStateOptions } from "./useAgentState";
export { CANVAS_AGENT_SIDEBAR_STORAGE_KEY, useAgentState } from "./useAgentState";
