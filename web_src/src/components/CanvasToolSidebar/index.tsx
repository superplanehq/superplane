import { Bot, ChevronRight, Loader2, SquareTerminal } from "lucide-react";
import { useCallback, useEffect, useMemo, useState, type ReactNode } from "react";
import type { AgentMode } from "@/components/AgentSidebar/agentMode";
import { ChatComposer } from "@/components/AgentSidebar/ChatComposer";
import { isSystemNotification, formatSystemNotification } from "@/components/AgentSidebar/systemMessages";
import { useChatScroll } from "@/components/AgentSidebar/useChatScroll";
import { useDraftActions } from "@/components/AgentSidebar/useDraftActions";
import { DraftActionsWidget } from "@/components/AgentSidebar/widgets/DraftActionsWidget";
import {
  OutcomeProgressWidget,
  type OutcomeState,
  type OutcomePhase,
  type IterationEntry,
  type GradingEntry,
} from "@/components/AgentSidebar/widgets/OutcomeProgressWidget";
import type { RubricCategory } from "@/components/AgentSidebar/widgets/parser";
import { RichMessage } from "@/components/AgentSidebar/widgets/RichMessage";
import {
  useAgentChatMessages,
  useCanvasAgentChat,
  useDefineAgentOutcome,
  useInterruptAgentChat,
  useSendAgentChatMessage,
} from "@/hooks/useAgentChats";
import { useAgentSessionWebsocket } from "@/hooks/useAgentSessionWebsocket";
import { cn } from "@/lib/utils";
import { EmptyToolTab } from "./EmptyToolTab";
import { SidebarShell } from "./SidebarShell";
import { ToolTabsHeader } from "./ToolTabsHeader";
import type { AgentMessage } from "./types";
import type { CanvasToolSidebarState } from "./useCanvasToolSidebarState";

const TAB_AGENT = "agent",
  TAB_RUNS = "runs",
  TAB_VERSIONS = "versions";

type CanvasToolSidebarMode = "default" | "version-live" | "version-edit" | "runs" | "dashboard";

type OutcomeEvaluationPayload = {
  iteration: number;
  result?: string;
  explanation?: string;
};

export interface CanvasToolSidebarProps {
  toolSidebarState: CanvasToolSidebarState;
  mode?: CanvasToolSidebarMode;
  onSelectRuns?: () => void;
  onExitRunsMode?: () => void;
  runsContent?: ReactNode;
  isVersionControlOpen?: boolean;
  onToggleVersionControl?: () => void;
  versionsContent?: ReactNode;
}

export function CanvasToolSidebar({
  toolSidebarState,
  mode = "default",
  onSelectRuns,
  onExitRunsMode,
  runsContent,
  isVersionControlOpen,
  onToggleVersionControl,
  versionsContent,
}: CanvasToolSidebarProps) {
  if (!toolSidebarState.showToolSidebarToggle || !toolSidebarState.isToolSidebarOpen || !toolSidebarState.canvasId) {
    return null;
  }

  return (
    <OpenCanvasToolSidebar
      toolSidebarState={toolSidebarState}
      mode={mode}
      onSelectRuns={onSelectRuns}
      onExitRunsMode={onExitRunsMode}
      runsContent={runsContent}
      isVersionControlOpen={isVersionControlOpen}
      onToggleVersionControl={onToggleVersionControl}
      versionsContent={versionsContent}
    />
  );
}

