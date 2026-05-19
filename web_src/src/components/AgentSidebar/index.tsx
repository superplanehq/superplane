import { Bot, ChevronRight, Loader2, SquareTerminal, X } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { cn } from "@/lib/utils";
import {
  useAgentChatMessages,
  useCanvasAgentChat,
  useInterruptAgentChat,
  useDefineAgentOutcome,
  useSendAgentChatMessage,
} from "@/hooks/useAgentChats";
import { useAgentSessionWebsocket } from "@/hooks/useAgentSessionWebsocket";
import type { AgentMessage } from "./types";
import type { AgentState } from "./useAgentState";
import type { AgentMode } from "./useAgentState";
import { useSidebarWidth } from "./useSidebarWidth";
import { RichMessage } from "./widgets/RichMessage";
import { DraftActionsWidget } from "./widgets/DraftActionsWidget";
import { ChatComposer } from "./ChatComposer";
import { useDraftActions } from "./useDraftActions";
import { useChatScroll } from "./useChatScroll";
import { isSystemNotification, formatSystemNotification } from "./systemMessages";
import { OutcomeProgressWidget, type OutcomeState, type OutcomePhase, type IterationEntry, type GradingEntry } from "./widgets/OutcomeProgressWidget";
import type { RubricCategory } from "./widgets/parser";

export interface AgentSidebarProps {
  agentState: AgentState;
}

export function AgentSidebar({ agentState }: AgentSidebarProps) {
  if (!agentState.showAgentSidebarToggle || !agentState.isAgentSidebarOpen) {
    return null;
  }
  if (!agentState.canvasId) {
    return null;
  }
  return <OpenAgentSidebar agentState={agentState} />;
}

function OpenAgentSidebar({ agentState }: AgentSidebarProps) {
  const canvasId = agentState.canvasId ?? "";
  const organizationId = agentState.organizationId ?? "";
  const chatQuery = useCanvasAgentChat(canvasId, organizationId, agentState.isAgentSidebarOpen);
  const chatId = chatQuery.data?.id ?? null;

  return (
    <SidebarShell onClose={agentState.closeSidebar}>
      {chatQuery.isLoading || !chatId ? (
        <div className="flex items-center justify-center py-8 text-sm text-muted-foreground">
          <Loader2 className="size-4 animate-spin mr-2" /> Loading…
        </div>
      ) : (
        <ChatConversation
          chatId={chatId}
          canvasId={canvasId}
          organizationId={organizationId}
          agentMode={agentState.agentMode}
          onModeSwitch={agentState.switchAgentMode}
        />
      )}
    </SidebarShell>
  );
}

function SidebarShell({ children, onClose }: { children: React.ReactNode; onClose: () => void }) {
  const { sidebarRef, width, isResizing, handleMouseDown } = useSidebarWidth();
  return (
    <aside
      ref={sidebarRef}
      data-testid="agent-sidebar"
      className="relative border-r border-border shrink-0 h-full z-21 flex flex-col overflow-hidden bg-white"
      style={{ width }}
    >
      <header className="flex items-center justify-between gap-3 px-4 py-2.5 border-b border-border shrink-0 min-w-0">
        <h2 className="text-base font-medium min-w-0 flex-1 truncate">Agent</h2>
        <button
          type="button"
          onClick={onClose}
          aria-label="Close SuperPlane Agent"
          data-testid="close-agent-sidebar-button"
          className="z-40 w-6 h-6 hover:bg-slate-950/5 rounded-md flex items-center justify-center cursor-pointer text-muted-foreground"
        >
          <X size={16} />
        </button>
      </header>
      <div className="flex flex-1 flex-col min-h-0">{children}</div>
      <div
        onMouseDown={handleMouseDown}
        className={cn(
          "absolute right-0 top-0 bottom-0 w-1.5 -mr-0.5 cursor-ew-resize hover:bg-violet-300/40",
          isResizing && "bg-violet-400/60",
        )}
        aria-hidden
        data-testid="agent-sidebar-resize-handle"
      />
    </aside>
  );
}

