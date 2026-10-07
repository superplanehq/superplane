import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { useEffect, useRef, useState, type KeyboardEvent, type MouseEvent } from "react";

type ClickToRenameProps = {
  value: string;
  onSave: (next: string) => void;
  canEdit: boolean;
  busy?: boolean;
  testId: string;
  ariaLabel: string;
  className?: string;
  inputClassName?: string;
};

/**
 * Click (or Enter/Space) opens an input. Enter or blur saves a non-empty
 * name. Escape restores the previous value.
 *
 * The label keeps the layout size. The field paints around that label so
 * opening edit does not shift the header or the columns.
 */
export function ClickToRename({
  value,
  onSave,
  canEdit,
  busy = false,
  testId,
  ariaLabel,
  className,
  inputClassName,
}: ClickToRenameProps) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(value);
  const inputRef = useRef<HTMLInputElement>(null);
  const skipBlurCommitRef = useRef(false);

  useEffect(() => {
    if (!editing) {
      setDraft(value);
    }
  }, [editing, value]);

  useEffect(() => {
    if (!editing) {
      return;
    }
    const frame = window.requestAnimationFrame(() => {
      inputRef.current?.focus();
      inputRef.current?.select();
    });
    return () => window.cancelAnimationFrame(frame);
  }, [editing]);

  const startEditing = (event?: MouseEvent) => {
    if (!canEdit || busy) {
      return;
    }
    event?.preventDefault();
    skipBlurCommitRef.current = false;
    setDraft(value);
    setEditing(true);
  };

  const cancelEditing = () => {
    skipBlurCommitRef.current = true;
    setDraft(value);
    setEditing(false);
  };

  const commitDraft = () => {
    const next = draft.trim();
    if (!next) {
      inputRef.current?.focus();
      return;
    }
    if (next !== value.trim()) {
      onSave(next);
    }
    skipBlurCommitRef.current = true;
    setEditing(false);
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter") {
      event.preventDefault();
      commitDraft();
      return;
    }
    if (event.key === "Escape") {
      event.preventDefault();
      cancelEditing();
    }
  };

  return (
    <span className={cn("relative inline-flex max-w-full min-w-0 items-center overflow-visible", className)}>
      <span
        className={cn(
          "min-w-0 whitespace-nowrap",
          editing ? "invisible" : "truncate",
          canEdit && !editing
            ? "cursor-text rounded-sm hover:bg-muted/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            : undefined,
        )}
        title={canEdit && !editing ? "Click to rename" : undefined}
        data-testid={testId}
        tabIndex={canEdit && !editing ? 0 : undefined}
        onMouseDown={(event) => {
          if (canEdit && !editing) {
            event.preventDefault();
          }
        }}
        onClick={(event) => startEditing(event)}
        onKeyDown={(event) => {
          if (event.key === "Enter" || event.key === " ") {
            event.preventDefault();
            startEditing();
          }
        }}
      >
        {editing ? draft || "\u00a0" : value}
      </span>
      {editing ? (
        <Input
          ref={inputRef}
          value={draft}
          onChange={(event) => setDraft(event.target.value)}
          onBlur={() => {
            if (skipBlurCommitRef.current) {
              skipBlurCommitRef.current = false;
              return;
            }
            commitDraft();
          }}
          onKeyDown={handleKeyDown}
          disabled={busy}
          aria-label={ariaLabel}
          data-testid={`${testId}-input`}
          className={cn(
            "absolute top-1/2 left-1/2 z-10 h-[calc(100%+10px)] w-[calc(100%+18px)] -translate-x-1/2 -translate-y-1/2 rounded-md border border-border bg-background px-1.5 py-0 text-foreground shadow-none focus:border-foreground/40 focus:ring-1 focus:ring-ring",
            inputClassName,
          )}
        />
      ) : null}
    </span>
  );
}
