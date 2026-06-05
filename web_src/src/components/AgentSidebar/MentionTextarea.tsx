import { cn } from "@/lib/utils";
import { BackdropContent } from "./BackdropContent";
import type { InsertedMention } from "./useMentions";

interface MentionTextareaProps {
  value: string;
  mentions: InsertedMention[];
  onChange: (e: React.ChangeEvent<HTMLTextAreaElement>) => void;
  onSelect: (e: React.SyntheticEvent<HTMLTextAreaElement>) => void;
  onKeyDown: (e: React.KeyboardEvent<HTMLTextAreaElement>) => void;
  onScroll: () => void;
  placeholder?: string;
  textareaRef: React.RefObject<HTMLTextAreaElement | null>;
  backdropRef: React.RefObject<HTMLDivElement | null>;
}

export function MentionTextarea({
  value,
  mentions,
  onChange,
  onSelect,
  onKeyDown,
  onScroll,
  placeholder,
  textareaRef,
  backdropRef,
}: MentionTextareaProps) {
  return (
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
        onChange={onChange}
        onSelect={onSelect}
        onKeyUp={onSelect}
        onClick={onSelect}
        onScroll={onScroll}
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
