import { useCallback, useRef } from "react";
import type { AgentMode } from "./agentMode";
import { ComposerToolbar } from "./ComposerToolbar";
import { useMentions } from "./useMentions";
import { useMentionCandidates } from "./useMentionCandidates";
import { MentionDropdown } from "./MentionDropdown";
import { MentionTextarea } from "./MentionTextarea";
import type { SuperplaneComponentsNode } from "@/api-client";
import type { CanvasesCanvasRun } from "@/api-client";

type ChatComposerProps = {
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
};

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
}: ChatComposerProps) {
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

  const candidates = useMentionCandidates(nodes, runs, filter, showDropdown);
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

  const handleToolbarSend = useStableCallback(() => {
    void handleSend();
  });

  return (
    <footer className="px-3 pb-3 pt-2">
      <div
        ref={containerRef}
        className="mx-auto w-full max-w-[800px] overflow-hidden rounded-lg bg-white shadow-sm outline outline-1 outline-slate-950/15"
      >
        <MentionTextarea
          value={value}
          mentions={mentions}
          setValue={setValue}
          setCursorPos={setCursorPos}
          onKeyDown={handleKeyDown}
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
          onSend={handleToolbarSend}
        />
      </div>
      {showDropdown ? (
        <MentionDropdown
          items={candidates}
          visible={showDropdown}
          anchorEl={containerRef.current}
          onSelect={handleMentionSelect}
          onDismiss={handleDismiss}
          keyboardRef={mentionKeyboardRef}
        />
      ) : null}
    </footer>
  );
}

function useStableCallback(callback: () => void): () => void {
  const callbackRef = useRef(callback);
  callbackRef.current = callback;

  return useCallback(() => callbackRef.current(), []);
}
