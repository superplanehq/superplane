import { ArrowUp, ChevronRight, Loader2, SquareTerminal } from "lucide-react";
import { memo, useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { RichMessage } from "@/components/AgentSidebar/widgets/RichMessage";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import { useAgentChatMessages, useCanvasAgentChat, useSendAgentChatMessage } from "@/hooks/useAgentChats";
import { useAgentSessionWebsocket } from "@/hooks/useAgentSessionWebsocket";
import { EmptyToolTab } from "./EmptyToolTab";
import { SidebarShell } from "./SidebarShell";
import { ToolTabsHeader } from "./ToolTabsHeader";
import type { AgentMessage } from "./types";
import type { CanvasToolSidebarState } from "./useCanvasToolSidebarState";

const TAB_AGENT = "agent",
  TAB_RUNS = "runs",
  TAB_VERSIONS = "versions";
type CanvasToolSidebarMode = "default" | "version-live" | "version-edit" | "runs" | "dashboard";

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
  if (!toolSidebarState.showToolSidebarToggle || !toolSidebarState.isToolSidebarOpen || !toolSidebarState.canvasId)
    return null;

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

function AgentTabPanel({ toolSidebarState }: CanvasToolSidebarProps) {
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
  return <ChatConversation chatId={chatId} canvasId={canvasId} organizationId={organizationId} />;
}

function ChatConversation({
  chatId,
  canvasId,
  organizationId,
}: {
  chatId: string;
  canvasId: string;
  organizationId: string;
}) {
  const messagesQuery = useAgentChatMessages(chatId, organizationId, true);
  const sendMutation = useSendAgentChatMessage(organizationId, canvasId);

  const [status, setStatus] = useState<string>("idle");
  const [error, setError] = useState<string | null>(null);

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
      onStatusChange: (next: string, err?: string) => {
        setStatus(next || "idle");
        setError(err ?? null);
      },
    }),
    [],
  );
  useAgentSessionWebsocket(chatId, organizationId, wsCallbacks);

  const sendContent = useCallback(
    async (content: string) => {
      const value = content.trim();
      if (!value || sendMutation.isPending) return false;
      setError(null);
      try {
        await sendMutation.mutateAsync({ chatId, content: value });
        return true;
      } catch (err) {
        setError(err instanceof Error ? err.message : "failed to send message");
        return false;
      }
    },
    [chatId, sendMutation],
  );

  const scrollRef = useRef<HTMLDivElement>(null);
  const previousScrollHeight = useRef<number | null>(null);

  useEffect(() => {
    const element = scrollRef.current;
    if (!element) return;

    const onScroll = () => {
      if (element.scrollTop > 24) return;
      if (!messagesQuery.hasNextPage || messagesQuery.isFetchingNextPage) return;
      previousScrollHeight.current = element.scrollHeight;
      void messagesQuery.fetchNextPage();
    };

    element.addEventListener("scroll", onScroll);
    return () => element.removeEventListener("scroll", onScroll);
  }, [messagesQuery]);

  useLayoutEffect(() => {
    const element = scrollRef.current;
    if (!element) return;
    if (previousScrollHeight.current !== null) {
      element.scrollTop = element.scrollHeight - previousScrollHeight.current;
      previousScrollHeight.current = null;
      return;
    }
    element.scrollTop = element.scrollHeight;
  }, [chatId, messages.length, showThinking]);

  const messageGroups = useMemo(() => groupMessages(messages), [messages]);

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div ref={scrollRef} className="min-h-0 flex-1 space-y-2.5 overflow-y-auto p-3" data-testid="agent-chat-messages">
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
              ) : (
                <MessageRow
                  key={group.message.id}
                  message={group.message}
                  canvasId={canvasId}
                  organizationId={organizationId}
                  onAction={sendContent}
                />
              ),
            )}
          </>
        )}
        {showThinking ? <ThinkingRow /> : null}
        {error ? <p className="px-3 py-2 text-sm text-red-600">{error}</p> : null}
      </div>
      <ChatComposer
        key={chatId}
        onSend={sendContent}
        sending={sendMutation.isPending}
        statusLabel={statusLabel(status)}
      />
    </div>
  );
}

