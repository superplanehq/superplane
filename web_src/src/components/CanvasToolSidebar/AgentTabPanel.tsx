import { Loader2 } from "lucide-react";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import type { AgentMode } from "@/components/AgentSidebar/agentMode";
import { AccountContext } from "@/contexts/accountContextState";
import { SessionListView } from "./SessionListView";
import { Users } from "lucide-react";
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
  useCanvasSessions,
  useDefineAgentOutcome,
  useInterruptAgentChat,
  useSendAgentChatMessage,
} from "@/hooks/useAgentChats";
import { useAgentSessionWebsocket } from "@/hooks/useAgentSessionWebsocket";
import { ConversationTranscript } from "./AgentConversationTranscript";
import {
  buildRubricText,
  createInitialOutcomeState,
  createWebsocketCallbacks,
  isOutcomeActive,
  statusLabel,
  useConversationMessages,
  useStoredOutcomeState,
  useThinkingIndicator,
} from "./agentConversationState";
import type { AgentMessage } from "./types";
import type { CanvasToolSidebarState } from "./useCanvasToolSidebarState";
import { groupMessages } from "./agentMessageGroups";

type ChatConversationProps = {
  chatId: string;
  canvasId: string;
  organizationId: string;
  agentMode: AgentMode;
  onModeSwitch: (mode: AgentMode) => void;
  isEditing: boolean;
  readOnly?: boolean;
};

type DraftActionsBarProps = {
  messages: AgentMessage[];
  canvasId: string;
  organizationId: string;
  chatId: string;
  sendMutation: ReturnType<typeof useSendAgentChatMessage>;
  agentMode: AgentMode;
  isEditing: boolean;
  outcomePassed?: boolean;
  onVersionPublished?: () => void;
};

type ConversationHandlers = {
  handleSend: (content: string) => Promise<void>;
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
  const [viewMode, setViewMode] = useState<"my-session" | "session-list" | "viewing-session">("my-session");
  const [viewingSessionId, setViewingSessionId] = useState<string | null>(null);
  const sessionsQuery = useCanvasSessions(canvasId, organizationId, viewMode === "session-list");

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

  if (viewMode === "session-list") {
    return (
      <SessionListView
        sessions={sessionsQuery.data ?? []}
        currentUserId={account?.id ?? ""}
        onSelectSession={(sessionId) => {
          setViewingSessionId(sessionId);
          setViewMode("viewing-session");
        }}
        onBack={() => setViewMode("my-session")}
      />
    );
  }

  if (viewMode === "viewing-session" && viewingSessionId) {
    return (
      <div className="flex min-h-0 flex-1 flex-col">
        <div className="flex items-center gap-2 border-b border-border px-3 py-2">
          <button type="button" onClick={() => setViewMode("session-list")} className="flex items-center gap-1 text-xs text-slate-500 hover:text-slate-700">
            <Users className="size-3" />
            All sessions
          </button>
        </div>
        <ChatConversation
          chatId={viewingSessionId}
          canvasId={canvasId}
          organizationId={organizationId}
          agentMode={toolSidebarState.agentMode}
          onModeSwitch={toolSidebarState.switchAgentMode}
          isEditing={toolSidebarState.isEditing}
          readOnly
        />
      </div>
    );
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="flex items-center justify-end border-b border-border px-3 py-1">
        <button
          type="button"
          onClick={() => setViewMode("session-list")}
          className="flex items-center gap-1 text-[11px] text-slate-400 hover:text-slate-600"
        >
          <Users className="size-3" />
          All sessions
        </button>
      </div>
      <ChatConversation
        chatId={chatId}
        canvasId={canvasId}
        organizationId={organizationId}
        agentMode={toolSidebarState.agentMode}
        onModeSwitch={toolSidebarState.switchAgentMode}
        isEditing={toolSidebarState.isEditing}
      />
    </div>
  );
}

