import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import type { AgentMode } from "@/components/AgentSidebar/agentMode";
import { AccountContext } from "@/contexts/accountContextState";
import { createSystemMessage } from "@/components/AgentSidebar/systemMessages";
import { ChatComposer } from "@/components/AgentSidebar/ChatComposer";
import { useChatScroll } from "@/components/AgentSidebar/useChatScroll";
import { useDraftActions } from "@/components/AgentSidebar/useDraftActions";
import { DraftActionsWidget } from "@/components/AgentSidebar/widgets/DraftActionsWidget";
import { OutcomeProgressWidget, type OutcomeState } from "@/components/AgentSidebar/widgets/OutcomeProgressWidget";
import type { RubricCategory } from "@/components/AgentSidebar/widgets/parser";
import {
  useAgentChatMessages,
  useCanvasAgentChat,
  useDefineAgentOutcome,
  useInterruptAgentChat,
  useResetCanvasAgentChat,
  useSendAgentChatMessage,
} from "@/hooks/useAgentChats";
import { useAgentSessionWebsocket } from "@/hooks/useAgentSessionWebsocket";
import { useCanvas, useInfiniteCanvasRuns } from "@/hooks/useCanvasData";
import {
  AGENT_BOOT_CONTEXT_READY_EVENT,
  clearAgentBootContext,
  clearAgentBootContextForCanvas,
  getAgentBootMessage,
  isAgentBootReady,
} from "@/lib/agentBootContext";
import { isCanvasWorkflowTab, type CanvasPageHeaderMode } from "@/pages/app/viewState";
import { ConversationTranscript } from "./AgentConversationTranscript";
import {
  createWebsocketCallbacks,
  isOutcomeActive,
  statusLabel,
  useConversationMessages,
  useStoredOutcomeState,
  useThinkingIndicator,
} from "./agentConversationState";
import type { AgentMessage, AgentOutgoingImage } from "./types";
import type { AgentStagingReadyHandler, CanvasToolSidebarState } from "./useCanvasToolSidebarState";
import {
  buildAgentStagingAutoOpenKey,
  claimAgentStagingAutoOpen,
  releaseAgentStagingAutoOpen,
} from "@/pages/app/lib/agent-staging-auto-open";
import { groupMessages } from "./agentMessageGroups";
import { useGreetedMessages } from "./useGreetedMessages";
import { AgentSetupNotice } from "./AgentSetupState";
import { getAgentSetupState, isSessionBusyError } from "./agentSetupStateModel";

const STREAMING_STATUS_RECONCILE_INTERVAL_MS = 15000;

type ChatConversationProps = {
  chatId: string;
  canvasId: string;
  organizationId: string;
  initialStatus: string;
  refreshChatStatus: () => Promise<string | undefined>;
  agentMode: AgentMode;
  onModeSwitch: (mode: AgentMode) => void;
  isEditing: boolean;
  onAgentStagingReady?: AgentStagingReadyHandler;
  onAgentStagingCommit?: (commitMessage: string) => Promise<boolean>;
  liveCanvasVersionId?: string;
  headerMode?: CanvasPageHeaderMode;
  isRunInspectionMode?: boolean;
};

type DraftActionsBarProps = {
  messages: AgentMessage[];
  canvasId: string;
  organizationId: string;
  isEditing: boolean;
  outcomePassed?: boolean;
  onVersionPublished?: () => void;
  onAgentStagingReady?: AgentStagingReadyHandler;
  onAgentStagingCommit?: (commitMessage: string) => Promise<boolean>;
  liveCanvasVersionId?: string;
  headerMode?: CanvasPageHeaderMode;
  isRunInspectionMode?: boolean;
};

type ConversationHandlers = {
  handleSend: (content: string, images?: AgentOutgoingImage[]) => Promise<void>;
  handleStop: () => void;
  handleQuickAction: (action: string) => Promise<void>;
  handleStartBuilding: (rubric: { title: string; criteria: string[]; categories?: RubricCategory[] }) => Promise<void>;
};

