import { useCallback, useRef } from "react";
import type { SuperplaneComponentsNode } from "@/api-client";
import type { CanvasesCanvasRun } from "@/api-client";
import { MentionDropdown } from "@/components/AgentSidebar/MentionDropdown";
import { SkillSlashDropdown } from "@/components/AgentSidebar/SkillSlashMenu";
import { useFlushAgentComposerSend } from "@/components/AgentSidebar/useFlushAgentComposerSend";
import { useMentionCandidates } from "@/components/AgentSidebar/useMentionCandidates";
import { useMentions } from "@/components/AgentSidebar/useMentions";
import {
  MAX_IMAGE_ATTACHMENTS,
  isSupportedImageFile,
  useImageAttachments,
} from "@/components/AgentSidebar/useImageAttachments";
import { mimeToApiImageMediaType, type AgentOutgoingImage } from "@/components/CanvasToolSidebar/types";
import { useSkillSlashCandidates } from "@/hooks/useSkillSlashCandidates";
import type { SkillSlashCandidate } from "@/lib/skillSlash";
import { FactoryComposerToolbar } from "./FactoryComposerToolbar";
import { FactoryImageAttachmentPreviews } from "./FactoryImageAttachmentPreviews";
import { FactoryMentionTextarea } from "./FactoryMentionTextarea";

type ChatComposerProps = {
  canvasId: string;
  organizationId?: string;
  factoryId?: string;
  onSend: (content: string, images: AgentOutgoingImage[]) => Promise<void>;
  onStop: () => void;
  onClearChat: () => void;
  clearing: boolean;
  sending: boolean;
  sendPending: boolean;
  stopping?: boolean;
  statusLabel: string;
  nodes?: SuperplaneComponentsNode[];
  runs?: CanvasesCanvasRun[];
};

const COMPOSER_PLACEHOLDER = "Describe the change to build...";

export function FactoryChatComposer({
  canvasId,
  organizationId,
  factoryId,
  onSend,
  onStop,
  onClearChat,
  clearing,
  sending,
  sendPending,
  stopping,
  statusLabel,
  nodes,
  runs,
}: ChatComposerProps) {
  const c = useComposerController({ canvasId, organizationId, factoryId, onSend, sendPending, nodes, runs });

  return (
    <footer className="px-3 pb-3 pt-2">
      <div
        ref={c.containerRef}
        className="mx-auto w-full max-w-[800px] overflow-hidden rounded-lg bg-card shadow-sm outline outline-1 outline-border"
      >
        <FactoryImageAttachmentPreviews images={c.images} onRemove={c.removeImage} />
        <FactoryMentionTextarea
          value={c.value}
          mentions={c.mentions}
          setValue={c.setValue}
          setCursorPos={c.setCursorPos}
          onKeyDown={c.handleKeyDown}
          onPaste={c.handlePaste}
          placeholder={COMPOSER_PLACEHOLDER}
          textareaRef={c.textareaRef}
          backdropRef={c.backdropRef}
        />
        <FactoryComposerToolbar
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
      {c.showSkillDropdown ? (
        <SkillSlashDropdown
          candidates={c.skillCandidates}
          visible={c.showSkillDropdown}
          anchorEl={c.containerRef.current}
          onSelect={c.handleSkillSelect}
          onDismiss={c.handleDismiss}
          keyboardRef={c.mentionKeyboardRef}
        />
      ) : null}
    </footer>
  );
}

type ComposerControllerArgs = {
  canvasId: string;
  organizationId?: string;
  factoryId?: string;
  onSend: (content: string, images: AgentOutgoingImage[]) => Promise<void>;
  sendPending: boolean;
  nodes?: SuperplaneComponentsNode[];
  runs?: CanvasesCanvasRun[];
};

function useComposerController({
  canvasId,
  organizationId,
  factoryId,
  onSend,
  sendPending,
  nodes,
  runs,
}: ComposerControllerArgs) {
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const backdropRef = useRef<HTMLDivElement>(null);
  const mentionKeyboardRef = useRef<((e: React.KeyboardEvent) => boolean) | null>(null);
  const mentionsApi = useMentions();
  const { value, setValue, showDropdown, showSkillDropdown, filter, setCursorPos, getMarkdown, mentions, isEmpty } =
    mentionsApi;
  const { images, addFiles, removeImage, clear: clearImages } = useImageAttachments();

  const candidates = useMentionCandidates(nodes, runs, filter, showDropdown);
  const skillCandidates = useSkillSlashCandidates(organizationId, factoryId, filter, showSkillDropdown);
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

  useFlushAgentComposerSend(canvasId, onSend, sendPending);

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

  const handleSkillSelect = useCallback(
    (candidate: SkillSlashCandidate) => {
      const pos = mentionsApi.insertSkill(candidate.command);
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
    showSkillDropdown,
    candidates,
    skillCandidates,
    images,
    addFiles,
    removeImage,
    canSend,
    canAttach,
    handleSend,
    handlePaste,
    handleMentionSelect,
    handleSkillSelect,
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
