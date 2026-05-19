import { Bot, ChevronRight, Loader2, SquareTerminal } from "lucide-react";
import { memo, useCallback, useEffect, useState, type RefObject } from "react";
import { formatSystemNotification, isSystemNotification } from "@/components/AgentSidebar/systemMessages";
import type { RubricCategory } from "@/components/AgentSidebar/widgets/parser";
import { RichMessage } from "@/components/AgentSidebar/widgets/RichMessage";
import { cn } from "@/lib/utils";
import type { AgentMessage } from "./types";
import type { MessageGroup } from "./agentMessageGroups";

export function ConversationTranscript({
  error,
  canvasId,
  organizationId,
  messageGroups,
  isLoading,
  isLoadingMore,
  onAction,
  onStartBuilding,
  scrollRef,
  showThinking,
}: {
  error: string | null;
  canvasId: string;
  organizationId: string;
  messageGroups: MessageGroup[];
  isLoading: boolean;
  isLoadingMore: boolean;
  onAction: (action: string) => Promise<void>;
  onStartBuilding: (rubric: { title: string; criteria: string[]; categories?: RubricCategory[] }) => Promise<void>;
  scrollRef: RefObject<HTMLDivElement | null>;
  showThinking: boolean;
}) {
  return (
    <div ref={scrollRef} className="min-h-0 flex-1 space-y-2 overflow-y-auto p-3" data-testid="agent-chat-messages">
      {isLoading ? (
        <LoadingState label="Loading…" />
      ) : (
        <>
          {isLoadingMore ? <LoadingOlderMessages /> : null}
          {messageGroups.map((group) => (
            <ConversationGroup
              key={group.type === "message" ? group.message.id : group.messages[0].id}
              group={group}
              canvasId={canvasId}
              organizationId={organizationId}
              onAction={onAction}
              onStartBuilding={onStartBuilding}
            />
          ))}
        </>
      )}
      {showThinking ? <ThinkingRow /> : null}
      {error ? <p className="px-3 py-2 text-sm text-red-600">{error}</p> : null}
    </div>
  );
}

function ConversationGroup({
  group,
  canvasId,
  organizationId,
  onAction,
  onStartBuilding,
}: {
  group: MessageGroup;
  canvasId: string;
  organizationId: string;
  onAction: (action: string) => Promise<void>;
  onStartBuilding: (rubric: { title: string; criteria: string[]; categories?: RubricCategory[] }) => Promise<void>;
}) {
  if (group.type === "tool-group") {
    return <ToolGroupRow messages={group.messages} />;
  }

  if (group.type === "subagent-group") {
    return <SubagentCard messages={group.messages} />;
  }

  return (
    <MessageRow
      message={group.message}
      canvasId={canvasId}
      organizationId={organizationId}
      onAction={onAction}
      onStartBuilding={onStartBuilding}
    />
  );
}

function LoadingState({ label }: { label: string }) {
  return (
    <div className="flex items-center justify-center py-8 text-sm text-muted-foreground">
      <Loader2 className="mr-2 size-4 animate-spin" /> {label}
    </div>
  );
}

function LoadingOlderMessages() {
  return (
    <div className="flex items-center justify-center py-2 text-xs text-muted-foreground">
      <Loader2 className="mr-2 size-3 animate-spin" /> Loading older messages…
    </div>
  );
}

const MessageRow = memo(function MessageRow({
  message,
  canvasId,
  organizationId,
  onAction,
  onStartBuilding,
}: {
  message: AgentMessage;
  canvasId: string;
  organizationId: string;
  onAction: (action: string) => Promise<void>;
  onStartBuilding: (rubric: { title: string; criteria: string[]; categories?: RubricCategory[] }) => Promise<void>;
}) {
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
            onAction={onAction}
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
});

function SubagentCard({ messages }: { messages: AgentMessage[] }) {
  const [expanded, setExpanded] = useState(false);
  const { agentName, isRunning, question, response } = getSubagentSummary(messages);
  const toggleExpanded = useCallback(() => {
    setExpanded((current) => !current);
  }, []);

  return (
    <div className="py-1 text-sm" data-testid="subagent-card">
      <button
        type="button"
        onClick={toggleExpanded}
        className="flex cursor-pointer items-center gap-2 text-slate-700 hover:text-slate-900"
      >
        <Bot className="size-4 shrink-0" />
        <span>{agentName}</span>
        <SubagentStatus isRunning={isRunning} />
        <ChevronRight className={cn("size-3 transition-transform", expanded && "rotate-90")} />
      </button>
      {expanded ? <SubagentDetails question={question} response={response} /> : null}
    </div>
  );
}

function getSubagentSummary(messages: AgentMessage[]) {
  const sent = messages.find((message) => message.toolStatus === "started");
  const received = messages.find((message) => message.toolStatus === "finished");
  return {
    agentName: (sent?.toolName || received?.toolName || "subagent:").replace("subagent:", ""),
    isRunning: Boolean(sent) && !received,
    question: sent?.content || "",
    response: received?.content || "",
  };
}

function SubagentStatus({ isRunning }: { isRunning: boolean }) {
  return (
    <span className={cn("text-[10px] font-medium", isRunning ? "text-blue-600" : "text-emerald-600")}>
      {isRunning ? "Working…" : "Done"}
    </span>
  );
}

function SubagentDetails({ question, response }: { question: string; response: string }) {
  return (
    <div className="mt-2 space-y-2 pl-6">
      {question ? <p className="text-xs italic text-slate-500">"{truncateQuestion(question)}"</p> : null}
      {response ? (
        <div className="max-h-60 overflow-y-auto">
          <p className="whitespace-pre-wrap text-xs text-slate-700">{response}</p>
        </div>
      ) : null}
    </div>
  );
}

function truncateQuestion(question: string): string {
  return question.length > 200 ? `${question.slice(0, 200)}…` : question;
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

function formatTime(iso: string): string {
  const date = new Date(iso);
  return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}