export function AgentTabPanel({ toolSidebarState }: { toolSidebarState: CanvasToolSidebarState }) {
  const canvasId = toolSidebarState.canvasId ?? "";
  const organizationId = toolSidebarState.organizationId ?? "";
  const chatQuery = useCanvasAgentChat(canvasId, organizationId, toolSidebarState.isToolSidebarOpen);
  const chatId = chatQuery.data?.id ?? null;
  const { account } = useContext(AccountContext);
  const firstName = account?.name?.split(" ")[0] ?? "there";
  const { refetch: refetchChat } = chatQuery;
  const refreshChatStatus = useCallback(async () => {
    const result = await refetchChat();
    return result.data?.status;
  }, [refetchChat]);

  const setupState = getAgentSetupState({
    chatId,
    error: chatQuery.error,
    isError: chatQuery.isError,
    isFetching: chatQuery.isFetching,
    isLoading: chatQuery.isLoading,
  });
  const agentUnavailable = setupState === "unavailable";
  const { markAgentAvailable, markAgentUnavailable } = toolSidebarState;
  useEffect(() => {
    if (agentUnavailable) markAgentUnavailable();
    if (!agentUnavailable && chatId) markAgentAvailable();
  }, [agentUnavailable, chatId, markAgentAvailable, markAgentUnavailable]);

  if (setupState) {
    return <AgentSetupNotice firstName={firstName} onRetry={() => void chatQuery.refetch()} state={setupState} />;
  }

  const readyChatId = chatId as string;
  return (
    <ChatConversation
      chatId={readyChatId}
      canvasId={canvasId}
      organizationId={organizationId}
      initialStatus={chatQuery.data?.status ?? "idle"}
      refreshChatStatus={refreshChatStatus}
      agentMode={toolSidebarState.agentMode}
      onModeSwitch={toolSidebarState.switchAgentMode}
      isEditing={toolSidebarState.isEditing}
      onAgentStagingReady={toolSidebarState.onAgentStagingReady}
      onAgentStagingCommit={toolSidebarState.onAgentStagingCommit}
      liveCanvasVersionId={toolSidebarState.liveCanvasVersionId}
      headerMode={toolSidebarState.headerMode}
      isRunInspectionMode={toolSidebarState.isRunInspectionMode}
    />
  );
}

