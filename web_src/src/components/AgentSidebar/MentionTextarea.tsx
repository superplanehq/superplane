import { useCallback } from "react";
import { cn } from "@/lib/utils";
import { BackdropContent } from "./BackdropContent";
import type { InsertedMention } from "./useMentions";

interface MentionTextareaProps {
  value: string;
  mentions: InsertedMention[];
  setValue: (v: string) => void;
  setCursorPos: (pos: number) => void;
  onKeyDown: (e: React.KeyboardEvent<HTMLTextAreaElement>) => void;
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
  placeholder,
  textareaRef,
  backdropRef,
}: MentionTextareaProps) {
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

  const handleScroll = useCallback(() => {
    if (textareaRef.current && backdropRef.current) {
      backdropRef.current.scrollTop = textareaRef.current.scrollTop;
      backdropRef.current.scrollLeft = textareaRef.current.scrollLeft;
    }
  }, [textareaRef, backdropRef]);

  return (
    <div className="relative">
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
      <textarea
        ref={textareaRef}
        value={value}
        onChange={handleChange}
        onSelect={handleSelect}
        onKeyUp={handleSelect}
        onClick={handleSelect}
        onScroll={handleScroll}
        rows={1}
        placeholder={placeholder}
        data-testid="agent-input"
        className={cn(
          "relative min-h-9 w-full resize-none border-0 bg-transparent px-3 py-2.5 text-sm shadow-none",
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
