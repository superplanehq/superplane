import { ChevronLeft, Loader2, Send, Sparkles, SquareTerminal, Trash2, X } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import ReactMarkdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import {
  useAgentChatMessages,
  useAgentChats,
  useArchiveAgentChat,
  useCreateAgentChat,
  useSendAgentChatMessage,
} from "@/hooks/useAgentChats";
import { useAgentSessionWebsocket } from "@/hooks/useAgentSessionWebsocket";
import { formatTimeAgo } from "@/lib/date";
import type { AgentChat, AgentMessage } from "./types";
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

type Mode = { kind: "list" } | { kind: "compose" } | { kind: "open"; chatId: string };

function OpenAgentSidebar({ agentState }: AgentSidebarProps) {
  const canvasId = agentState.canvasId ?? "";
  const organizationId = agentState.organizationId ?? "";

  const chatsQuery = useAgentChats(canvasId, organizationId, agentState.isAgentSidebarOpen);
  const [mode, setMode] = useState<Mode>({ kind: "list" });

  useEffect(() => {
    setMode({ kind: "list" });
  }, [canvasId]);

  const goToList = useCallback(() => setMode({ kind: "list" }), []);
  const goToCompose = useCallback(() => setMode({ kind: "compose" }), []);
  const goToChat = useCallback((id: string) => setMode({ kind: "open", chatId: id }), []);

  if (mode.kind === "list") {
    return (
      <SidebarShell title="Agent" onClose={agentState.closeSidebar}>
        <ChatList
          chats={chatsQuery.data ?? []}
          loading={chatsQuery.isLoading}
          onSelect={goToChat}
          onStartCompose={goToCompose}
        />
      </SidebarShell>
    );
  }

  if (mode.kind === "compose") {
    return (
      <SidebarShell title="New conversation" onClose={agentState.closeSidebar} onBack={goToList}>
        <ChatConversation
          chatId={null}
          canvasId={canvasId}
          organizationId={organizationId}
          onCreated={goToChat}
          onArchived={goToList}
        />
      </SidebarShell>
    );
  }

  const currentChat = chatsQuery.data?.find((c) => c.id === mode.chatId) ?? null;
  return (
    <SidebarShell
      title={currentChat?.title?.trim() || "Agent chat"}
      onClose={agentState.closeSidebar}
      onBack={goToList}
    >
      <ChatConversation
        chatId={mode.chatId}
        canvasId={canvasId}
        organizationId={organizationId}
        onArchived={goToList}
      />
    </SidebarShell>
  );
}