function ChatConversation({
  chatId,
  canvasId,
  organizationId,
  initialStatus,
  refreshChatStatus,
  agentMode,
  onModeSwitch,
  isEditing,
  onAgentStagingReady,
  onAgentStagingCommit,
  liveCanvasVersionId,
  headerMode,
  isRunInspectionMode,
}: ChatConversationProps) {
  const messagesQuery = useAgentChatMessages(chatId, organizationId, true);
  const sendMutation = useSendAgentChatMessage(organizationId, canvasId);
  const interruptMutation = useInterruptAgentChat(organizationId);
  const outcomeMutation = useDefineAgentOutcome(organizationId);
  const resetMutation = useResetCanvasAgentChat(organizationId, canvasId);
  const [status, setStatus] = useState<string>(initialStatus || "idle");
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [outcomeState, setOutcomeState] = useStoredOutcomeState(chatId);
  const rawMessages = useConversationMessages(messagesQuery.data);
  const messages = useGreetedMessages(rawMessages, canvasId);

  // chatId is a dep so a /clear session swap resets a stale "streaming" status.
  useEffect(() => {
    setStatus(initialStatus || "idle");
  }, [initialStatus, chatId]);
  useStreamingStatusReconciler(status, setStatus, refreshChatStatus);

  const showThinking = useThinkingIndicator(rawMessages, status);
  useAgentBootKickoff({ messagesQuery, sendMutation, chatId, canvasId, agentMode });
  const handlers = useConversationHandlers({
    agentMode,
    chatId,
    canvasId,
    isBusy: status === "streaming" || outcomeMutation.isPending || isOutcomeActive(outcomeState),
    outcomeMutation,
    interruptMutation,
    resetMutation,
    sendMutation,
    setError,
    setNotice,
    setOutcomeState,
  });

  const wsCallbacks = useMemo(
    () => createWebsocketCallbacks(setStatus, setError, setOutcomeState, setNotice),
    [setOutcomeState],
  );
  useAgentSessionWebsocket(chatId, organizationId, wsCallbacks);

  const scrollRef = useChatScroll(messagesQuery, chatId, messages.length, showThinking);
  const messageGroups = useMemo(() => groupMessages(messages), [messages]);
  const outcomeActive = isOutcomeActive(outcomeState);
  const agentBusy = status === "streaming" || outcomeMutation.isPending || resetMutation.isPending || outcomeActive;

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <ConversationTranscript
        error={error}
        notice={notice}
        canvasId={canvasId}
        organizationId={organizationId}
        messageGroups={messageGroups}
        isLoading={messagesQuery.isLoading}
        isLoadingMore={messagesQuery.isFetchingNextPage}
        onAction={handlers.handleQuickAction}
        onStartBuilding={handlers.handleStartBuilding}
        scrollRef={scrollRef}
        showThinking={showThinking}
      />

      {outcomeState ? (
        <div className="border-t border-slate-200 px-3 py-2">
          <div className="mx-auto w-full max-w-[800px]">
            <OutcomeProgressWidget state={outcomeState} onDismiss={() => setOutcomeState(null)} />
          </div>
        </div>
      ) : null}

      <DraftActionsBar
        messages={messages}
        canvasId={canvasId}
        organizationId={organizationId}
        isEditing={isEditing}
        outcomePassed={outcomeState?.phase === "passed"}
        onVersionPublished={() => setOutcomeState(null)}
        onAgentStagingReady={onAgentStagingReady}
        onAgentStagingCommit={onAgentStagingCommit}
        liveCanvasVersionId={liveCanvasVersionId}
        headerMode={headerMode}
        isRunInspectionMode={isRunInspectionMode}
      />

      <ComposerWithCanvasData
        canvasId={canvasId}
        organizationId={organizationId}
        onSend={handlers.handleSend}
        onStop={handlers.handleStop}
        onClearChat={() => void handlers.handleSend("/clear")}
        clearing={resetMutation.isPending}
        sending={agentBusy}
        sendPending={sendMutation.isPending || resetMutation.isPending}
        stopping={interruptMutation.isPending}
        statusLabel={resolveComposerStatusLabel(resetMutation.isPending, sendMutation.isPending, status)}
        agentMode={agentMode}
        onModeSwitch={onModeSwitch}
        modeDisabled={agentBusy}
      />
    </div>
  );
}

function resolveComposerStatusLabel(resetPending: boolean, sendPending: boolean, status: string): string {
  if (resetPending) return "Clearing chat...";
  if (sendPending) return "Starting agent...";
  return statusLabel(status);
}

function useStreamingStatusReconciler(
  status: string,
  setStatus: (value: string) => void,
  refreshChatStatus: () => Promise<string | undefined>,
) {
  const activeRef = useRef(false);
  const inFlightRef = useRef(false);
  const reconcile = useCallback(async () => {
    if (inFlightRef.current) {
      return;
    }

    inFlightRef.current = true;
    try {
      const nextStatus = await refreshChatStatus();
      if (activeRef.current && nextStatus && nextStatus !== "streaming") {
        setStatus(nextStatus);
      }
    } catch {
      // Websocket events remain the primary status path; refetch only repairs missed terminal events.
    } finally {
      inFlightRef.current = false;
    }
  }, [refreshChatStatus, setStatus]);

  useEffect(() => {
    if (status !== "streaming") {
      return;
    }

    activeRef.current = true;
    void reconcile();
    const intervalId = window.setInterval(() => {
      void reconcile();
    }, STREAMING_STATUS_RECONCILE_INTERVAL_MS);

    return () => {
      activeRef.current = false;
      window.clearInterval(intervalId);
    };
  }, [reconcile, status]);
}