function OpenCanvasToolSidebar({
  toolSidebarState,
  mode = "default",
  onSelectRuns,
  onExitRunsMode,
  runsContent,
  isVersionControlOpen,
  onToggleVersionControl,
  versionsContent,
}: CanvasToolSidebarProps) {
  const showRunsTab = Boolean(onSelectRuns || mode === "runs" || runsContent);
  const showVersionsTab = Boolean(onToggleVersionControl || isVersionControlOpen || versionsContent);
  const [activeTab, setActiveTab] = useState(() => {
    if (mode === "runs" && showRunsTab) return TAB_RUNS;
    if (isVersionControlOpen && showVersionsTab) return TAB_VERSIONS;
    return TAB_AGENT;
  });

  useEffect(() => {
    if (mode === "runs" && showRunsTab) {
      setActiveTab(TAB_RUNS);
      return;
    }
    if (isVersionControlOpen && showVersionsTab) {
      setActiveTab(TAB_VERSIONS);
      return;
    }
    setActiveTab((currentTab) => (currentTab === TAB_RUNS || currentTab === TAB_VERSIONS ? TAB_AGENT : currentTab));
  }, [isVersionControlOpen, mode, showRunsTab, showVersionsTab]);

  const tabs = [
    { value: TAB_AGENT, label: "Agent" },
    ...(showRunsTab ? ([{ value: TAB_RUNS, label: "Runs" }] as const) : []),
    ...(showVersionsTab ? ([{ value: TAB_VERSIONS, label: "Versions" }] as const) : []),
  ] as const;

  const handleClose = useCallback(() => {
    if (activeTab === TAB_RUNS) {
      onExitRunsMode?.();
    }

    if (activeTab === TAB_VERSIONS && isVersionControlOpen) {
      onToggleVersionControl?.();
    }

    toolSidebarState.closeToolSidebar();
  }, [activeTab, isVersionControlOpen, onExitRunsMode, onToggleVersionControl, toolSidebarState]);

  const handleTabSelect = useCallback(
    (nextTab: typeof TAB_AGENT | typeof TAB_RUNS | typeof TAB_VERSIONS) => {
      setActiveTab(nextTab);

      if (nextTab === TAB_RUNS) {
        if (isVersionControlOpen) onToggleVersionControl?.();
        if (mode !== "runs") {
          toolSidebarState.openToolSidebar();
          onSelectRuns?.();
        }
        return;
      }

      if (mode === "runs") onExitRunsMode?.();

      if (nextTab === TAB_VERSIONS) {
        if (!isVersionControlOpen) {
          toolSidebarState.openToolSidebar();
          onToggleVersionControl?.();
        }
        return;
      }

      if (isVersionControlOpen) onToggleVersionControl?.();
    },
    [isVersionControlOpen, mode, onExitRunsMode, onSelectRuns, onToggleVersionControl, toolSidebarState],
  );

  return (
    <SidebarShell>
      <div className="flex min-h-0 flex-1 flex-col gap-0">
        <ToolTabsHeader
          tabs={tabs}
          activeTab={activeTab}
          onSelectTab={(value) => handleTabSelect(value as typeof TAB_AGENT | typeof TAB_RUNS | typeof TAB_VERSIONS)}
          onClose={handleClose}
        />

        {activeTab === TAB_AGENT ? (
          <div className="m-0 flex min-h-0 flex-1 flex-col overflow-hidden" role="tabpanel">
            <AgentTabPanel toolSidebarState={toolSidebarState} />
          </div>
        ) : null}
        {activeTab === TAB_RUNS ? (
          <div className="m-0 flex min-h-0 flex-1 flex-col" role="tabpanel">
            {runsContent ?? <EmptyToolTab />}
          </div>
        ) : null}
        {activeTab === TAB_VERSIONS ? (
          <div className="m-0 flex min-h-0 flex-1 flex-col" role="tabpanel">
            {versionsContent ?? <EmptyToolTab />}
          </div>
        ) : null}
      </div>
    </SidebarShell>
  );
}

function applyOutcomeStart(prev: OutcomeState): OutcomeState {
  const updatedLog = [...prev.log];
  const lastEntry = updatedLog[updatedLog.length - 1];

  if (lastEntry && "phase" in lastEntry && (lastEntry as IterationEntry).phase === "building") {
    updatedLog[updatedLog.length - 1] = { phase: "finished" };
  }

  updatedLog.push({ phase: "grading" });
  return {
    ...prev,
    phase: "grading" as OutcomePhase,
    log: updatedLog,
  };
}

function updateGradingLog(log: OutcomeState["log"], evaluation: OutcomeEvaluationPayload): OutcomeState["log"] {
  const updatedLog = [...log];

  for (let index = updatedLog.length - 1; index >= 0; index--) {
    const entry = updatedLog[index] as GradingEntry;
    if (entry.phase !== "grading") {
      continue;
    }

    updatedLog[index] = {
      phase: evaluation.result === "satisfied" ? "satisfied" : "needs_revision",
      explanation: evaluation.explanation,
    };
    break;
  }

  return updatedLog;
}

