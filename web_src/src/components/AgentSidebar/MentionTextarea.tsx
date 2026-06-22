import { useCallback, useLayoutEffect } from "react";
import { cn } from "@/lib/utils";
import { BackdropContent } from "./BackdropContent";
import type { InsertedMention } from "./useMentions";

const composerTextMetrics = "px-3 py-2.5 text-sm leading-5 font-normal tracking-normal";
const maxTextareaHeight = 144;
const maxScrollRatio = 0.96;

interface MentionTextareaProps {
  value: string;
  mentions: InsertedMention[];
  setValue: (v: string) => void;
  setCursorPos: (pos: number) => void;
  onKeyDown: (e: React.KeyboardEvent<HTMLTextAreaElement>) => void;
  onPaste?: (e: React.ClipboardEvent<HTMLTextAreaElement>) => void;
  placeholder?: string;
  textareaRef: React.RefObject<HTMLTextAreaElement | null>;
  backdropRef: React.RefObject<HTMLDivElement | null>;
}

export function MentionTextarea({
  value,
  mentions,
  setValue,
  setCursorPos,
  onKeyDown,
  onPaste,
  placeholder,
  textareaRef,
  backdropRef,
}: MentionTextareaProps) {
  const syncBackdropScroll = useCallback(
    (textarea: HTMLTextAreaElement) => {
      if (!backdropRef.current) {
        return;
      }

      const scrollTop = clampScrollTop(textarea);
      if (textarea.scrollTop !== scrollTop) {
        textarea.scrollTop = scrollTop;
      }

      backdropRef.current.scrollTop = scrollTop;
      backdropRef.current.scrollLeft = textarea.scrollLeft;
    },
    [backdropRef],
  );

  const syncTextareaLayout = useCallback(
    (textarea: HTMLTextAreaElement) => {
      textarea.style.height = "auto";
      const nextHeight = Math.min(textarea.scrollHeight, maxTextareaHeight);

      if (nextHeight > 0) {
        textarea.style.height = `${nextHeight}px`;
      }

      textarea.style.overflowY = textarea.scrollHeight > maxTextareaHeight ? "auto" : "hidden";
      syncBackdropScroll(textarea);
    },
    [syncBackdropScroll],
  );

  useLayoutEffect(() => {
    if (!textareaRef.current) {
      return;
    }

    syncTextareaLayout(textareaRef.current);
  }, [syncTextareaLayout, textareaRef, value]);

  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLTextAreaElement>) => {
      setValue(e.target.value);
      setCursorPos(e.target.selectionStart ?? e.target.value.length);
      syncTextareaLayout(e.target);
    },
    [setValue, setCursorPos, syncTextareaLayout],
  );

  const handleSelect = useCallback(
    (e: React.SyntheticEvent<HTMLTextAreaElement>) => {
      setCursorPos((e.target as HTMLTextAreaElement).selectionStart ?? 0);
    },
    [setCursorPos],
  );

  const handleScroll = useCallback(() => {
    if (textareaRef.current) {
      syncBackdropScroll(textareaRef.current);
    }
  }, [textareaRef, syncBackdropScroll]);

  return (
    <div className="relative">
      <div
        ref={backdropRef}
        aria-hidden="true"
        className={cn(
          "pointer-events-none absolute inset-0 whitespace-pre-wrap break-words overflow-hidden",
          composerTextMetrics,
        )}
      >
        <BackdropContent text={value} mentions={mentions} />
      </div>
      <textarea
        ref={textareaRef}
        value={value}
        onChange={handleChange}
        onSelect={handleSelect}
        onKeyUp={handleSelect}
        onClick={handleSelect}
        onScroll={handleScroll}
        onPaste={onPaste}
        rows={1}
        placeholder={placeholder}
        data-testid="agent-input"
        className={cn(
          "relative min-h-9 w-full resize-none border-0 bg-transparent shadow-none",
          composerTextMetrics,
          "outline-none ring-0 focus-visible:border-0 focus-visible:ring-0 focus-visible:outline-none",
          "placeholder:text-muted-foreground disabled:cursor-not-allowed disabled:opacity-50",
          "text-transparent caret-slate-900 selection:bg-blue-200/50",
          "dark:bg-transparent",
        )}
        onKeyDown={onKeyDown}
      />
    </div>
  );
}

function clampScrollTop(textarea: HTMLTextAreaElement): number {
  const maxScrollTop = Math.max(0, textarea.scrollHeight - textarea.clientHeight);
  const maxAllowedScrollTop = maxScrollTop * maxScrollRatio;

  return Math.min(textarea.scrollTop, maxAllowedScrollTop);
}
