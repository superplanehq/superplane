import { Loader2 } from "lucide-react";
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
  useSendAgentChatMessage,
} from "@/hooks/useAgentChats";
import { useAgentSessionWebsocket } from "@/hooks/useAgentSessionWebsocket";
import { useCanvas, useCanvasVersion, useCanvasVersions, useInfiniteCanvasRuns } from "@/hooks/useCanvasData";
import {
  AGENT_BOOT_CONTEXT_READY_EVENT,
  clearAgentBootContext,
  getAgentBootInitialMessage,
  getAgentBootMessage,
  isAgentBootReady,
} from "@/lib/agentBootContext";
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
import type { CanvasToolSidebarState } from "./useCanvasToolSidebarState";
import { groupMessages } from "./agentMessageGroups";

type ChatConversationProps = {
  chatId: string;
  canvasId: string;
  organizationId: string;
  initialStatus: string;
  agentMode: AgentMode;
  onModeSwitch: (mode: AgentMode) => void;
  isEditing: boolean;
};

type DraftActionsBarProps = {
  messages: AgentMessage[];
  canvasId: string;
  organizationId: string;
  isEditing: boolean;
  outcomePassed?: boolean;
  onVersionPublished?: () => void;
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

  if (chatQuery.isLoading || !chatId) {
    return (
      <div className="flex min-h-0 flex-1 flex-col">
        <div className="flex-1 overflow-y-auto p-3">
          <div className="flex flex-col items-start">
            <div className="max-w-[85%] break-words rounded-lg px-3 py-2 text-sm bg-slate-100 text-slate-900">
              Hi {firstName}! I'm your SuperPlane agent. Give me a moment to set up and I'll help you build.
              <div className="mt-2 flex items-center gap-2 text-xs text-slate-400">
                <Loader2 className="size-3 animate-spin" /> Setting up...
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <ChatConversation
      chatId={chatId}
      canvasId={canvasId}
      organizationId={organizationId}
      initialStatus={chatQuery.data?.status ?? "idle"}
      agentMode={toolSidebarState.agentMode}
      onModeSwitch={toolSidebarState.switchAgentMode}
      isEditing={toolSidebarState.isEditing}
    />
  );
}

function ChatConversation({
  chatId,
  canvasId,
  organizationId,
  initialStatus,
  agentMode,
  onModeSwitch,
  isEditing,
}: ChatConversationProps) {
  const messagesQuery = useAgentChatMessages(chatId, organizationId, true);
  const sendMutation = useSendAgentChatMessage(organizationId, canvasId);
  const interruptMutation = useInterruptAgentChat(organizationId);
  const outcomeMutation = useDefineAgentOutcome(organizationId);
  const [status, setStatus] = useState<string>(initialStatus || "idle");
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [outcomeState, setOutcomeState] = useStoredOutcomeState(chatId);
  const rawMessages = useConversationMessages(messagesQuery.data);
  const { account } = useContext(AccountContext);
  const greetingFirstName = account?.name?.split(" ")[0] ?? "there";
  const bootInitialMessage = useMemo(() => getAgentBootInitialMessage(canvasId), [canvasId]);

  useEffect(() => {
    setStatus(initialStatus || "idle");
  }, [initialStatus]);

  // Prepend a synthetic greeting as the first message so it never disappears
  const messages = useMemo(() => {
    const greeting: AgentMessage = {
      id: "__greeting__",
      role: "assistant",
      content: `Hi ${greetingFirstName}! I'm your SuperPlane agent. I'll help you build and modify this canvas.`,
      createdAt: rawMessages[0]?.createdAt ?? null,
      toolCallId: "",
      toolName: "",
      toolStatus: "",
    };

    if (!bootInitialMessage) {
      return [greeting, ...rawMessages];
    }

    const templateIntro: AgentMessage = {
      id: "__boot_initial_message__",
      role: "assistant",
      content: bootInitialMessage,
      createdAt: rawMessages[0]?.createdAt ?? null,
      toolCallId: "",
      toolName: "",
      toolStatus: "",
    };

    return [greeting, templateIntro, ...rawMessages];
  }, [rawMessages, greetingFirstName, bootInitialMessage]);

  const showThinking = useThinkingIndicator(rawMessages, status);
  useAgentBootKickoff({ messagesQuery, sendMutation, chatId, canvasId, agentMode });
  const handlers = useConversationHandlers({
    agentMode,
    chatId,
    outcomeMutation,
    interruptMutation,
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
  const agentBusy = status === "streaming" || outcomeMutation.isPending || outcomeActive;
  const modeDisabled = agentBusy;

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
      />

      <ComposerWithCanvasData
        canvasId={canvasId}
        organizationId={organizationId}
        onSend={handlers.handleSend}
        onStop={handlers.handleStop}
        sending={agentBusy}
        sendPending={sendMutation.isPending}
        stopping={interruptMutation.isPending}
        statusLabel={sendMutation.isPending ? "Starting agent..." : statusLabel(status)}
        agentMode={agentMode}
        onModeSwitch={onModeSwitch}
        modeDisabled={modeDisabled}
      />
    </div>
  );
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
  outcomeMutation,
  interruptMutation,
  sendMutation,
  setError,
  setNotice,
  setOutcomeState: _setOutcomeState,
}: {
  agentMode: AgentMode;
  chatId: string;
  outcomeMutation: ReturnType<typeof useDefineAgentOutcome>;
  interruptMutation: ReturnType<typeof useInterruptAgentChat>;
  sendMutation: ReturnType<typeof useSendAgentChatMessage>;
  setError: (value: string | null) => void;
  setNotice: (value: string | null) => void;
  setOutcomeState: (update: OutcomeState | null | ((prev: OutcomeState | null) => OutcomeState | null)) => void;
}): ConversationHandlers {
  // React Query mutation objects are new on every render; keep latest refs in
  // a ref so the handler callbacks stay stable across parent re-renders
  // (canvas zoom/pan ticks the parent often).
  const mutationsRef = useRef({ sendMutation, interruptMutation, outcomeMutation });
  mutationsRef.current = { sendMutation, interruptMutation, outcomeMutation };

  const handleSend = useCallback(
    async (content: string, images?: AgentOutgoingImage[]) => {
      const { sendMutation: send } = mutationsRef.current;
      if ((!content.trim() && (images?.length ?? 0) === 0) || send.isPending) return;
      setError(null);
      setNotice(null);
      await send.mutateAsync({ chatId, content, mode: agentMode, images }).catch((error) => {
        setError(error instanceof Error ? error.message : "failed to send message");
        throw error;
      });
    },
    [agentMode, chatId, setError, setNotice],
  );

  const handleStop = useCallback(() => {
    mutationsRef.current.interruptMutation.mutate({ chatId });
  }, [chatId]);

  const handleQuickAction = useCallback(
    async (action: string) => {
      const { sendMutation: send } = mutationsRef.current;
      if (send.isPending) return;
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
      const { sendMutation: send } = mutationsRef.current;

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
}: DraftActionsBarProps) {
  const { latestDraft, dismiss } = useDraftActions({
    messages,
    canvasId,
    organizationId,
    outcomePassed,
    onVersionPublished,
  });

  useEffect(() => {
    if (!latestDraft) {
      return;
    }

    window.dispatchEvent(new CustomEvent("agent:draft-ready", { detail: { versionId: latestDraft.versionId } }));
  }, [canvasId, latestDraft]);

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
  const { data: versions } = useCanvasVersions(organizationId, canvasId);
  const latestVersion = versions?.[0];
  const isDraft = latestVersion?.metadata?.state === "STATE_DRAFT";
  const { data: draftVersion } = useCanvasVersion(organizationId, canvasId, latestVersion?.metadata?.id ?? "", isDraft);

  // Use draft nodes if a draft exists, otherwise fall back to published
  const nodes = useMemo(
    () => (isDraft && draftVersion?.spec?.nodes ? draftVersion.spec.nodes : canvas?.spec?.nodes) ?? [],
    [isDraft, draftVersion, canvas],
  );

  const runsQuery = useInfiniteCanvasRuns(canvasId, {}, true);
  const runs = useMemo(() => runsQuery.data?.pages?.flatMap((p) => p?.runs ?? []) ?? [], [runsQuery.data]);

  return <ChatComposer {...composerProps} nodes={nodes} runs={runs} />;
}