function ChatConversation({
  chatId,
  canvasId,
  organizationId,
  agentMode,
  onModeSwitch,
}: {
  chatId: string;
  canvasId: string;
  organizationId: string;
  agentMode: AgentMode;
  onModeSwitch: (mode: AgentMode) => void;
}) {
  const [searchParams] = useSearchParams();
  const isEditing = searchParams.has("version");
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

  // pages[0] is the latest fetch; later entries are older batches loaded
  // via scroll-up. Reverse so chronological order falls out of flatMap.
  const messages = useMemo(
    () =>
      messagesQuery.data?.pages
        .slice()
        .reverse()
        .flatMap((p) => p.messages) ?? [],
    [messagesQuery.data],
  );
  const hasRunningTool = useMemo(
    () => messages.some((m) => m.role === "tool" && m.toolStatus === "started"),
    [messages],
  );
  // Thinking is just a placeholder for the gap between sending and the first
  // assistant block landing in the cache. Once a tool starts running its own
  // row signals activity, so we suppress Thinking to avoid double-indicators.
  const showThinking = status === "streaming" && !hasRunningTool && messages[messages.length - 1]?.role === "user";

  const wsCallbacks = useMemo(
    () => ({
      onPersistedMessage: (message: AgentMessage) => {
        // Clear outcome widget when draft is published or discarded
        if (message.content?.includes("published") || message.content?.includes("discarded")) {
          setOutcomeState(null);
        }
      },
      onStatusChange: (next: string, err?: string) => {
        setStatus(next || "idle");
        setError(err ?? null);
      },
      onOutcomeEvent: (
        phase: "start" | "end",
        evaluation: { iteration: number; result?: string; explanation?: string },
      ) => {
        setOutcomeState((prev) => {
          if (!prev) return prev;
          if (phase === "start") {
            // Grading started — update log with grading entry
            const updatedLog = [...prev.log];
            // Mark last iteration as finished
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
          // phase === "end" — now we get result + explanation from SSE
          if (evaluation.result) {
            const updatedLog = [...prev.log];
            // Update the last grading entry with result
            for (let i = updatedLog.length - 1; i >= 0; i--) {
              const entry = updatedLog[i] as GradingEntry;
              if (entry.phase === "grading") {
                updatedLog[i] = {
                  phase: evaluation.result === "satisfied" ? "satisfied" : "needs_revision",
                  explanation: evaluation.explanation,
                };
                break;
              }
            }
            if (evaluation.result === "satisfied") {
              return { ...prev, phase: "passed" as OutcomePhase, log: updatedLog };
            }
            // Needs revision — add next iteration
            const nextIteration = prev.iteration + 1;
            const isExhausted = nextIteration > prev.maxIterations;
            if (isExhausted) {
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
          return prev;
        });
      },
    }),
    [],
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
  }, [chatId, draft, sendMutation, agentMode]);

  const handleStop = useCallback(() => {
    interruptMutation.mutate({ chatId });
  }, [chatId, interruptMutation]);

  const handleStartBuilding = useCallback(
    async (rubric: { title: string; criteria: string[]; categories?: RubricCategory[] }) => {
      // Format rubric text with categories if present
      const rubricText = rubric.categories && rubric.categories.length > 0
        ? `# ${rubric.title}\n\n${rubric.categories.map((cat) => `## ${cat.heading}\n${cat.criteria.map((c) => `- ${c.text}`).join("\n")}`).join("\n\n")}`
        : `# ${rubric.title}\n\n${rubric.criteria.map((c) => `- ${c}`).join("\n")}`;

      // In Build mode: rubric is a spec confirmation, not an outcome.
      // Agent already has full context — just confirm.
      if (agentMode === "builder") {
        await sendMutation.mutateAsync({
          chatId,
          content: "Specs approved. Start building.",
          mode: "builder",
        });
        return;
      }

      // In Plan mode: kick off outcome with grading loop
      setOutcomeState({
        title: rubric.title,
        criteria: rubric.criteria.map((c) => ({ text: c })),
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
      } catch (err) {
        console.error("Failed to define outcome:", err);
        setOutcomeState(null);
        await sendMutation.mutateAsync({
          chatId,
          content: `Start building based on this plan:\n\n${rubricText}`,
          mode: "builder",
        });
      }
    },
    [chatId, agentMode, outcomeMutation, sendMutation],
  );

  const scrollRef = useChatScroll(messagesQuery, chatId, messages.length, showThinking);

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <div ref={scrollRef} className="flex-1 min-h-0 overflow-y-auto p-3 space-y-2" data-testid="agent-chat-messages">
        {messagesQuery.isLoading ? (
          <div className="flex items-center justify-center py-8 text-sm text-muted-foreground">
            <Loader2 className="size-4 animate-spin mr-2" /> Loading…
          </div>
        ) : (
          <>
            {messagesQuery.isFetchingNextPage ? (
              <div className="flex items-center justify-center py-2 text-xs text-muted-foreground">
                <Loader2 className="size-3 animate-spin mr-2" /> Loading older messages…
              </div>
            ) : null}
            {groupMessages(messages).map((group) =>
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
                  onModeSwitch={onModeSwitch}
                  onStartBuilding={handleStartBuilding}
                />
              ),
            )}
          </>
        )}
        {showThinking ? <ThinkingRow /> : null}
        {error ? <p className="text-sm text-red-600 px-3 py-2">{error}</p> : null}
      </div>
      {outcomeState && (
        <div className="px-3 py-2 border-t border-slate-200">
          <OutcomeProgressWidget state={outcomeState} onDismiss={() => setOutcomeState(null)} />
        </div>
      )}
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
        modeDisabled={status === "streaming" || (outcomeState != null && outcomeState.phase !== "passed" && outcomeState.phase !== "exhausted")}
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
  onModeSwitch: _onModeSwitch,
  onStartBuilding,
}: {
  message: AgentMessage;
  sendMutation: ReturnType<typeof useSendAgentChatMessage>;
  chatId: string;
  canvasId: string;
  organizationId: string;
  agentMode: AgentMode;
  onModeSwitch: (mode: AgentMode) => void;
  onStartBuilding: (rubric: { title: string; criteria: string[]; categories?: RubricCategory[] }) => void;
}) {
  const handleAction = useCallback(
    async (action: string) => {
      if (sendMutation.isPending) return;
      try {
        await sendMutation.mutateAsync({ chatId, content: action, mode: agentMode });
      } catch (err) {
        console.error("Failed to send action:", err);
      }
    },
    [chatId, sendMutation, agentMode],
  );

  if (message.role === "tool") {
    return <ToolMessageRow message={message} />;
  }

  // System notification messages (draft published/discarded)
  if (message.role === "system" || (message.role === "user" && isSystemNotification(message.content))) {
    const text = message.role === "system" ? formatSystemNotification(message.content) : message.content;
    return (
      <div className="flex justify-center">
        <span className="text-[11px] text-slate-400 italic px-2">{text}</span>
      </div>
    );
  }

  const isUser = message.role === "user";

  return (
    <div className={cn("flex flex-col", isUser ? "items-end" : "items-start")}>
      <div
        className={cn(
          "rounded-lg px-3 py-2 text-sm max-w-[85%] break-words",
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
      {message.createdAt && (
        <span className="text-[10px] text-slate-400 mt-0.5 px-1">{formatTime(message.createdAt)}</span>
      )}
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

  function flushSubagent() {
    if (subagentBuffer.length > 0) {
      groups.push({ type: "subagent-group", messages: [...subagentBuffer] });
      subagentBuffer = [];
    }
  }

  for (const m of messages) {
    if (m.role === "tool" && m.toolName?.startsWith("subagent:")) {
      flushTools();
      subagentBuffer.push(m);
    } else if (m.role === "tool") {
      flushSubagent();
      toolBuffer.push(m);
    } else {
      flushTools();
      flushSubagent();
      groups.push({ type: "message", message: m });
    }
  }
  flushTools();
  flushSubagent();
  return groups;
}

function SubagentCard({ messages }: { messages: AgentMessage[] }) {
  const [expanded, setExpanded] = useState(false);
  const sent = messages.find((m) => m.toolStatus === "started");
  const received = messages.find((m) => m.toolStatus === "finished");
  const isRunning = sent && !received;
  const agentName = (sent?.toolName || received?.toolName || "subagent:").replace("subagent:", "");
  const question = sent?.content || "";
  const response = received?.content || "";

  return (
    <div className="text-sm py-1" data-testid="subagent-card">
      <button
        type="button"
        onClick={() => setExpanded((prev) => !prev)}
        className="flex items-center gap-2 cursor-pointer text-slate-700 hover:text-slate-900"
      >
        <Bot className="size-4 shrink-0" />
        <span>{agentName}</span>
        <span className={cn("text-[10px] font-medium", isRunning ? "text-blue-600" : "text-emerald-600")}>
          {isRunning ? "Working\u2026" : "Done"}
        </span>
        <ChevronRight className={cn("size-3 transition-transform", expanded && "rotate-90")} />
      </button>
      {expanded && (
        <div className="mt-2 space-y-2 pl-6">
          {question && (
            <p className="text-xs text-slate-500 italic">"{question.length > 200 ? question.slice(0, 200) + "\u2026" : question}"</p>
          )}
          {response && (
            <div className="max-h-60 overflow-y-auto">
              <p className="text-xs text-slate-700 whitespace-pre-wrap">{response}</p>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function ToolGroupRow({ messages }: { messages: AgentMessage[] }) {
  const [expanded, setExpanded] = useState(true);
  const hasRunning = messages.some((m) => m.toolStatus === "started");
  const count = messages.length;
  const label = hasRunning
    ? `Running command${count > 1 ? ` (${count})` : ""}...`
    : `Ran ${count} command${count !== 1 ? "s" : ""}`;

  return (
    <div className={cn("text-sm py-1", hasRunning && "animate-tool-glow")} data-testid="agent-tool-group">
      <button
        type="button"
        onClick={() => setExpanded((prev) => !prev)}
        className="flex items-center gap-2 cursor-pointer text-slate-700 hover:text-slate-900"
      >
        <SquareTerminal className="size-4 shrink-0" />
        <span>{label}</span>
        <ChevronRight className={cn("size-3 transition-transform", expanded && "rotate-90")} />
      </button>
      {expanded && (
        <div className="mt-2 space-y-1">
          {messages.map((m) => (
            <ToolMessageRow key={m.id} message={m} />
          ))}
        </div>
      )}
    </div>
  );
}

function ToolMessageRow({ message }: { message: AgentMessage }) {
  const running = message.toolStatus === "started";
  const [expanded, setExpanded] = useState(running);
  const command = message.content;
  const canExpand = Boolean(command);
  const preview = command ? command.split("\n")[0].substring(0, 80) : "command";

  // Auto-expand when command starts running, collapse when it finishes
  useEffect(() => {
    setExpanded(running);
  }, [running]);

  return (
    <div className="text-xs">
      <button
        type="button"
        onClick={() => canExpand && setExpanded((prev) => !prev)}
        disabled={!canExpand}
        className={cn(
          "flex items-center gap-1.5 text-left w-full",
          running ? "text-slate-700" : "text-slate-600",
          canExpand && "cursor-pointer hover:text-slate-900",
        )}
      >
        <span className="shrink-0 text-[10px]">{running ? "▶" : "✓"}</span>
        <span className="truncate">{running ? "Running..." : preview}</span>
      </button>
      {expanded && command ? (
        <div className="mt-1 rounded-lg border border-slate-200 bg-white overflow-hidden">
          <div className="flex items-center justify-between px-3 py-1 bg-slate-50 border-b border-slate-200">
            <span className="text-[10px] font-medium text-slate-500 uppercase tracking-wider">bash</span>
          </div>
          <pre className="p-3 text-xs font-mono text-slate-700 whitespace-pre-wrap break-words overflow-auto max-h-[200px]">
            {command}
          </pre>
        </div>
      ) : null}
    </div>
  );
}

function ThinkingRow() {
  return (
    <div className="flex items-center gap-2 text-sm py-1 text-slate-500 animate-tool-glow" data-testid="agent-thinking">
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
  const d = new Date(iso);
  return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}
