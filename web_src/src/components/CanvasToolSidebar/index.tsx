import { ArrowUp, Loader2, SquareTerminal } from "lucide-react";
import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import ReactMarkdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import { useAgentChatMessages, useCanvasAgentChat, useSendAgentChatMessage } from "@/hooks/useAgentChats";
import { useAgentSessionWebsocket } from "@/hooks/useAgentSessionWebsocket";
import type { AgentMessage } from "./types";
import type { CanvasToolSidebarState } from "./useCanvasToolSidebarState";
import { useSidebarWidth } from "./useSidebarWidth";

const TAB_AGENT = "agent";
const TAB_RUNS = "runs";
const TAB_VERSIONS = "versions";

export interface CanvasToolSidebarProps {
  toolSidebarState: CanvasToolSidebarState;
}

export function CanvasToolSidebar({ toolSidebarState }: CanvasToolSidebarProps) {
  if (!toolSidebarState.showToolSidebarToggle || !toolSidebarState.isToolSidebarOpen) {
    return null;
  }
  if (!toolSidebarState.canvasId) {
    return null;
  }
  return <OpenCanvasToolSidebar toolSidebarState={toolSidebarState} />;
}

function OpenCanvasToolSidebar({ toolSidebarState }: CanvasToolSidebarProps) {
  const [activeTab, setActiveTab] = useState(TAB_AGENT);

  const tabs = [
    { value: TAB_AGENT, label: "Agent" },
    { value: TAB_RUNS, label: "Runs" },
    { value: TAB_VERSIONS, label: "Versions" },
  ] as const;

  return (
    <SidebarShell>
      <div className="flex min-h-0 flex-1 flex-col gap-0">
        <div
          className="flex h-10 min-h-10 shrink-0 flex-row items-stretch border-b border-slate-950/15 px-4"
          role="tablist"
          aria-label="Canvas tools"
        >
          {tabs.map(({ value, label }) => (
            <button
              key={value}
              type="button"
              role="tab"
              aria-selected={activeTab === value}
              onClick={() => setActiveTab(value)}
              className={cn(
                "mr-4 mb-[-1px] flex items-center border-b text-[13px] font-medium transition-colors",
                activeTab === value
                  ? "border-gray-700 text-gray-800 dark:border-blue-600 dark:text-blue-400"
                  : "border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300",
              )}
            >
              {label}
            </button>
          ))}
        </div>

        {activeTab === TAB_AGENT ? (
          <div className="m-0 flex min-h-0 flex-1 flex-col overflow-hidden" role="tabpanel">
            <AgentTabPanel toolSidebarState={toolSidebarState} />
          </div>
        ) : null}
        {activeTab === TAB_RUNS ? (
          <div className="m-0 flex min-h-0 flex-1 flex-col" role="tabpanel">
            <EmptyToolTab />
          </div>
        ) : null}
        {activeTab === TAB_VERSIONS ? (
          <div className="m-0 flex min-h-0 flex-1 flex-col" role="tabpanel">
            <EmptyToolTab />
          </div>
        ) : null}
      </div>
    </SidebarShell>
  );
}

function EmptyToolTab() {
  return <div className="min-h-0 flex-1" aria-hidden />;
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

function SidebarShell({ children }: { children: React.ReactNode }) {
  const { sidebarRef, width, isResizing, handleMouseDown } = useSidebarWidth();
  return (
    <aside
      ref={sidebarRef}
      data-testid="canvas-tool-sidebar"
      className="relative z-21 flex h-full shrink-0 flex-col border-r border-border bg-white"
      style={{ width }}
    >
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden">{children}</div>
      <div
        onMouseDown={handleMouseDown}
        className={cn(
          "absolute top-0 right-0 bottom-0 z-10 w-1 translate-x-1/2 cursor-col-resize bg-transparent transition-colors duration-150 ease-out delay-0",
          "hover:delay-300 hover:bg-slate-950/10",
          isResizing && "bg-slate-950/10 delay-0",
        )}
        aria-hidden
        data-testid="canvas-tool-sidebar-resize-handle"
      />
    </aside>
  );
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

  const [draft, setDraft] = useState("");
  const [status, setStatus] = useState<string>("idle");
  const [error, setError] = useState<string | null>(null);

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

  const handleSend = useCallback(async () => {
    const value = draft.trim();
    if (!value || sendMutation.isPending) return;
    setDraft("");
    setError(null);
    try {
      await sendMutation.mutateAsync({ chatId, content: value });
    } catch (err) {
      setError(err instanceof Error ? err.message : "failed to send message");
    }
  }, [chatId, draft, sendMutation]);

  const scrollRef = useRef<HTMLDivElement>(null);
  const previousScrollHeight = useRef<number | null>(null);

  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    const onScroll = () => {
      if (el.scrollTop > 24) return;
      if (!messagesQuery.hasNextPage || messagesQuery.isFetchingNextPage) return;
      previousScrollHeight.current = el.scrollHeight;
      void messagesQuery.fetchNextPage();
    };
    el.addEventListener("scroll", onScroll);
    return () => el.removeEventListener("scroll", onScroll);
  }, [messagesQuery]);

  useLayoutEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    if (previousScrollHeight.current !== null) {
      el.scrollTop = el.scrollHeight - previousScrollHeight.current;
      previousScrollHeight.current = null;
      return;
    }
    el.scrollTop = el.scrollHeight;
  }, [chatId, messages.length, showThinking]);

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
            {messages.map((m) => (
              <MessageRow key={m.id} message={m} />
            ))}
          </>
        )}
        {showThinking ? <ThinkingRow /> : null}
        {error ? <p className="px-3 py-2 text-sm text-red-600">{error}</p> : null}
      </div>
      <ChatComposer
        draft={draft}
        onDraftChange={setDraft}
        onSend={handleSend}
        sending={sendMutation.isPending}
        statusLabel={statusLabel(status)}
      />
    </div>
  );
}

