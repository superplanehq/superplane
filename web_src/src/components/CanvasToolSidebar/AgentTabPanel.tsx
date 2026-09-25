import { useCallback, useContext, useEffect, useMemo, useState } from "react";
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

type ChatConversationProps = {
  chatId: string;
  canvasId: string;
  organizationId: string;
  initialStatus: string;
  refreshChatStatus: () => Promise<string | undefined>;
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
  const showThinking = useThinkingIndicator(rawMessages, status);
  useAgentChatBootKickoff({ messagesQuery, sendMutation, chatId, canvasId, isAutoLayoutOnUpdateEnabled });
  const handlers = useAgentConversationHandlers({
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

  const reconcileStreamingStatus = useCallback(async () => {
    if (status !== "streaming") {
      return;
    }
    try {
      const nextStatus = await refreshChatStatus();
      if (nextStatus && nextStatus !== "streaming") {
        setStatus(nextStatus);
      }
    } catch {
      // Live events remain authoritative. A later reconnect retries recovery.
    }
  }, [refreshChatStatus, status]);
  const wsCallbacks = useMemo(
    () => ({
      ...createWebsocketCallbacks(setStatus, setError, setOutcomeState, setNotice),
      onConnectionOpen: () => void reconcileStreamingStatus(),
    }),
    [reconcileStreamingStatus, setOutcomeState],
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
      />
    </div>
  );
}

function resolveComposerStatusLabel(resetPending: boolean, sendPending: boolean, status: string): string {
  if (resetPending) return "Clearing chat...";
  if (sendPending) return "Starting agent...";
  return statusLabel(status);
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
}) {
  const { data: canvas } = useCanvas(organizationId, canvasId, {
    staleTime: Infinity,
    refetchOnWindowFocus: false,
    refetchOnMount: false,
  });

  const nodes = useMemo(() => canvas?.spec?.nodes ?? [], [canvas]);

  const runsQuery = useInfiniteCanvasRuns(canvasId, {}, true);
  const runs = useMemo(() => runsQuery.data?.pages?.flatMap((p) => p?.runs ?? []) ?? [], [runsQuery.data]);

  return (
    <ChatComposer
      {...composerProps}
      canvasId={canvasId}
      organizationId={organizationId}
      factoryId={canvas?.metadata?.factoryId ?? ""}
      nodes={nodes}
      runs={runs}
    />
  );
}
