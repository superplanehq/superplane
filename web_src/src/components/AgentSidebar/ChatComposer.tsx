import { useCallback, useRef } from "react";
import type { AgentMode } from "./agentMode";
import { ComposerToolbar } from "./ComposerToolbar";
import { useMentions } from "./useMentions";
import { useMentionCandidates } from "./useMentionCandidates";
import { MentionDropdown } from "./MentionDropdown";
import { MentionTextarea } from "./MentionTextarea";
import type { SuperplaneComponentsNode } from "@/api-client";
import type { CanvasesCanvasRun } from "@/api-client";

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

  const candidates = useMentionCandidates(nodes, runs, filter);
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
      const pos = insertMention(item);
      requestAnimationFrame(() => {
        const ta = textareaRef.current;
        if (ta) {
          ta.focus();
          ta.setSelectionRange(pos, pos);
        }
      });
    },
    [insertMention],
  );

  const handleDismiss = useCallback(() => {
    dismiss();
    textareaRef.current?.focus();
  }, [dismiss]);

  const handleScroll = useCallback(() => {
    if (textareaRef.current && backdropRef.current) {
      backdropRef.current.scrollTop = textareaRef.current.scrollTop;
      backdropRef.current.scrollLeft = textareaRef.current.scrollLeft;
    }
  }, []);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (mentionKeyboardRef.current?.(e)) return;
      if (e.key !== "Enter") return;
      if ("isComposing" in e.nativeEvent && e.nativeEvent.isComposing) return;
      if (e.shiftKey) return;
      e.preventDefault();
      if (canSend) void handleSend();
    },
    [canSend, handleSend],
  );

  return (
    <footer className="px-3 pb-3 pt-2">
      <div
        ref={containerRef}
        className="mx-auto w-full max-w-[800px] overflow-hidden rounded-lg bg-white shadow-sm outline outline-1 outline-slate-950/15"
      >
        <MentionTextarea
          value={value}
          mentions={mentions}
          onChange={handleChange}
          onSelect={handleSelect}
          onKeyDown={handleKeyDown}
          onScroll={handleScroll}
          placeholder="Ask the agent…"
          textareaRef={textareaRef}
          backdropRef={backdropRef}
        />
        <ComposerToolbar
          agentMode={agentMode}
          onModeSwitch={onModeSwitch}
          modeDisabled={modeDisabled}
          sending={sending}
          stopping={stopping}
          statusLabel={statusLabel}
          canSend={canSend}
          onStop={onStop}
          onSend={() => void handleSend()}
        />
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
