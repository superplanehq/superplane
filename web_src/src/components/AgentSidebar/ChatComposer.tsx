import { useCallback, useRef } from "react";
import type { AgentMode } from "./agentMode";
import { ComposerToolbar } from "./ComposerToolbar";
import { useMentions } from "./useMentions";
import { useMentionCandidates } from "./useMentionCandidates";
import { MentionDropdown } from "./MentionDropdown";
import { MentionTextarea } from "./MentionTextarea";
import { ImageAttachmentPreviews } from "./ImageAttachmentPreviews";
import { MAX_IMAGE_ATTACHMENTS, isSupportedImageFile, useImageAttachments } from "./useImageAttachments";
import { mimeToApiImageMediaType, type AgentOutgoingImage } from "@/components/CanvasToolSidebar/types";
import type { SuperplaneComponentsNode } from "@/api-client";
import type { CanvasesCanvasRun } from "@/api-client";

type ChatComposerProps = {
  onSend: (content: string, images: AgentOutgoingImage[]) => Promise<void>;
  onStop: () => void;
  onClearChat: () => void;
  clearing: boolean;
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

const modePlaceholder = {
  builder: "Describe the change to build...",
  operator: "Ask the agent…",
} as const;

export function ChatComposer({
  onSend,
  onStop,
  onClearChat,
  clearing,
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
  const c = useComposerController({ onSend, sendPending, nodes, runs });

  return (
    <footer className="px-3 pb-3 pt-2">
      <div
        ref={c.containerRef}
        className="mx-auto w-full max-w-[800px] overflow-hidden rounded-lg bg-white shadow-sm outline outline-1 outline-slate-950/15"
      >
        <ImageAttachmentPreviews images={c.images} onRemove={c.removeImage} />
        <MentionTextarea
          value={c.value}
          mentions={c.mentions}
          setValue={c.setValue}
          setCursorPos={c.setCursorPos}
          onKeyDown={c.handleKeyDown}
          onPaste={c.handlePaste}
          placeholder={modePlaceholder[agentMode]}
          textareaRef={c.textareaRef}
          backdropRef={c.backdropRef}
        />
        <ComposerToolbar
          agentMode={agentMode}
          onModeSwitch={onModeSwitch}
          modeDisabled={modeDisabled}
          onClearChat={onClearChat}
          clearing={clearing}
          sending={sending}
          stopping={stopping}
          statusLabel={statusLabel}
          canSend={c.canSend}
          canAttach={c.canAttach}
          onStop={onStop}
          onSend={c.handleToolbarSend}
          onAddFiles={c.addFiles}
        />
      </div>
      {c.showDropdown ? (
        <MentionDropdown
          items={c.candidates}
          visible={c.showDropdown}
          anchorEl={c.containerRef.current}
          onSelect={c.handleMentionSelect}
          onDismiss={c.handleDismiss}
          keyboardRef={c.mentionKeyboardRef}
        />
      ) : null}
    </footer>
  );
}

type ComposerControllerArgs = {
  onSend: (content: string, images: AgentOutgoingImage[]) => Promise<void>;
  sendPending: boolean;
  nodes?: SuperplaneComponentsNode[];
  runs?: CanvasesCanvasRun[];
};

function useComposerController({ onSend, sendPending, nodes, runs }: ComposerControllerArgs) {
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const backdropRef = useRef<HTMLDivElement>(null);
  const mentionKeyboardRef = useRef<((e: React.KeyboardEvent) => boolean) | null>(null);
  const mentionsApi = useMentions();
  const { value, setValue, showDropdown, filter, setCursorPos, getMarkdown, mentions, isEmpty } = mentionsApi;
  const { images, addFiles, removeImage, clear: clearImages } = useImageAttachments();

  const candidates = useMentionCandidates(nodes, runs, filter, showDropdown);
  const hasImages = images.length > 0;
  const canSend = (!isEmpty || hasImages) && !sendPending;
  const canAttach = images.length < MAX_IMAGE_ATTACHMENTS;

  const handleSend = useCallback(async () => {
    const content = getMarkdown().trim();
    if (!content && !hasImages) return;
    const outgoingImages = images.map(({ mediaType, data }) => ({
      mediaType: mimeToApiImageMediaType(mediaType),
      data,
    }));
    mentionsApi.snapshot();
    mentionsApi.clear();
    try {
      await onSend(content, outgoingImages);
      clearImages();
    } catch {
      mentionsApi.restore();
    }
  }, [hasImages, images, getMarkdown, clearImages, onSend, mentionsApi]);

  const handlePaste = useCallback(
    (e: React.ClipboardEvent<HTMLTextAreaElement>) => {
      const files = imageFilesFromClipboard(e);
      if (files.length === 0) return;
      if (e.clipboardData.getData("text/plain").length === 0) e.preventDefault();
      void addFiles(files);
    },
    [addFiles],
  );

  const handleMentionSelect = useCallback(
    (item: { type: "node" | "run"; id: string; label: string; meta?: string }) => {
      const pos = mentionsApi.insertMention(item);
      requestAnimationFrame(() => {
        const ta = textareaRef.current;
        if (ta) {
          ta.focus();
          ta.setSelectionRange(pos, pos);
        }
      });
    },
    [mentionsApi],
  );

  const handleDismiss = useCallback(() => {
    mentionsApi.dismiss();
    textareaRef.current?.focus();
  }, [mentionsApi]);

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

  return {
    textareaRef,
    containerRef,
    backdropRef,
    mentionKeyboardRef,
    value,
    setValue,
    setCursorPos,
    mentions,
    showDropdown,
    candidates,
    images,
    addFiles,
    removeImage,
    canSend,
    canAttach,
    handleSend,
    handlePaste,
    handleMentionSelect,
    handleDismiss,
    handleKeyDown,
    handleToolbarSend,
  };
}

function imageFilesFromClipboard(e: React.ClipboardEvent<HTMLTextAreaElement>): File[] {
  return Array.from(e.clipboardData.items)
    .filter((item) => item.kind === "file")
    .map((item) => item.getAsFile())
    .filter((file): file is File => file !== null && isSupportedImageFile(file));
}

function useStableCallback(callback: () => void): () => void {
  const callbackRef = useRef(callback);
  callbackRef.current = callback;

  return useCallback(() => callbackRef.current(), []);
}
