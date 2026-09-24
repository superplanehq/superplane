import { createPortal } from "react-dom";
import { useCallback, useEffect, useRef, useState } from "react";

import type { SkillSlashCandidate } from "@/lib/skillSlash";
import { SKILL_SLASH_MENU_TEST_ID, type SkillSlashMenuPortal } from "@/lib/skillSlashMenu";
import { cn } from "@/lib/utils";

export function SkillSlashMenu({
  candidates,
  highlightIndex,
  onHighlight,
  onSelect,
  portal,
}: {
  candidates: SkillSlashCandidate[];
  highlightIndex: number;
  onHighlight: (index: number) => void;
  onSelect: (candidate: SkillSlashCandidate) => void;
  portal?: SkillSlashMenuPortal;
}) {
  if (candidates.length === 0) {
    return null;
  }

  const list = (
    <ul
      role="listbox"
      aria-label="Insert a skill command"
      data-testid={SKILL_SLASH_MENU_TEST_ID}
      className={cn(
        "max-h-56 overflow-y-auto rounded-lg border border-border bg-background py-1 shadow-md",
        portal
          ? portal.root === document.body
            ? "fixed z-[70]"
            : "absolute z-30"
          : "absolute inset-x-0 bottom-full z-20 mb-1",
      )}
      style={
        portal ? { left: portal.left, width: portal.width, top: portal.top, maxHeight: portal.maxHeight } : undefined
      }
    >
      {candidates.map((candidate, index) => {
        const highlighted = index === highlightIndex;
        return (
          <li key={candidate.id} role="option" aria-selected={highlighted}>
            <button
              type="button"
              data-testid={`skill-slash-option-${candidate.command}`}
              className={cn(
                "flex w-full flex-col px-2 py-1.5 text-left",
                highlighted ? "bg-accent" : "hover:bg-accent/60",
              )}
              onMouseEnter={() => onHighlight(index)}
              onMouseDown={(event) => {
                event.preventDefault();
                onSelect(candidate);
              }}
            >
              <span className="truncate text-[13px] font-medium text-foreground">/{candidate.command}</span>
              {candidate.description ? (
                <span className="truncate text-[11px] text-muted-foreground">{candidate.description}</span>
              ) : candidate.title && candidate.title !== candidate.command ? (
                <span className="truncate text-[11px] text-muted-foreground">{candidate.title}</span>
              ) : null}
            </button>
          </li>
        );
      })}
    </ul>
  );

  return portal ? createPortal(list, portal.root) : list;
}

export function SkillSlashDropdown({
  candidates,
  visible,
  anchorEl,
  onSelect,
  onDismiss,
  keyboardRef,
}: {
  candidates: SkillSlashCandidate[];
  visible: boolean;
  anchorEl: HTMLElement | null;
  onSelect: (candidate: SkillSlashCandidate) => void;
  onDismiss: () => void;
  keyboardRef?: React.MutableRefObject<((event: React.KeyboardEvent) => boolean) | null>;
}) {
  const [highlightedIndex, setHighlightedIndex] = useState(0);
  const listRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    setHighlightedIndex(0);
  }, [candidates]);

  useEffect(() => {
    if (!keyboardRef) {
      return;
    }
    if (!visible || candidates.length === 0) {
      keyboardRef.current = null;
      return;
    }
    const handleKeyboard = (event: React.KeyboardEvent): boolean => {
      if (event.key === "ArrowDown") {
        event.preventDefault();
        setHighlightedIndex((index) => (index < candidates.length - 1 ? index + 1 : 0));
        return true;
      }
      if (event.key === "ArrowUp") {
        event.preventDefault();
        setHighlightedIndex((index) => (index > 0 ? index - 1 : candidates.length - 1));
        return true;
      }
      if (event.key === "Enter" || event.key === "Tab") {
        event.preventDefault();
        const selected = candidates[highlightedIndex];
        if (selected) {
          onSelect(selected);
        }
        return true;
      }
      if (event.key === "Escape") {
        event.preventDefault();
        onDismiss();
        return true;
      }
      return false;
    };
    keyboardRef.current = handleKeyboard;
    return () => {
      if (keyboardRef.current === handleKeyboard) {
        keyboardRef.current = null;
      }
    };
  }, [visible, candidates, highlightedIndex, onSelect, onDismiss, keyboardRef]);

  useEffect(() => {
    if (!listRef.current) {
      return;
    }
    listRef.current.querySelector(`[data-index="${highlightedIndex}"]`)?.scrollIntoView({ block: "nearest" });
  }, [highlightedIndex]);

  const handleClick = useCallback(
    (candidate: SkillSlashCandidate) => {
      onSelect(candidate);
    },
    [onSelect],
  );

  if (!visible || candidates.length === 0 || !anchorEl) {
    return null;
  }

  const rect = anchorEl.getBoundingClientRect();
  const style: React.CSSProperties = {
    position: "fixed",
    bottom: window.innerHeight - rect.top + 4,
    left: rect.left,
    width: Math.min(320, rect.width),
    zIndex: 50,
  };

  return createPortal(
    <div
      style={style}
      className="overflow-hidden rounded-lg border border-slate-200 bg-white shadow-lg dark:border-gray-700 dark:bg-gray-800"
    >
      <div ref={listRef} className="max-h-64 overflow-y-auto py-1">
        {candidates.map((candidate, index) => (
          <button
            key={candidate.id}
            type="button"
            data-index={index}
            data-testid={`skill-slash-option-${candidate.command}`}
            className={cn(
              "flex w-full flex-col px-3 py-1.5 text-left text-sm transition-colors",
              index === highlightedIndex ? "bg-slate-100 dark:bg-gray-700" : "hover:bg-slate-50 dark:hover:bg-gray-700",
            )}
            onMouseEnter={() => setHighlightedIndex(index)}
            onMouseDown={(event) => {
              event.preventDefault();
              handleClick(candidate);
            }}
          >
            <span className="truncate font-medium text-slate-700 dark:text-gray-200">/{candidate.command}</span>
            {candidate.description ? (
              <span className="truncate text-[11px] text-slate-400 dark:text-gray-500">{candidate.description}</span>
            ) : null}
          </button>
        ))}
      </div>
    </div>,
    document.body,
  );
}
