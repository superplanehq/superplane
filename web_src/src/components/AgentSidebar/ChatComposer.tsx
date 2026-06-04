import { ArrowUp, Loader2, Square } from "lucide-react";
import { useCallback, useMemo, useRef } from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { AgentMode } from "./agentMode";
import { ModeToggle } from "./ModeToggle";
import { useMentions } from "./useMentions";
import { MentionDropdown, type MentionCandidate } from "./MentionDropdown";
import type { SuperplaneComponentsNode } from "@/api-client";
import type { CanvasesCanvasRun } from "@/api-client";

function timeAgo(dateStr?: string): string {
  if (!dateStr) return "";
  const diff = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

/**
 * Renders the text with @mentions styled as inline chips.
 * Non-mention text is rendered transparent (invisible — shows through from textarea caret).
 * Mention spans are styled as visible pills.
 */
function BackdropContent({ text, mentions }: { text: string; mentions: { label: string }[] }) {
  if (mentions.length === 0) {
    // No mentions — render the text normally to maintain layout
    return <span className="whitespace-pre-wrap break-words text-[rgba(10,10,10,1)]">{text || "\u00A0"}</span>;
  }

  // Build segments using tracked startIndex positions for accurate rendering
  const sorted = [...mentions]
    .filter((m) => {
      const expected = `@${m.label}`;
      return text.slice(m.startIndex, m.startIndex + expected.length) === expected;
    })
    .sort((a, b) => a.startIndex - b.startIndex);

  const segments: { text: string; isMention: boolean }[] = [];
  let pos = 0;

  for (const m of sorted) {
    const displayText = `@${m.label}`;
    if (m.startIndex > pos) {
      segments.push({ text: text.slice(pos, m.startIndex), isMention: false });
    }
    segments.push({ text: displayText, isMention: true });
    pos = m.startIndex + displayText.length;
  }

  if (pos < text.length) {
    segments.push({ text: text.slice(pos), isMention: false });
  }

  return (
    <>
      {segments.map((seg, i) =>
        seg.isMention ? (
          <span
            key={i}
            className="inline-flex items-center rounded bg-blue-100 px-1 py-0.5 text-xs font-medium text-blue-700"
          >
            {seg.text}
          </span>
        ) : (
          <span key={i} className="whitespace-pre-wrap text-[rgba(10,10,10,1)]">
            {seg.text}
          </span>
        ),
      )}
    </>
  );
}

export function ChatComposer({
  onSend,
  onStop,
  sending,
  sendPending,
  stopping,
  statusLabel,
  agentMode,
  onModeSwitch,
  modeDisabled,
  nodes,
  runs,
}: {
  onSend: (content: string) => Promise<void>;
  onStop: () => void;
  sending: boolean;
  sendPending: boolean;
  stopping?: boolean;
  statusLabel: string;
  agentMode: AgentMode;
  onModeSwitch: (mode: AgentMode) => void;
  modeDisabled?: boolean;
  nodes?: SuperplaneComponentsNode[];
  runs?: CanvasesCanvasRun[];
}) {
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const backdropRef = useRef<HTMLDivElement>(null);
  const mentionKeyboardRef = useRef<((e: React.KeyboardEvent) => boolean) | null>(null);
  const {
    value,
    setValue,
    showDropdown,
    filter,
    setCursorPos,
    insertMention,
    getMarkdown,
    mentions,
    isEmpty,
    clear,
    snapshot,
    restore,
    dismiss,
  } = useMentions();

  // Build mention candidates from nodes + runs, filtered by current input
  const candidates = useMemo((): MentionCandidate[] => {
    const filterLower = filter.toLowerCase();
    const result: MentionCandidate[] = [];

    // Nodes
    if (nodes) {
      for (const node of nodes) {
        const name = node.name || node.id || "";
        if (filterLower && !name.toLowerCase().includes(filterLower)) continue;
        result.push({
          type: "node",
          id: node.id || "",
          label: name,
          meta: node.component,
          isTrigger: node.type === "TYPE_TRIGGER",
        });
      }
    }

    // Runs (show recent 10)
    if (runs) {
      const recentRuns = runs.slice(0, 10);
      for (const run of recentRuns) {
        const label = `Run #${run.id?.slice(0, 6) || "?"}`;
        if (filterLower && !label.toLowerCase().includes(filterLower)) continue;
        result.push({
          type: "run",
          id: run.id || "",
          label,
          meta: run.result || run.state,
          timeAgo: timeAgo(run.createdAt),
        });
      }
    }

    return result;
  }, [nodes, runs, filter]);

  const canSend = !isEmpty && !sendPending;

  const handleSend = useCallback(async () => {
    if (isEmpty) return;
    const content = getMarkdown().trim();
    if (!content) return;

    snapshot();
    clear();

    try {
      await onSend(content);
    } catch {
      restore();
    }
  }, [isEmpty, getMarkdown, clear, onSend, snapshot, restore]);

  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLTextAreaElement>) => {
      setValue(e.target.value);
      setCursorPos(e.target.selectionStart ?? e.target.value.length);
    },
    [setValue, setCursorPos],
  );

  const handleSelect = useCallback(
    (e: React.SyntheticEvent<HTMLTextAreaElement>) => {
      setCursorPos((e.target as HTMLTextAreaElement).selectionStart ?? 0);
    },
    [setCursorPos],
  );

  const handleMentionSelect = useCallback(
    (item: { type: "node" | "run"; id: string; label: string; meta?: string }) => {
      const newCursorPos = insertMention(item);
      // Set DOM caret position after React re-renders
      requestAnimationFrame(() => {
        const ta = textareaRef.current;
        if (ta) {
          ta.focus();
          ta.setSelectionRange(newCursorPos, newCursorPos);
        }
      });
    },
    [insertMention],
  );

  const handleDismiss = useCallback(() => {
    dismiss();
    textareaRef.current?.focus();
  }, [dismiss]);

  // Sync scroll between textarea and backdrop
  const handleScroll = useCallback(() => {
    if (textareaRef.current && backdropRef.current) {
      backdropRef.current.scrollTop = textareaRef.current.scrollTop;
      backdropRef.current.scrollLeft = textareaRef.current.scrollLeft;
    }
  }, []);

  return (
    <footer className="px-3 pb-3 pt-2">
      <div
        ref={containerRef}
        className="mx-auto w-full max-w-[800px] overflow-hidden rounded-lg bg-white shadow-sm outline outline-1 outline-slate-950/15"
      >
        {/* Textarea wrapper with backdrop overlay */}
        <div className="relative">
          {/* Backdrop: renders styled text + mention chips behind transparent textarea */}
          <div
            ref={backdropRef}
            aria-hidden="true"
            className={cn(
              "pointer-events-none absolute inset-0 whitespace-pre-wrap break-words overflow-hidden",
              "px-3 py-2.5 text-sm",
            )}
          >
            <BackdropContent text={value} mentions={mentions} />
          </div>
          {/* Textarea — text is transparent when mentions exist so chips show through */}
          <textarea
            ref={textareaRef}
            value={value}
            onChange={handleChange}
            onSelect={handleSelect}
            onKeyUp={handleSelect}
            onClick={handleSelect}
            onScroll={handleScroll}
            rows={1}
            placeholder="Ask the agent…"
            data-testid="agent-input"
            className={cn(
              "relative min-h-9 w-full resize-none border-0 bg-transparent px-3 py-2.5 text-sm shadow-none",
              "outline-none ring-0 focus-visible:border-0 focus-visible:ring-0 focus-visible:outline-none",
              "placeholder:text-muted-foreground disabled:cursor-not-allowed disabled:opacity-50",
              // Always transparent — backdrop provides visual text
              "text-transparent caret-slate-900 selection:bg-blue-200/50",
              "dark:bg-transparent",
            )}
            onKeyDown={(e) => {
              // Let MentionDropdown handle keys when visible
              if (mentionKeyboardRef.current?.(e)) {
                return;
              }
              if (e.key !== "Enter") return;
              const nativeEvent = e.nativeEvent;
              if ("isComposing" in nativeEvent && nativeEvent.isComposing) return;
              if (e.shiftKey) return;
              e.preventDefault();
              if (!canSend) return;
              void handleSend();
            }}
          />
        </div>
        <div className="flex items-center justify-between gap-2 px-2 pb-2">
          <ModeToggle mode={agentMode} onSwitch={onModeSwitch} disabled={modeDisabled} streaming={sending} />
          <div className="flex min-w-0 shrink-0 items-center gap-2">
            <span className="truncate text-xs text-muted-foreground">{statusLabel}</span>
            {sending && (
              <Button
                type="button"
                variant="outline"
                size="icon"
                className="size-7 shrink-0 rounded-full border-slate-300 bg-white text-slate-700 hover:bg-slate-100"
                onClick={onStop}
                disabled={stopping}
                aria-label={stopping ? "Stopping" : "Stop"}
                title={stopping ? "Stopping..." : "Stop"}
                data-testid="agent-stop-button"
              >
                {stopping ? (
                  <Loader2 className="size-3.5 animate-spin" aria-hidden />
                ) : (
                  <Square className="size-3 fill-current" aria-hidden />
                )}
              </Button>
            )}
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
              <ArrowUp className="size-3.5" aria-hidden />
            </Button>
          </div>
        </div>
      </div>

      <MentionDropdown
        items={candidates}
        visible={showDropdown}
        anchorEl={containerRef.current}
        onSelect={handleMentionSelect}
        onDismiss={handleDismiss}
        keyboardRef={mentionKeyboardRef}
      />
    </footer>
  );
}