function useAgentBootKickoff({
  messagesQuery,
  sendMutation,
  chatId,
  canvasId,
  agentMode,
}: {
  messagesQuery: ReturnType<typeof useAgentChatMessages>;
  sendMutation: ReturnType<typeof useSendAgentChatMessage>;
  chatId: string;
  canvasId: string;
  agentMode: AgentMode;
}) {
  const [bootReadinessSignal, setBootReadinessSignal] = useState(0);
  const bootState = useRef<"idle" | "sending" | "sent">("idle");

  useEffect(() => {
    const handleBootReady = (event: Event) => {
      const detail = (event as CustomEvent<{ canvasId?: string }>).detail;
      if (detail?.canvasId === canvasId) {
        setBootReadinessSignal((current) => current + 1);
      }
    };

    window.addEventListener(AGENT_BOOT_CONTEXT_READY_EVENT, handleBootReady);
    return () => window.removeEventListener(AGENT_BOOT_CONTEXT_READY_EVENT, handleBootReady);
  }, [canvasId]);

  useEffect(() => {
    if (bootState.current !== "idle") return;
    if (!messagesQuery.data || messagesQuery.isLoading) return;
    if (!isAgentBootReady(canvasId)) return;

    const allMessages = messagesQuery.data.pages?.flatMap((p) => p.messages) ?? [];
    if (allMessages.length > 0) return;

    const bootMessage = getAgentBootMessage(canvasId);

    // If no boot message (e.g. blank canvas), skip sending — static greeting only.
    // Don't clear the boot context so refreshes still see the blank marker and skip again.
    if (!bootMessage) {
      bootState.current = "sent";
      return;
    }

    bootState.current = "sending";
    void sendMutation
      .mutateAsync({
        chatId,
        content: createSystemMessage(bootMessage),
        mode: agentMode,
      })
      .then(() => {
        bootState.current = "sent";
        clearAgentBootContext();
      })
      .catch(() => {
        bootState.current = "idle";
      });
  }, [messagesQuery.data, messagesQuery.isLoading, bootReadinessSignal, chatId, canvasId, agentMode, sendMutation]);
}