function ChatConversation({
  chatId,
  canvasId,
  organizationId,
  agentMode,
  onModeSwitch,
  isEditing,
  readOnly,
}: ChatConversationProps) {
  const messagesQuery = useAgentChatMessages(chatId, organizationId, true);
  const sendMutation = useSendAgentChatMessage(organizationId, canvasId);
  const interruptMutation = useInterruptAgentChat(organizationId);
  const outcomeMutation = useDefineAgentOutcome(organizationId);
  const [status, setStatus] = useState<string>("idle");
  const [error, setError] = useState<string | null>(null);
  const [outcomeState, setOutcomeState] = useStoredOutcomeState(chatId);
  const rawMessages = useConversationMessages(messagesQuery.data);
  const { account } = useContext(AccountContext);
  const greetingFirstName = account?.name?.split(" ")[0] ?? "there";

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
    return [greeting, ...rawMessages];
  }, [rawMessages, greetingFirstName]);

  const showThinking = useThinkingIndicator(rawMessages, status);

  // Auto-kickoff: send a boot message when session is new (no messages yet)
  const bootState = useRef<"idle" | "sending" | "sent">("idle");
  useEffect(() => {
    if (bootState.current !== "idle") return;
    if (!messagesQuery.data || messagesQuery.isLoading) return;

    const allMessages = messagesQuery.data.pages?.flatMap((p) => p.messages) ?? [];
    if (allMessages.length === 0) {
      bootState.current = "sending";
      void sendMutation
        .mutateAsync({
          chatId,
          content: createSystemMessage(
            "Session ready. Read the current canvas state, check connected integrations, and greet the user.",
          ),
          mode: agentMode,
        })
        .then(() => {
          bootState.current = "sent";
        })
        .catch(() => {
          bootState.current = "idle";
        });
    }
  }, [messagesQuery.data, messagesQuery.isLoading, chatId, agentMode, sendMutation]);
  const handlers = useConversationHandlers({
    agentMode,
    chatId,
    outcomeMutation,
    interruptMutation,
    sendMutation,
    setError,
    setOutcomeState,
  });

  const wsCallbacks = useMemo(() => createWebsocketCallbacks(setStatus, setError, setOutcomeState), [setOutcomeState]);
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
          <OutcomeProgressWidget state={outcomeState} onDismiss={() => setOutcomeState(null)} />
        </div>
      ) : null}

      <DraftActionsBar
        messages={messages}
        canvasId={canvasId}
        organizationId={organizationId}
        chatId={chatId}
        sendMutation={sendMutation}
        agentMode={agentMode}
        isEditing={isEditing}
        outcomePassed={outcomeState?.phase === "passed"}
        onVersionPublished={() => setOutcomeState(null)}
      />

      {!readOnly && (
      <ChatComposer
        onSend={handlers.handleSend}
        onStop={handlers.handleStop}
        sending={agentBusy}
        sendPending={sendMutation.isPending}
        stopping={interruptMutation.isPending}
        statusLabel={statusLabel(status)}
        agentMode={agentMode}
        onModeSwitch={onModeSwitch}
        modeDisabled={modeDisabled}
      />
      )}
    </div>
  );
}

function useConversationHandlers({
  agentMode,
  chatId,
  outcomeMutation,
  interruptMutation,
  sendMutation,
  setError,
  setOutcomeState,
}: {
  agentMode: AgentMode;
  chatId: string;
  outcomeMutation: ReturnType<typeof useDefineAgentOutcome>;
  interruptMutation: ReturnType<typeof useInterruptAgentChat>;
  sendMutation: ReturnType<typeof useSendAgentChatMessage>;
  setError: (value: string | null) => void;
  setOutcomeState: (update: OutcomeState | null | ((prev: OutcomeState | null) => OutcomeState | null)) => void;
}): ConversationHandlers {
  // React Query mutation objects are new on every render; keep latest refs in
  // a ref so the handler callbacks stay stable across parent re-renders
  // (canvas zoom/pan ticks the parent often).
  const mutationsRef = useRef({ sendMutation, interruptMutation, outcomeMutation });
  mutationsRef.current = { sendMutation, interruptMutation, outcomeMutation };

  const handleSend = useCallback(
    async (content: string) => {
      const { sendMutation: send } = mutationsRef.current;
      if (!content.trim() || send.isPending) return;
      setError(null);
      await send.mutateAsync({ chatId, content, mode: agentMode }).catch((error) => {
        setError(error instanceof Error ? error.message : "failed to send message");
        throw error;
      });
    },
    [agentMode, chatId, setError],
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
    async (rubric: { title: string; criteria: string[]; categories?: RubricCategory[] }) => {
      const rubricText = buildRubricText(rubric);
      const { sendMutation: send, outcomeMutation: outcome } = mutationsRef.current;

      // In Build mode: rubric is a spec confirmation, not an outcome.
      // Agent already has full context — just confirm.
      if (agentMode === "builder") {
        await send.mutateAsync({
          chatId,
          content: "Specs approved. Start building.",
          mode: "builder",
        });
        return;
      }

      // In Plan mode: kick off outcome with grading loop
      setOutcomeState(createInitialOutcomeState(rubric));

      try {
        await outcome.mutateAsync({
          chatId,
          description: `Build a canvas based on this plan: ${rubric.title}`,
          rubric: rubricText,
          maxIterations: 3,
        });
      } catch {
        setOutcomeState(null);
        await send.mutateAsync({
          chatId,
          content: `Start building based on this plan:\n\n${rubricText}`,
          mode: "builder",
        });
      }
    },
    [chatId, agentMode, setOutcomeState],
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
  chatId,
  sendMutation,
  agentMode,
  isEditing,
  outcomePassed,
  onVersionPublished,
}: DraftActionsBarProps) {
  const { latestDraft, dismiss } = useDraftActions({
    messages,
    canvasId,
    organizationId,
    chatId,
    sendMutation,
    agentMode,
    outcomePassed,
    onVersionPublished,
  });

  if (!latestDraft) return null;

  return (
    <div className="border-t border-slate-200 bg-slate-50/80 px-3 py-2">
      <DraftActionsWidget
        versionId={latestDraft.versionId}
        message={latestDraft.message}
        canvasId={canvasId}
        organizationId={organizationId}
        isEditing={isEditing}
        onDismiss={dismiss}
      />
    </div>
  );
}
