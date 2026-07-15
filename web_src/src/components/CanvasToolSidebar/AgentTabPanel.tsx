import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import type { AgentMode } from "@/components/AgentSidebar/agentMode";
import { AccountContext } from "@/contexts/accountContextState";
import { ChatComposer } from "@/components/AgentSidebar/ChatComposer";
import { useChatScroll } from "@/components/AgentSidebar/useChatScroll";
import { OutcomeProgressWidget } from "@/components/AgentSidebar/widgets/OutcomeProgressWidget";
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
import type { CanvasPageHeaderMode } from "@/pages/app/viewState";
import { ConversationTranscript } from "./AgentConversationTranscript";
import { StagingActionsBar } from "./StagingActionsBar";
import {
  createWebsocketCallbacks,
  isOutcomeActive,
  statusLabel,
  useConversationMessages,
  useStoredOutcomeState,
  useThinkingIndicator,
} from "./agentConversationState";
import type { AgentOutgoingImage } from "./types";
import type { AgentStagingReadyHandler, CanvasToolSidebarState } from "./useCanvasToolSidebarState";
import { groupMessages } from "./agentMessageGroups";
import { useGreetedMessages } from "./useGreetedMessages";
import { AgentSetupNotice } from "./AgentSetupState";
import { getAgentSetupState } from "./agentSetupStateModel";
import { useAgentChatBootKickoff } from "./useAgentChatBootKickoff";
import { useAgentConversationHandlers } from "./useAgentConversationHandlers";

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
  isAutoLayoutOnUpdateEnabled: boolean;
  onAgentStagingReady?: AgentStagingReadyHandler;
  onAgentStagingCommit?: (commitMessage: string) => Promise<boolean>;
  liveCanvasVersionId?: string;
  headerMode?: CanvasPageHeaderMode;
  isRunInspectionMode?: boolean;
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
      isAutoLayoutOnUpdateEnabled={toolSidebarState.isAutoLayoutOnUpdateEnabled}
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
  isAutoLayoutOnUpdateEnabled,
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

  useEffect(() => {
    setStatus(initialStatus || "idle");
  }, [initialStatus, chatId]);
  useStreamingStatusReconciler(status, setStatus, refreshChatStatus);

  const showThinking = useThinkingIndicator(rawMessages, status);
  useAgentChatBootKickoff({ messagesQuery, sendMutation, chatId, canvasId, agentMode, isAutoLayoutOnUpdateEnabled });
  const handlers = useAgentConversationHandlers({
    agentMode,
    chatId,
    canvasId,
    isAutoLayoutOnUpdateEnabled,
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
        <div className="border-t border-slate-200 px-3 py-2 dark:border-gray-800/70">
          <div className="mx-auto w-full max-w-[800px]">
            <OutcomeProgressWidget state={outcomeState} onDismiss={() => setOutcomeState(null)} />
          </div>
        </div>
      ) : null}

      <StagingActionsBar
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