function applyOutcomeEnd(prev: OutcomeState, evaluation: OutcomeEvaluationPayload): OutcomeState {
  if (!evaluation.result) {
    return prev;
  }

  const updatedLog = updateGradingLog(prev.log, evaluation);

  switch (evaluation.result) {
    case "satisfied":
      return { ...prev, phase: "passed" as OutcomePhase, log: updatedLog };
    case "max_iterations_reached":
      return { ...prev, phase: "exhausted" as OutcomePhase, log: updatedLog };
    case "failed":
    case "interrupted":
      return { ...prev, phase: "failed" as OutcomePhase, log: updatedLog };
  }

  const nextIteration = prev.iteration + 1;
  if (nextIteration > prev.maxIterations) {
    return { ...prev, phase: "exhausted" as OutcomePhase, log: updatedLog };
  }

  updatedLog.push({ phase: "building" });
  return {
    ...prev,
    iteration: nextIteration,
    phase: "building" as OutcomePhase,
    log: updatedLog,
  };
}

function AgentTabPanel({ toolSidebarState }: { toolSidebarState: CanvasToolSidebarState }) {
  const canvasId = toolSidebarState.canvasId ?? "";
  const organizationId = toolSidebarState.organizationId ?? "";
  const chatQuery = useCanvasAgentChat(canvasId, organizationId, toolSidebarState.isToolSidebarOpen);
  const chatId = chatQuery.data?.id ?? null;

  if (chatQuery.isLoading || !chatId) {
    return (
      <div className="flex flex-1 items-center justify-center py-8 text-sm text-muted-foreground">
        <Loader2 className="mr-2 size-4 animate-spin" /> Loading…
      </div>
    );
  }

  return (
    <ChatConversation
      chatId={chatId}
      canvasId={canvasId}
      organizationId={organizationId}
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
  agentMode,
  onModeSwitch,
  isEditing,
}: {
  chatId: string;
  canvasId: string;
  organizationId: string;
  agentMode: AgentMode;
  onModeSwitch: (mode: AgentMode) => void;
  isEditing: boolean;
}) {
  const messagesQuery = useAgentChatMessages(chatId, organizationId, true);
  const sendMutation = useSendAgentChatMessage(organizationId, canvasId);
  const interruptMutation = useInterruptAgentChat(organizationId);
  const outcomeMutation = useDefineAgentOutcome(organizationId);

  const [draft, setDraft] = useState("");
  const [status, setStatus] = useState<string>("idle");
  const [error, setError] = useState<string | null>(null);
  const [outcomeState, setOutcomeStateRaw] = useState<OutcomeState | null>(() => {
    try {
      const stored = sessionStorage.getItem(`outcome-${chatId}`);
      return stored ? JSON.parse(stored) : null;
    } catch {
      return null;
    }
  });
  const setOutcomeState = useCallback(
    (update: OutcomeState | null | ((prev: OutcomeState | null) => OutcomeState | null)) => {
      setOutcomeStateRaw((prev) => {
        const next = typeof update === "function" ? update(prev) : update;
        if (next) {
          sessionStorage.setItem(`outcome-${chatId}`, JSON.stringify(next));
        } else {
          sessionStorage.removeItem(`outcome-${chatId}`);
        }
        return next;
      });
    },
    [chatId],
  );

  const messages = useMemo(
    () =>
      messagesQuery.data?.pages
        .slice()
        .reverse()
        .flatMap((page) => page.messages) ?? [],
    [messagesQuery.data],
  );
  const hasRunningTool = useMemo(
    () => messages.some((message) => message.role === "tool" && message.toolStatus === "started"),
    [messages],
  );
  const showThinking = status === "streaming" && !hasRunningTool && messages[messages.length - 1]?.role === "user";

  const wsCallbacks = useMemo(
    () => ({
      onPersistedMessage: (message: AgentMessage) => {
        if (message.content?.includes("published") || message.content?.includes("discarded")) {
          setOutcomeState(null);
        }
      },
      onStatusChange: (next: string, err?: string) => {
        setStatus(next || "idle");
        setError(err ?? null);
      },
      onOutcomeEvent: (phase: "start" | "end", evaluation: OutcomeEvaluationPayload) => {
        setOutcomeState((prev) => {
          if (!prev) return prev;
          return phase === "start" ? applyOutcomeStart(prev) : applyOutcomeEnd(prev, evaluation);
        });
      },
    }),
    [setOutcomeState],
  );
  useAgentSessionWebsocket(chatId, organizationId, wsCallbacks);

  const handleSend = useCallback(async () => {
    const value = draft.trim();
    if (!value || sendMutation.isPending) return;
    setDraft("");
    setError(null);
    try {
      await sendMutation.mutateAsync({ chatId, content: value, mode: agentMode });
    } catch (err) {
      setError(err instanceof Error ? err.message : "failed to send message");
    }
  }, [agentMode, chatId, draft, sendMutation]);

  const handleStop = useCallback(() => {
    interruptMutation.mutate({ chatId });
  }, [chatId, interruptMutation]);

  const handleStartBuilding = useCallback(
    async (rubric: { title: string; criteria: string[]; categories?: RubricCategory[] }) => {
      const rubricText =
        rubric.categories && rubric.categories.length > 0
          ? `# ${rubric.title}\n\n${rubric.categories.map((category) => `## ${category.heading}\n${category.criteria.map((criterion) => `- ${criterion.text}`).join("\n")}`).join("\n\n")}`
          : `# ${rubric.title}\n\n${rubric.criteria.map((criterion) => `- ${criterion}`).join("\n")}`;

      setOutcomeState({
        title: rubric.title,
        criteria: rubric.criteria.map((criterion) => ({ text: criterion })),
        categories: rubric.categories,
        iteration: 1,
        maxIterations: 3,
        phase: "building",
        log: [{ phase: "building" }],
      });

      try {
        await outcomeMutation.mutateAsync({
          chatId,
          description: `Build a canvas based on this plan: ${rubric.title}`,
          rubric: rubricText,
          maxIterations: 3,
        });
      } catch {
        setOutcomeState(null);
        await sendMutation.mutateAsync({
          chatId,
          content: `Start building based on this plan:\n\n${rubricText}`,
          mode: "builder",
        });
      }
    },
    [chatId, outcomeMutation, sendMutation, setOutcomeState],
  );

  const scrollRef = useChatScroll(messagesQuery, chatId, messages.length, showThinking);
  const messageGroups = useMemo(() => groupMessages(messages), [messages]);

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div ref={scrollRef} className="min-h-0 flex-1 space-y-2 overflow-y-auto p-3" data-testid="agent-chat-messages">
        {messagesQuery.isLoading ? (
          <div className="flex items-center justify-center py-8 text-sm text-muted-foreground">
            <Loader2 className="mr-2 size-4 animate-spin" /> Loading…
          </div>
        ) : (
          <>
            {messagesQuery.isFetchingNextPage ? (
              <div className="flex items-center justify-center py-2 text-xs text-muted-foreground">
                <Loader2 className="mr-2 size-3 animate-spin" /> Loading older messages…
              </div>
            ) : null}
            {messageGroups.map((group) =>
              group.type === "tool-group" ? (
                <ToolGroupRow key={group.messages[0].id} messages={group.messages} />
              ) : group.type === "subagent-group" ? (
                <SubagentCard key={group.messages[0].id} messages={group.messages} />
              ) : (
                <MessageRow
                  key={group.message.id}
                  message={group.message}
                  sendMutation={sendMutation}
                  chatId={chatId}
                  canvasId={canvasId}
                  organizationId={organizationId}
                  agentMode={agentMode}
                  onStartBuilding={handleStartBuilding}
                />
              ),
            )}
          </>
        )}
        {showThinking ? <ThinkingRow /> : null}
        {error ? <p className="px-3 py-2 text-sm text-red-600">{error}</p> : null}
      </div>

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

      <ChatComposer
        draft={draft}
        onDraftChange={setDraft}
        onSend={handleSend}
        onStop={handleStop}
        sending={status === "streaming"}
        stopping={interruptMutation.isPending}
        statusLabel={statusLabel(status)}
        agentMode={agentMode}
        onModeSwitch={onModeSwitch}
        modeDisabled={
          status === "streaming" ||
          (outcomeState != null && outcomeState.phase !== "passed" && outcomeState.phase !== "exhausted")
        }
      />
    </div>
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
}: {
  messages: AgentMessage[];
  canvasId: string;
  organizationId: string;
  chatId: string;
  sendMutation: ReturnType<typeof useSendAgentChatMessage>;
  agentMode: AgentMode;
  isEditing: boolean;
  outcomePassed?: boolean;
  onVersionPublished?: () => void;
}) {
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
    <div className="border-t border-violet-200 bg-violet-50/80 px-3 py-2">
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

function MessageRow({
  message,
  sendMutation,
  chatId,
  canvasId,
  organizationId,
  agentMode,
  onStartBuilding,
}: {
  message: AgentMessage;
  sendMutation: ReturnType<typeof useSendAgentChatMessage>;
  chatId: string;
  canvasId: string;
  organizationId: string;
  agentMode: AgentMode;
  onStartBuilding: (rubric: { title: string; criteria: string[]; categories?: RubricCategory[] }) => void;
}) {
  const handleAction = useCallback(
    async (action: string) => {
      if (sendMutation.isPending) return;
      try {
        await sendMutation.mutateAsync({ chatId, content: action, mode: agentMode });
      } catch {
        // Keep the current transcript unchanged when quick actions fail.
      }
    },
    [agentMode, chatId, sendMutation],
  );

  if (message.role === "tool") {
    return <ToolMessageRow message={message} />;
  }

  if (message.role === "system" || (message.role === "user" && isSystemNotification(message.content))) {
    const text = message.role === "system" ? formatSystemNotification(message.content) : message.content;
    return (
      <div className="flex justify-center">
        <span className="px-2 text-[11px] italic text-slate-400">{text}</span>
      </div>
    );
  }

  const isUser = message.role === "user";

  return (
    <div className={cn("flex flex-col", isUser ? "items-end" : "items-start")}>
      <div
        className={cn(
          "max-w-[85%] break-words rounded-lg px-3 py-2 text-sm",
          isUser ? "bg-violet-600 text-white whitespace-pre-wrap" : "bg-slate-100 text-slate-900",
        )}
        data-testid={isUser ? "agent-user-message" : "agent-assistant-message"}
      >
        {isUser ? (
          message.content
        ) : (
          <RichMessage
            content={message.content}
            onAction={handleAction}
            onStartBuilding={onStartBuilding}
            canvasId={canvasId}
            organizationId={organizationId}
          />
        )}
      </div>
      {message.createdAt ? (
        <span className="mt-0.5 px-1 text-[10px] text-slate-400">{formatTime(message.createdAt)}</span>
      ) : null}
    </div>
  );
}

type MessageGroup =
  | { type: "message"; message: AgentMessage }
  | { type: "tool-group"; messages: AgentMessage[] }
  | { type: "subagent-group"; messages: AgentMessage[] };

function groupMessages(messages: AgentMessage[]): MessageGroup[] {
  const groups: MessageGroup[] = [];
  let toolBuffer: AgentMessage[] = [];
  let subagentBuffer: AgentMessage[] = [];

  function flushTools() {
    if (toolBuffer.length > 0) {
      groups.push({ type: "tool-group", messages: [...toolBuffer] });
      toolBuffer = [];
    }
  }

  function flushSubagents() {
    if (subagentBuffer.length > 0) {
      groups.push({ type: "subagent-group", messages: [...subagentBuffer] });
      subagentBuffer = [];
    }
  }

  for (const message of messages) {
    if (message.role === "tool" && message.toolName?.startsWith("subagent:")) {
      flushTools();
      if (shouldStartNewSubagentGroup(subagentBuffer, message)) {
        flushSubagents();
      }
      subagentBuffer.push(message);
      continue;
    }

    if (message.role === "tool") {
      flushSubagents();
      toolBuffer.push(message);
      continue;
    }

    flushTools();
    flushSubagents();
    groups.push({ type: "message", message });
  }

  flushTools();
  flushSubagents();
  return groups;
}

function shouldStartNewSubagentGroup(buffer: AgentMessage[], message: AgentMessage): boolean {
  if (buffer.length === 0) {
    return false;
  }

  if (buffer[0]?.toolName !== message.toolName) {
    return true;
  }

  const hasStarted = buffer.some((entry) => entry.toolStatus === "started");
  const hasFinished = buffer.some((entry) => entry.toolStatus === "finished");

  if (hasStarted && hasFinished) {
    return true;
  }

  if (message.toolStatus === "started" && hasStarted) {
    return true;
  }

  if (message.toolStatus === "finished" && hasFinished) {
    return true;
  }

  return false;
}

function SubagentCard({ messages }: { messages: AgentMessage[] }) {
  const [expanded, setExpanded] = useState(false);
  const sent = messages.find((message) => message.toolStatus === "started");
  const received = messages.find((message) => message.toolStatus === "finished");
  const isRunning = Boolean(sent) && !received;
  const agentName = (sent?.toolName || received?.toolName || "subagent:").replace("subagent:", "");
  const question = sent?.content || "";
  const response = received?.content || "";

  return (
    <div className="py-1 text-sm" data-testid="subagent-card">
      <button
        type="button"
        onClick={() => setExpanded((current) => !current)}
        className="flex cursor-pointer items-center gap-2 text-slate-700 hover:text-slate-900"
      >
        <Bot className="size-4 shrink-0" />
        <span>{agentName}</span>
        <span className={cn("text-[10px] font-medium", isRunning ? "text-blue-600" : "text-emerald-600")}>
          {isRunning ? "Working…" : "Done"}
        </span>
        <ChevronRight className={cn("size-3 transition-transform", expanded && "rotate-90")} />
      </button>
      {expanded ? (
        <div className="mt-2 space-y-2 pl-6">
          {question ? (
            <p className="text-xs italic text-slate-500">
              "{question.length > 200 ? `${question.slice(0, 200)}…` : question}"
            </p>
          ) : null}
          {response ? (
            <div className="max-h-60 overflow-y-auto">
              <p className="whitespace-pre-wrap text-xs text-slate-700">{response}</p>
            </div>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}

function ToolGroupRow({ messages }: { messages: AgentMessage[] }) {
  const [expanded, setExpanded] = useState(true);
  const hasRunning = messages.some((message) => message.toolStatus === "started");
  const count = messages.length;
  const label = hasRunning
    ? `Running command${count > 1 ? ` (${count})` : ""}...`
    : `Ran ${count} command${count !== 1 ? "s" : ""}`;

  return (
    <div className={cn("py-1 text-sm", hasRunning && "animate-tool-glow")} data-testid="agent-tool-group">
      <button
        type="button"
        onClick={() => setExpanded((current) => !current)}
        className="flex cursor-pointer items-center gap-2 text-slate-700 hover:text-slate-900"
      >
        <SquareTerminal className="size-4 shrink-0" />
        <span>{label}</span>
        <ChevronRight className={cn("size-3 transition-transform", expanded && "rotate-90")} />
      </button>
      {expanded ? (
        <div className="mt-2 space-y-1">
          {messages.map((message) => (
            <ToolMessageRow key={message.id} message={message} />
          ))}
        </div>
      ) : null}
    </div>
  );
}

function ToolMessageRow({ message }: { message: AgentMessage }) {
  const running = message.toolStatus === "started";
  const [expanded, setExpanded] = useState(running);
  const command = message.content;
  const canExpand = Boolean(command);
  const preview = command ? command.split("\n")[0].substring(0, 80) : "command";

  useEffect(() => {
    setExpanded(running);
  }, [running]);

  return (
    <div className="text-xs" data-testid="agent-tool-message">
      <button
        type="button"
        onClick={() => canExpand && setExpanded((current) => !current)}
        disabled={!canExpand}
        className={cn(
          "flex w-full items-center gap-1.5 text-left",
          running ? "text-slate-700" : "text-slate-600",
          canExpand && "cursor-pointer hover:text-slate-900",
        )}
      >
        <span className="shrink-0 text-[10px]">{running ? "▶" : "✓"}</span>
        <span className="truncate">{running ? "Running..." : preview}</span>
      </button>
      {expanded && command ? (
        <div className="mt-1 overflow-hidden rounded-lg border border-slate-200 bg-white">
          <div className="flex items-center justify-between border-b border-slate-200 bg-slate-50 px-3 py-1">
            <span className="text-[10px] font-medium uppercase tracking-wider text-slate-500">bash</span>
          </div>
          <pre className="max-h-[200px] overflow-auto break-words whitespace-pre-wrap p-3 font-mono text-xs text-slate-700">
            {command}
          </pre>
        </div>
      ) : null}
    </div>
  );
}

function ThinkingRow() {
  return (
    <div className="flex animate-tool-glow items-center gap-2 py-1 text-sm text-slate-500" data-testid="agent-thinking">
      <Loader2 className="size-4 shrink-0 animate-spin" />
      <span>Thinking…</span>
    </div>
  );
}

function statusLabel(status: string): string {
  switch (status) {
    case "streaming":
      return "Agent is working...";
    case "failed":
      return "Last turn failed";
    case "terminated":
      return "Session ended";
    default:
      return "Ready";
  }
}

function formatTime(iso: string): string {
  const date = new Date(iso);
  return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}