function ChatComposer({
  onSend,
  sending,
  statusLabel,
}: {
  onSend: (content: string) => Promise<boolean>;
  sending: boolean;
  statusLabel: string;
}) {
  const [draft, setDraft] = useState("");
  const [isSending, setIsSending] = useState(false);
  const canSend = Boolean(draft.trim()) && !sending && !isSending;
  const handleSend = useCallback(async () => {
    if (!canSend) return;
    const valueToSend = draft;
    setDraft("");
    setIsSending(true);
    try {
      const ok = await onSend(valueToSend);
      if (!ok) {
        setDraft((nextDraft) => (nextDraft.trim() ? nextDraft : valueToSend));
      }
    } finally {
      setIsSending(false);
    }
  }, [canSend, draft, onSend]);

  return (
    <footer className="border-t border-slate-950/15 px-3 pb-3">
      <Textarea
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        rows={1}
        placeholder="Ask the agent…"
        data-testid="agent-input"
        className={cn(
          "min-h-9 w-full resize-none border-0 bg-transparent px-0 py-2 text-sm shadow-none",
          "outline-none ring-0 focus-visible:border-0 focus-visible:ring-0 focus-visible:outline-none",
          "placeholder:text-muted-foreground disabled:cursor-not-allowed disabled:opacity-50",
          "text-[rgba(10,10,10,1)] dark:bg-transparent",
        )}
        onKeyDown={(e) => {
          if (e.key !== "Enter") return;
          const nativeEvent = e.nativeEvent;
          if ("isComposing" in nativeEvent && nativeEvent.isComposing) return;
          if (e.shiftKey) return;
          e.preventDefault();
          void handleSend();
        }}
      />
      <div className="flex items-center justify-between gap-2 pt-1">
        <span className="min-w-0 flex-1 text-xs text-muted-foreground">{statusLabel}</span>
        <Button
          type="button"
          variant="default"
          size="icon"
          className="size-7 shrink-0 rounded-full"
          onClick={() => void handleSend()}
          disabled={!canSend}
          aria-label="Send message"
          data-testid="agent-send-message-button"
        >
          {sending ? (
            <Loader2 className="size-3.5 animate-spin" aria-hidden />
          ) : (
            <ArrowUp className="size-3.5" aria-hidden />
          )}
        </Button>
      </div>
    </footer>
  );
}

const MessageRow = memo(function MessageRow({
  message,
  canvasId,
  organizationId,
  onAction,
}: {
  message: AgentMessage;
  canvasId: string;
  organizationId: string;
  onAction: (content: string) => Promise<boolean>;
}) {
  const handleAction = useCallback((action: string) => onAction(action), [onAction]);

  if (message.role === "tool") {
    return <ToolMessageRow message={message} />;
  }

  const isUser = message.role === "user";

  return (
    <div className={cn("flex", isUser ? "justify-end" : "justify-start")}>
      <div
        className={cn(
          "break-words text-sm",
          isUser
            ? "max-w-[85%] rounded-lg bg-slate-100 px-3 py-2 whitespace-pre-wrap text-slate-900"
            : "max-w-[720px] text-slate-900",
        )}
        data-testid={isUser ? "agent-user-message" : "agent-assistant-message"}
      >
        {isUser ? (
          message.content
        ) : (
          <RichMessage
            content={message.content}
            onAction={handleAction}
            canvasId={canvasId}
            organizationId={organizationId}
          />
        )}
      </div>
    </div>
  );
});

type MessageGroup = { type: "message"; message: AgentMessage } | { type: "tool-group"; messages: AgentMessage[] };

function groupMessages(messages: AgentMessage[]): MessageGroup[] {
  const groups: MessageGroup[] = [];
  let toolBuffer: AgentMessage[] = [];

  function flushTools() {
    if (toolBuffer.length > 0) {
      groups.push({ type: "tool-group", messages: [...toolBuffer] });
      toolBuffer = [];
    }
  }

  for (const message of messages) {
    if (message.role === "tool") {
      toolBuffer.push(message);
    } else {
      flushTools();
      groups.push({ type: "message", message });
    }
  }

  flushTools();
  return groups;
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
        onClick={() => setExpanded((previous) => !previous)}
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
  const [expanded, setExpanded] = useState(true);
  const command = message.content;
  const canExpand = Boolean(command);
  const running = message.toolStatus === "started";
  const preview = command ? command.split("\n")[0].substring(0, 80) : "command";

  return (
    <div className="text-xs" data-testid="agent-tool-message">
      <button
        type="button"
        onClick={() => canExpand && setExpanded((previous) => !previous)}
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
