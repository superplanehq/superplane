import { Bot, ChevronRight, ExternalLink, Loader2, Maximize2, Terminal } from "lucide-react";
import { memo, useCallback, useEffect, useMemo, useState, type RefObject } from "react";
import { isSystemNotification } from "@/components/AgentSidebar/systemMessages";
import type { RubricCategory } from "@/components/AgentSidebar/widgets/parser";
import { RichMessage } from "@/components/AgentSidebar/widgets/RichMessage";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogTitle } from "@/components/ui/dialog";
import { cn } from "@/lib/utils";
import type { AgentMessage } from "./types";
import type { MessageGroup } from "./agentMessageGroups";

const STICKY_USER_MESSAGE_MAX_CHARS = 240;
const STICKY_USER_MESSAGE_MAX_LINES = 4;
type MessageImage = NonNullable<AgentMessage["images"]>[number];

export const ConversationTranscript = memo(function ConversationTranscript({
  error,
  notice,
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
  notice?: string | null;
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
  // Slice the flat group list into "turns" (user message + everything that follows it until the
  // next user message). Each turn is its own block so the sticky user bubble inside is bounded by
  // its turn — when the turn scrolls past, the bubble scrolls with it and the next turn's bubble
  // pushes up to take its place. No two stickies ever overlap.
  const turns = useMemo(() => chunkIntoTurns(messageGroups.filter(isRenderableGroup)), [messageGroups]);

  return (
    <div ref={scrollRef} className="min-h-0 min-w-0 flex-1 overflow-y-auto px-3" data-testid="agent-chat-messages">
      <div className="mx-auto w-full max-w-[800px] py-3">
        {isLoading ? (
          <LoadingState label="Loading…" />
        ) : (
          <>
            {isLoadingMore ? <LoadingOlderMessages /> : null}
            {turns.map((turn) => (
              <div key={turnKey(turn)} className="space-y-2 [&+&]:mt-2">
                {turn.map((group) => (
                  <ConversationGroup
                    key={group.type === "message" ? group.message.id : group.messages[0].id}
                    group={group}
                    canvasId={canvasId}
                    organizationId={organizationId}
                    onAction={onAction}
                    onStartBuilding={onStartBuilding}
                  />
                ))}
              </div>
            ))}
          </>
        )}
        {showThinking ? <ThinkingRow /> : null}
        {notice ? <p className="px-3 py-2 text-sm text-amber-600">{notice}</p> : null}
        {error ? <p className="px-3 py-2 text-sm text-red-600">{error}</p> : null}
      </div>
    </div>
  );
});

function chunkIntoTurns(groups: MessageGroup[]): MessageGroup[][] {
  const turns: MessageGroup[][] = [];
  let current: MessageGroup[] = [];

  for (const group of groups) {
    const startsNewTurn =
      group.type === "message" && group.message.role === "user" && !isSystemNotification(group.message.content);

    if (startsNewTurn && current.length > 0) {
      turns.push(current);
      current = [];
    }

    current.push(group);
  }

  if (current.length > 0) turns.push(current);
  return turns;
}

function isRenderableGroup(group: MessageGroup): boolean {
  if (group.type !== "message") {
    return true;
  }

  return shouldRenderMessage(group.message);
}

function turnKey(turn: MessageGroup[]): string {
  const head = turn[0];
  return head.type === "message" ? head.message.id : head.messages[0].id;
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

  if (!shouldRenderMessage(message)) {
    return null;
  }

  const isUser = message.role === "user";
  const shouldStickUserMessage = isUser && isCompactUserMessage(message);

  return (
    <div
      className={cn(
        "flex w-full min-w-0 flex-col",
        isUser && "items-end py-1.5",
        !isUser && "items-start",
        // Compact user bubbles stick to the top of the scrollable transcript so the current prompt
        // remains visible while a long agent reply scrolls past. Long prompts must scroll normally;
        // otherwise the sticky bubble can cover the active Thinking or command rows.
        shouldStickUserMessage && "sticky top-0 z-10 bg-white dark:bg-gray-900",
      )}
    >
      <div
        className={cn(
          "min-w-0 break-words text-sm",
          isUser
            ? "max-w-[85%] rounded-lg bg-slate-100 px-3 py-1.5 whitespace-pre-wrap text-slate-900 dark:bg-gray-800 dark:text-gray-100"
            : "w-full max-w-[720px] text-slate-900 dark:text-gray-100",
        )}
        data-testid={isUser ? "agent-user-message" : "agent-assistant-message"}
      >
        <MessageImages images={message.images} />
        <RichMessage
          content={message.content}
          onAction={isUser ? undefined : onAction}
          onStartBuilding={isUser ? undefined : onStartBuilding}
          canvasId={canvasId}
          organizationId={organizationId}
        />
      </div>
      {message.createdAt ? (
        <span className="mt-0.5 text-[10px] text-slate-500 dark:text-gray-400">{formatTime(message.createdAt)}</span>
      ) : null}
    </div>
  );
});