function ChatComposer({
  draft,
  onDraftChange,
  onSend,
  sending,
  statusLabel,
}: {
  draft: string;
  onDraftChange: (value: string) => void;
  onSend: () => void;
  sending: boolean;
  statusLabel: string;
}) {
  const canSend = Boolean(draft.trim()) && !sending;
  return (
    <footer className="border-t border-slate-950/15 px-3 pb-3">
      <Textarea
        value={draft}
        onChange={(e) => onDraftChange(e.target.value)}
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
          const native = e.nativeEvent;
          if ("isComposing" in native && native.isComposing) return;
          if (e.shiftKey) return;
          e.preventDefault();
          if (!canSend) return;
          onSend();
        }}
      />
      <div className="flex items-center justify-between gap-2 pt-1">
        <span className="min-w-0 flex-1 text-xs text-muted-foreground">{statusLabel}</span>
        <Button
          type="button"
          variant="default"
          size="icon"
          className="size-7 shrink-0 rounded-full"
          onClick={onSend}
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

function MessageRow({ message }: { message: AgentMessage }) {
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
        {isUser ? message.content : <AgentMarkdown content={message.content} />}
      </div>
    </div>
  );
}

function ToolMessageRow({ message }: { message: AgentMessage }) {
  const [expanded, setExpanded] = useState(false);
  const command = message.content;
  const canExpand = Boolean(command);
  const running = message.toolStatus === "started";
  return (
    <div
      className={cn("flex items-start gap-2 py-1 text-sm text-slate-500", running && "animate-tool-glow")}
      data-testid="agent-tool-message"
    >
      <SquareTerminal className="mt-0.5 size-4 shrink-0" />
      <div className="min-w-0 flex-1">
        <button
          type="button"
          onClick={() => canExpand && setExpanded((prev) => !prev)}
          disabled={!canExpand}
          className={cn("text-left", canExpand && "cursor-pointer hover:text-slate-700")}
        >
          {running ? "Running command" : "Ran command"}
        </button>
        {expanded && command ? (
          <div className="mt-1 font-mono text-xs break-words whitespace-pre-wrap text-slate-400">{command}</div>
        ) : null}
      </div>
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

const AGENT_MARKDOWN_CLASSES =
  "max-w-none [&_h1]:mb-2 [&_h1]:mt-1.5 [&_h1]:text-base [&_h1]:font-semibold [&_h1:first-child]:mt-0 " +
  "[&_h2]:mb-2 [&_h2]:mt-2.5 [&_h2]:text-sm [&_h2]:font-semibold [&_h2:first-child]:mt-0 " +
  "[&_h3]:mb-1.5 [&_h3]:mt-2 [&_h3]:text-sm [&_h3]:font-semibold [&_h3:first-child]:mt-0 " +
  "[&_p]:mb-2.5 [&_p]:leading-relaxed [&_p:last-child]:mb-0 " +
  "[&_ol]:mt-1.5 [&_ol]:mb-3 [&_ol]:ml-5 [&_ol]:list-decimal [&_ul]:mt-1.5 [&_ul]:mb-3 [&_ul]:ml-5 [&_ul]:list-disc [&_li]:mb-1.5 " +
  "[&_blockquote]:my-2 [&_blockquote]:border-l-2 [&_blockquote]:border-slate-300 [&_blockquote]:pl-3 " +
  "[&_code]:rounded [&_code]:bg-slate-200/70 [&_code]:px-1 [&_code]:py-0.5 [&_code]:text-xs " +
  "[&_pre]:my-2 [&_pre]:overflow-auto [&_pre]:rounded [&_pre]:bg-slate-200/70 [&_pre]:p-2 " +
  "[&_pre_code]:bg-transparent [&_pre_code]:p-0 " +
  "[&_a]:underline [&_a]:underline-offset-2 [&_a]:decoration-current " +
  "[&_strong]:font-semibold [&_b]:font-semibold";

function AgentMarkdown({ content }: { content: string }) {
  if (!content) return null;
  return (
    <div className={AGENT_MARKDOWN_CLASSES}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkBreaks]}
        components={{
          a: ({ children, href }) => (
            <a href={href} target="_blank" rel="noopener noreferrer">
              {children}
            </a>
          ),
          table: ({ children }) => (
            <div className="my-3 w-full max-w-full overflow-x-auto border-y border-slate-200 bg-white">
              <table
                className={cn(
                  "w-full min-w-[12rem] border-collapse text-sm leading-snug text-slate-900",
                  "[&_th]:border-b [&_th]:border-slate-200 [&_th]:bg-slate-100 [&_th]:px-3 [&_th]:py-1.5 [&_th]:text-left [&_th]:font-semibold",
                  "[&_td]:border-b [&_td]:border-slate-200 [&_td]:px-3 [&_td]:py-1.5 [&_td]:align-top",
                  "[&_tbody_tr:last-child_td]:border-b-0 [&_tbody_tr:last-child_th]:border-b-0",
                  "[&_tbody_tr:nth-child(even)]:bg-slate-50/60",
                )}
              >
                {children}
              </table>
            </div>
          ),
        }}
      >
        {content}
      </ReactMarkdown>
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
