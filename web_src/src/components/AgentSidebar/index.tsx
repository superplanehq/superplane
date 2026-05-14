import { Loader2, Send, SquareTerminal, X } from "lucide-react";
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
import type { AgentState } from "./useAgentState";
import { useSidebarWidth } from "./useSidebarWidth";

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
        <ChatConversation chatId={chatId} canvasId={canvasId} organizationId={organizationId} />
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

  // Load older pages when the user scrolls to the top. We snapshot the
  // pre-fetch scrollHeight so we can restore the scroll position after the
  // new page lands (otherwise the chat jumps to the top).
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

  // Always land at the bottom in one paint. useLayoutEffect runs before the
  // browser paints, so there's no animation to interrupt.
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
            {messages.map((m) => (
              <MessageRow key={m.id} message={m} />
            ))}
          </>
        )}
        {showThinking ? <ThinkingRow /> : null}
        {error ? <p className="text-sm text-red-600 px-3 py-2">{error}</p> : null}
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
  return (
    <footer className="border-t border-border p-3 flex flex-col gap-2">
      <Textarea
        value={draft}
        onChange={(e) => onDraftChange(e.target.value)}
        rows={3}
        placeholder="Ask the agent…"
        data-testid="agent-input"
        className="resize-none"
        onKeyDown={(e) => {
          if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
            e.preventDefault();
            onSend();
          }
        }}
      />
      <div className="flex items-center justify-between">
        <span className="text-xs text-muted-foreground">{statusLabel}</span>
        <Button
          type="button"
          onClick={onSend}
          disabled={!draft.trim() || sending}
          data-testid="agent-send-message-button"
        >
          {sending ? <Loader2 className="size-4 animate-spin" /> : <Send className="size-4" />}
          Send
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
          "rounded-lg px-3 py-2 text-sm max-w-[85%] break-words",
          isUser ? "bg-violet-600 text-white whitespace-pre-wrap" : "bg-slate-100 text-slate-900",
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
      className={cn("flex items-start gap-2 text-sm py-1 text-slate-500", running && "animate-tool-glow")}
      data-testid="agent-tool-message"
    >
      <SquareTerminal className="size-4 shrink-0 mt-0.5" />
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
          <div className="font-mono text-xs text-slate-400 whitespace-pre-wrap break-words mt-1">{command}</div>
        ) : null}
      </div>
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

const AGENT_MARKDOWN_CLASSES =
  "max-w-none [&_h1]:mb-1.5 [&_h1]:mt-1 [&_h1]:text-base [&_h1]:font-semibold [&_h1:first-child]:mt-0 " +
  "[&_h2]:mb-1 [&_h2]:mt-1 [&_h2]:text-sm [&_h2]:font-semibold [&_h2:first-child]:mt-0 " +
  "[&_h3]:mb-0.5 [&_h3]:mt-1 [&_h3]:text-sm [&_h3]:font-semibold [&_h3:first-child]:mt-0 " +
  "[&_p]:mb-2 [&_p]:leading-relaxed [&_p:last-child]:mb-0 " +
  "[&_ol]:mb-2 [&_ol]:ml-5 [&_ol]:list-decimal [&_ul]:mb-2 [&_ul]:ml-5 [&_ul]:list-disc [&_li]:mb-0.5 " +
  "[&_blockquote]:my-2 [&_blockquote]:border-l-2 [&_blockquote]:border-slate-300 [&_blockquote]:pl-3 " +
  "[&_code]:rounded [&_code]:bg-slate-200/70 [&_code]:px-1 [&_code]:py-0.5 [&_code]:text-xs " +
  "[&_pre]:my-2 [&_pre]:overflow-auto [&_pre]:rounded [&_pre]:bg-slate-200/70 [&_pre]:p-2 " +
  "[&_pre_code]:bg-transparent [&_pre_code]:p-0 " +
  "[&_a]:underline [&_a]:underline-offset-2 [&_a]:decoration-current";

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