function MessageImages({ images }: { images: AgentMessage["images"] }) {
  const [selectedImage, setSelectedImage] = useState<MessageImage | null>(null);

  if (!images || images.length === 0) return null;

  return (
    <>
      <div className="mb-1.5 flex flex-wrap gap-1.5" data-testid="agent-message-images">
        {images.map((image, index) => (
          <button
            key={index}
            type="button"
            onClick={() => setSelectedImage(image)}
            className="group relative block cursor-zoom-in overflow-hidden rounded-md border border-slate-200 dark:border-gray-700"
            aria-label="Open attachment"
          >
            <img src={image.url} alt="attachment" className="max-h-40 max-w-[200px] object-contain" />
            <span className="absolute right-1 bottom-1 rounded bg-slate-950/70 p-1 text-white opacity-0 transition-opacity group-hover:opacity-100 group-focus-visible:opacity-100">
              <Maximize2 className="size-3" aria-hidden />
            </span>
          </button>
        ))}
      </div>
      <ImageLightbox image={selectedImage} onOpenChange={(open) => !open && setSelectedImage(null)} />
    </>
  );
}

function ImageLightbox({ image, onOpenChange }: { image: MessageImage | null; onOpenChange: (open: boolean) => void }) {
  return (
    <Dialog open={!!image} onOpenChange={onOpenChange}>
      <DialogContent
        size="large"
        className="flex max-h-[calc(100dvh-2rem)] w-[calc(100vw-2rem)] max-w-[1400px] grid-rows-none flex-col gap-3 overflow-hidden p-3 sm:max-h-[calc(100dvh-4rem)] sm:w-[calc(100vw-4rem)] sm:p-4"
      >
        <DialogTitle className="sr-only">Image attachment</DialogTitle>
        <DialogDescription className="sr-only">Expanded image attachment from the agent session.</DialogDescription>
        {image ? (
          <>
            <div className="min-h-0 flex-1 overflow-auto rounded-md bg-slate-950/5 dark:bg-black/30">
              <img
                src={image.url}
                alt="attachment"
                className="mx-auto h-auto max-h-[calc(100dvh-7rem)] max-w-full object-contain sm:max-h-[calc(100dvh-9rem)]"
              />
            </div>
            <div className="flex shrink-0 justify-end">
              <Button asChild variant="outline" size="sm">
                <a href={image.url} target="_blank" rel="noreferrer">
                  <ExternalLink className="size-3.5" />
                  Open original
                </a>
              </Button>
            </div>
          </>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

function shouldRenderMessage(message: AgentMessage): boolean {
  return message.role !== "system" && !(message.role === "user" && isSystemNotification(message.content));
}

function isCompactUserMessage(message: AgentMessage): boolean {
  // Messages with image attachments render tall thumbnails; pinning them would
  // cover in-progress agent output, so they never stick regardless of text length.
  if (message.images && message.images.length > 0) {
    return false;
  }
  const content = message.content;
  return content.length <= STICKY_USER_MESSAGE_MAX_CHARS && content.split("\n").length <= STICKY_USER_MESSAGE_MAX_LINES;
}

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
        className="flex cursor-pointer items-center gap-2 text-slate-700 hover:text-slate-900 dark:text-gray-300 dark:hover:text-gray-100"
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
      {question ? (
        <p className="text-xs italic text-slate-500 dark:text-gray-400">"{truncateQuestion(question)}"</p>
      ) : null}
      {response ? (
        <div className="max-h-60 overflow-y-auto">
          <p className="whitespace-pre-wrap text-xs text-slate-700 dark:text-gray-300">{response}</p>
        </div>
      ) : null}
    </div>
  );
}

function truncateQuestion(question: string): string {
  return question.length > 200 ? `${question.slice(0, 200)}…` : question;
}

// The `superplane_app` tool's `patch_staging` action edits the canvas — label it
// "Editing canvas" rather than a generic "Running command".
function isCanvasEditMessage(message: AgentMessage): boolean {
  if (message.toolName !== "superplane_app") return false;
  try {
    return (JSON.parse(message.content) as { action?: string })?.action === "patch_staging";
  } catch {
    return message.content.includes("patch_staging");
  }
}

function ToolGroupRow({ messages }: { messages: AgentMessage[] }) {
  const hasRunning = messages.some((message) => message.toolStatus === "started");
  const editRunning = messages.some((message) => message.toolStatus === "started" && isCanvasEditMessage(message));
  const editedAny = messages.some(isCanvasEditMessage);
  const [expanded, setExpanded] = useState(hasRunning);
  const count = messages.length;
  let label: string;
  if (hasRunning) {
    // Only call it "Editing canvas" when an edit is the tool actually running,
    // so a finished edit next to another running tool isn't mislabeled.
    label = editRunning ? "Editing canvas…" : `Running command${count > 1 ? ` (${count})` : ""}...`;
  } else {
    label = editedAny ? "Edited canvas" : `Ran ${count} command${count !== 1 ? "s" : ""}`;
  }

  useEffect(() => {
    setExpanded(hasRunning);
  }, [hasRunning]);

  return (
    <div className={cn("py-1 text-xs", hasRunning && "animate-tool-glow")} data-testid="agent-tool-group">
      <button
        type="button"
        onClick={() => setExpanded((current) => !current)}
        className="group flex cursor-pointer items-center gap-2"
      >
        <Terminal className="size-4 shrink-0 text-slate-500 group-hover:text-slate-800 dark:text-gray-400 dark:group-hover:text-gray-200" />
        <span className="text-slate-500 group-hover:text-slate-800 dark:text-gray-400 dark:group-hover:text-gray-200">
          {label}
        </span>
        <ChevronRight
          className={cn(
            "size-3 text-slate-500 transition-transform group-hover:text-slate-800 dark:text-gray-400 dark:group-hover:text-gray-200",
            expanded && "rotate-90",
          )}
        />
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
          running ? "text-slate-700 dark:text-gray-300" : "text-slate-600 dark:text-gray-400",
          canExpand && "cursor-pointer hover:text-slate-900 dark:hover:text-gray-200",
        )}
      >
        <span className="shrink-0 text-[10px]">{running ? "▶" : "✓"}</span>
        <span className="truncate">{running ? "Running..." : preview}</span>
      </button>
      {expanded && command ? (
        <div className="mt-1 overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-gray-800/70 dark:bg-gray-900">
          <div className="flex items-center justify-between border-b border-slate-200 bg-slate-50 px-3 py-1 dark:border-gray-800/70 dark:bg-gray-900">
            <span className="text-[10px] font-medium uppercase tracking-wider text-slate-500 dark:text-gray-400">
              bash
            </span>
          </div>
          <pre className="max-h-[200px] overflow-auto break-words whitespace-pre-wrap p-3 font-mono text-xs text-slate-700 dark:text-gray-300">
            {command}
          </pre>
        </div>
      ) : null}
    </div>
  );
}

function ThinkingRow() {
  return (
    <div
      className="flex animate-tool-glow items-center gap-2 py-1 text-sm text-slate-500 dark:text-gray-400"
      data-testid="agent-thinking"
    >
      <Loader2 className="size-4 shrink-0 animate-spin" />
      <span>Thinking…</span>
    </div>
  );
}

function formatTime(iso: string): string {
  const date = new Date(iso);
  return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}