function SidebarShell({
  title,
  children,
  onClose,
  onBack,
}: {
  title: string;
  children: React.ReactNode;
  onClose: () => void;
  onBack?: () => void;
}) {
  const { sidebarRef, width, isResizing, handleMouseDown } = useSidebarWidth();

  return (
    <aside
      ref={sidebarRef}
      data-testid="agent-sidebar"
      className="relative border-r border-border shrink-0 h-full z-21 flex flex-col overflow-hidden bg-white"
      style={{ width }}
    >
      <header className="flex items-center justify-between gap-3 px-4 py-2.5 border-b border-border shrink-0 min-w-0">
        <div className="flex min-w-0 flex-1 items-center gap-1">
          {onBack ? (
            <button
              type="button"
              onClick={onBack}
              aria-label="Back to chat list"
              data-testid="agent-back-button"
              className="z-40 shrink-0 w-6 h-6 hover:bg-slate-950/5 rounded-md flex items-center justify-center cursor-pointer text-muted-foreground"
            >
              <ChevronLeft size={18} />
            </button>
          ) : null}
          <h2 className="text-base font-medium min-w-0 flex-1 truncate" title={title}>
            {title}
          </h2>
        </div>
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

function ChatList({
  chats,
  loading,
  onSelect,
  onStartCompose,
}: {
  chats: AgentChat[];
  loading: boolean;
  onSelect: (chatId: string) => void;
  onStartCompose: () => void;
}) {
  return (
    <div className="flex flex-col flex-1 min-h-0 overflow-hidden">
      <div className="p-3 border-b border-border">
        <Button type="button" onClick={onStartCompose} className="w-full" data-testid="agent-new-chat-button">
          <Sparkles className="size-4" />
          New conversation
        </Button>
      </div>
      <div className="flex-1 min-h-0 overflow-y-auto p-2">
        {loading ? (
          <div className="flex items-center justify-center py-8 text-sm text-muted-foreground">
            <Loader2 className="size-4 animate-spin mr-2" /> Loading conversations…
          </div>
        ) : chats.length === 0 ? (
          <p className="text-sm text-muted-foreground px-3 py-6 text-center">
            Start a conversation to ask the agent for help building this canvas.
          </p>
        ) : (
          <ul className="space-y-1" data-testid="agent-chat-list">
            {chats.map((chat) => (
              <ChatListRow key={chat.id} chat={chat} onSelect={onSelect} />
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

function ChatConversation({
  chatId,
  canvasId,
  organizationId,
  onArchived,
  onCreated,
}: {
  chatId: string | null;
  canvasId: string;
  organizationId: string;
  onArchived: () => void;
  onCreated?: (chatId: string) => void;
}) {
  const composing = chatId === null;
  const messagesQuery = useAgentChatMessages(chatId, organizationId, !composing);
  const createMutation = useCreateAgentChat(canvasId, organizationId);
  const sendMutation = useSendAgentChatMessage(organizationId, canvasId);
  const archiveMutation = useArchiveAgentChat(canvasId, organizationId);

  const [draft, setDraft] = useState("");
  const [streamingText, setStreamingText] = useState("");
  const [status, setStatus] = useState<string>("idle");
  const [error, setError] = useState<string | null>(null);

  // Show only running tools — completed/failed rows would just be noise.
  const messages = (messagesQuery.data ?? []).filter(
    (m) => m.role !== "tool" || (m.toolStatus !== "finished" && m.toolStatus !== "failed"),
  );
  const hasRunningTool = messages.some((m) => m.role === "tool");
  const showThinking = status === "streaming" && !streamingText && !hasRunningTool;

  const wsCallbacks = useMemo(
    () => ({
      onAssistantDelta: (text: string) => setStreamingText((prev) => prev + text),
      onPersistedMessage: () => setStreamingText(""),
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
    if (!value || sendMutation.isPending || createMutation.isPending) return;
    setDraft("");
    setError(null);
    try {
      let targetChatId = chatId;
      if (targetChatId === null) {
        const chat = await createMutation.mutateAsync();
        if (!chat) throw new Error("failed to create chat");
        targetChatId = chat.id;
        onCreated?.(chat.id);
      }
      await sendMutation.mutateAsync({ chatId: targetChatId, content: value });
    } catch (err) {
      setError(err instanceof Error ? err.message : "failed to send message");
    }
  }, [chatId, createMutation, draft, onCreated, sendMutation]);

  const handleArchive = useCallback(async () => {
    if (!chatId) return;
    await archiveMutation.mutateAsync(chatId);
    onArchived();
  }, [archiveMutation, chatId, onArchived]);

  const scrollRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight, behavior: "smooth" });
  }, [messages.length, streamingText, showThinking]);

  return (
    <div className="flex flex-col flex-1 min-h-0">
      <div ref={scrollRef} className="flex-1 min-h-0 overflow-y-auto p-3 space-y-2" data-testid="agent-chat-messages">
        {messagesQuery.isLoading ? (
          <div className="flex items-center justify-center py-8 text-sm text-muted-foreground">
            <Loader2 className="size-4 animate-spin mr-2" /> Loading…
          </div>
        ) : (
          messages.map((m) => <MessageRow key={m.id} message={m} />)
        )}
        {streamingText ? <StreamingAssistantRow text={streamingText} /> : null}
        {showThinking ? <ThinkingRow /> : null}
        {error ? <p className="text-sm text-red-600 px-3 py-2">{error}</p> : null}
      </div>
      <ChatComposer
        draft={draft}
        onDraftChange={setDraft}
        onSend={handleSend}
        onArchive={composing ? undefined : handleArchive}
        sending={sendMutation.isPending || createMutation.isPending}
        archiving={archiveMutation.isPending}
        statusLabel={statusLabel(status)}
      />
    </div>
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

function ChatListRow({ chat, onSelect }: { chat: AgentChat; onSelect: (chatId: string) => void }) {
  const stamp = chat.updatedAt ?? chat.createdAt;
  const timeAgo = stamp ? formatTimeAgo(new Date(stamp)) : null;
  return (
    <li>
      <button
        type="button"
        onClick={() => onSelect(chat.id)}
        className="w-full text-left px-3 py-2 rounded-md hover:bg-slate-950/5 text-sm flex items-start gap-2"
      >
        <div className="flex-1 min-w-0 flex flex-col">
          <span className="truncate">{chat.title?.trim() || "Untitled chat"}</span>
          <span className="text-xs text-muted-foreground">{chat.status}</span>
        </div>
        {timeAgo ? <span className="text-xs text-muted-foreground/70 shrink-0 mt-0.5">{timeAgo}</span> : null}
      </button>
    </li>
  );
}

function ChatComposer({
  draft,
  onDraftChange,
  onSend,
  onArchive,
  sending,
  archiving,
  statusLabel,
}: {
  draft: string;
  onDraftChange: (value: string) => void;
  onSend: () => void;
  onArchive?: () => void;
  sending: boolean;
  archiving: boolean;
  statusLabel: string;
}) {
  return (
    <footer className="border-t border-border p-3 flex flex-col gap-2">
      <textarea
        value={draft}
        onChange={(e) => onDraftChange(e.target.value)}
        rows={3}
        placeholder="Ask the agent…"
        data-testid="agent-input"
        className="resize-none w-full border border-border rounded-md p-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        onKeyDown={(e) => {
          if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
            e.preventDefault();
            onSend();
          }
        }}
      />
      <div className="flex items-center justify-between">
        <span className="text-xs text-muted-foreground">{statusLabel}</span>
        <div className="flex gap-2">
          {onArchive ? (
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  onClick={onArchive}
                  disabled={archiving}
                  data-testid="agent-archive-chat-button"
                  aria-label="Archive conversation"
                >
                  <Trash2 className="size-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Archive this conversation</TooltipContent>
            </Tooltip>
          ) : null}
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
      </div>
    </footer>
  );
}

function ToolMessageRow({ message }: { message: AgentMessage }) {
  const [expanded, setExpanded] = useState(false);
  const command = message.content;
  const canExpand = Boolean(command);
  return (
    <div
      className="flex items-start gap-2 text-sm py-1 text-slate-500 animate-tool-glow"
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
          Running command
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

function StreamingAssistantRow({ text }: { text: string }) {
  return (
    <div className="flex justify-start">
      <div
        className="rounded-lg px-3 py-2 text-sm max-w-[85%] break-words bg-slate-100 text-slate-900"
        data-testid="agent-streaming-message"
      >
        <AgentMarkdown content={text} />
        <span className="inline-block w-1.5 h-3 bg-slate-500 animate-pulse ml-1 align-middle" />
      </div>
    </div>
  );
}

// Mirrors WorkflowMarkdownPreview's class set.
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
      return "Agent is thinking…";
    case "failed":
      return "Last turn failed";
    case "terminated":
      return "Session ended";
    default:
      return "Ready";
  }
}