function useConversationHandlers({
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
  setOutcomeState: _setOutcomeState,
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
  // React Query mutation objects are new on every render; keep latest refs in
  // a ref so the handler callbacks stay stable across parent re-renders
  // (canvas zoom/pan ticks the parent often). isBusy rides along too.
  const mutationsRef = useRef({ sendMutation, interruptMutation, outcomeMutation, resetMutation });
  mutationsRef.current = { sendMutation, interruptMutation, outcomeMutation, resetMutation };
  const isBusyRef = useRef(isBusy);
  isBusyRef.current = isBusy;

  const handleSend = useCallback(
    async (content: string, images?: AgentOutgoingImage[]) => {
      const trimmed = content.trim();
      const { sendMutation: send, resetMutation: reset } = mutationsRef.current;

      // Handle /clear before the generic send guard below so it isn't silently
      // swallowed while a send is pending; surface a notice instead.
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
          // The backend also guards against a busy session; if local status hadn't
          // caught up yet, show the intentional notice rather than a raw error.
          if (isSessionBusyError(error)) {
            setNotice("Wait for the agent to finish before clearing the chat.");
          } else {
            setError(error instanceof Error ? error.message : "failed to clear chat");
          }
          throw error;
        });
        // Only after a successful reset: scoped so the fresh session can't auto-boot
        // or keep showing this canvas's stale intro. Kept after the await so a failed
        // clear doesn't drop the boot context while the old session is still intact.
        clearAgentBootContextForCanvas(canvasId);
        _setOutcomeState(null);
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
    [agentMode, chatId, canvasId, setError, setNotice, _setOutcomeState],
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
      // Only guard against an in-flight reset here; unlike quick actions this path
      // intentionally doesn't gate on send.isPending (it fires right after a send).
      if (reset.isPending) return;

      // Rubric is a spec confirmation — tell the agent to start building.
      // Always use builder mode regardless of current agentMode.
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

function DraftActionsBar({
  messages,
  canvasId,
  organizationId,
  isEditing,
  outcomePassed,
  onVersionPublished,
  onAgentStagingReady,
  onAgentStagingCommit,
  liveCanvasVersionId,
  headerMode,
  isRunInspectionMode = false,
}: DraftActionsBarProps) {
  const { latestDraft, dismiss } = useDraftActions({
    messages,
    canvasId,
    organizationId,
    outcomePassed,
    onVersionPublished,
  });

  const openedStagingKeyRef = useRef<string | null>(null);
  const onAgentStagingReadyRef = useRef(onAgentStagingReady);
  onAgentStagingReadyRef.current = onAgentStagingReady;

  useEffect(() => {
    if (!latestDraft) {
      openedStagingKeyRef.current = null;
      return;
    }

    if (!onAgentStagingReadyRef.current || !liveCanvasVersionId) {
      return;
    }

    if (!isCanvasWorkflowTab(headerMode) || isRunInspectionMode) {
      return;
    }

    const stagingKey = buildAgentStagingAutoOpenKey(latestDraft.canvasId, latestDraft.message);
    if (openedStagingKeyRef.current === stagingKey || !claimAgentStagingAutoOpen(stagingKey)) {
      return;
    }

    let cancelled = false;

    void Promise.resolve(onAgentStagingReadyRef.current()).then((ready) => {
      if (cancelled || ready === false) {
        releaseAgentStagingAutoOpen(stagingKey);
        return;
      }
      openedStagingKeyRef.current = stagingKey;
    });

    return () => {
      cancelled = true;
    };
  }, [headerMode, isRunInspectionMode, latestDraft, liveCanvasVersionId]);

  if (!latestDraft) return null;

  return (
    <div className="border-t border-slate-200 bg-slate-50/80 px-3 py-2">
      <div className="mx-auto w-full max-w-[800px]">
        <DraftActionsWidget
          versionId={latestDraft.versionId}
          message={latestDraft.message}
          canvasId={canvasId}
          organizationId={organizationId}
          isEditing={isEditing}
          onDismiss={dismiss}
          onViewStaging={onAgentStagingReady}
          onCommitStaging={onAgentStagingCommit}
        />
      </div>
    </div>
  );
}

function ComposerWithCanvasData({
  canvasId,
  organizationId,
  ...composerProps
}: {
  canvasId: string;
  organizationId: string;
  onSend: (content: string, images: AgentOutgoingImage[]) => Promise<void>;
  onStop: () => void;
  onClearChat: () => void;
  clearing: boolean;
  sending: boolean;
  sendPending: boolean;
  stopping?: boolean;
  statusLabel: string;
  agentMode: AgentMode;
  onModeSwitch: (mode: AgentMode) => void;
  modeDisabled?: boolean;
}) {
  const { data: canvas } = useCanvas(organizationId, canvasId, {
    staleTime: Infinity,
    refetchOnWindowFocus: false,
    refetchOnMount: false,
  });

  const nodes = useMemo(() => canvas?.spec?.nodes ?? [], [canvas]);

  const runsQuery = useInfiniteCanvasRuns(canvasId, {}, true);
  const runs = useMemo(() => runsQuery.data?.pages?.flatMap((p) => p?.runs ?? []) ?? [], [runsQuery.data]);

  return <ChatComposer {...composerProps} nodes={nodes} runs={runs} />;
}
