import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { useEffect, useRef, useState, type KeyboardEvent, type MouseEvent } from "react";

type FactoryAppCanvasTitleEditorProps = {
  title: string;
  renameEnabled: boolean;
  configureBusy: boolean;
  onDraftTitleChange: (name: string) => void;
};

export function FactoryAppCanvasTitleEditor({
  title,
  renameEnabled,
  configureBusy,
  onDraftTitleChange,
}: FactoryAppCanvasTitleEditorProps) {
  const [editing, setEditing] = useState(false);
  const [draftName, setDraftName] = useState(title);
  const inputRef = useRef<HTMLInputElement>(null);
  const skipBlurCommitRef = useRef(false);

  useEffect(() => {
    if (!editing) {
      setDraftName(title);
    }
  }, [editing, title]);

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
    if (!renameEnabled || configureBusy) {
      return;
    }
    event?.preventDefault();
    skipBlurCommitRef.current = false;
    setDraftName(title);
    setEditing(true);
  };

  const cancelEditing = () => {
    skipBlurCommitRef.current = true;
    setDraftName(title);
    setEditing(false);
  };

  const commitDraft = () => {
    const nextName = draftName.trim();
    if (!nextName) {
      inputRef.current?.focus();
      return;
    }
    if (nextName !== title.trim()) {
      onDraftTitleChange(nextName);
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

  if (editing) {
    return (
      <Input
        ref={inputRef}
        value={draftName}
        onChange={(event) => setDraftName(event.target.value)}
        onBlur={() => {
          if (skipBlurCommitRef.current) {
            skipBlurCommitRef.current = false;
            return;
          }
          commitDraft();
        }}
        onKeyDown={handleKeyDown}
        disabled={configureBusy}
        size={Math.max(draftName.length + 1, 8)}
        aria-label="Automation name"
        data-testid="factory-app-rename-input"
        className="h-6 w-auto max-w-[16rem] px-1.5 py-0 text-[15px] font-semibold tracking-[-0.01em] leading-none"
      />
    );
  }

  return (
    <h2
      className={cn(
        "text-[15px] font-semibold tracking-[-0.01em] text-foreground",
        renameEnabled
          ? "cursor-text rounded-sm border border-transparent px-1 -mx-1 hover:border-border hover:bg-muted/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          : undefined,
      )}
      title={renameEnabled ? "Click to rename" : undefined}
      data-testid="factory-app-canvas-title"
      tabIndex={renameEnabled ? 0 : undefined}
      onMouseDown={(event) => {
        if (renameEnabled) {
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
      {title}
    </h2>
  );
}
