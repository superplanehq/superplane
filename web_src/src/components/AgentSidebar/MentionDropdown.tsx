import { createElement, useCallback, useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { cn, resolveIcon } from "@/lib/utils";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIconMaps";
import type { MentionItem } from "./useMentions";
import { BUILTIN_COMPONENT_ICON_SLUGS } from "./widgets/componentIcons";

export interface MentionCandidate {
  type: "node" | "run";
  id: string;
  label: string;
  /** Component name for nodes, status for runs */
  meta?: string;
  /** Is it a trigger node? */
  isTrigger?: boolean;
  /** Relative time string for runs */
  timeAgo?: string;
}

interface MentionDropdownProps {
  items: MentionCandidate[];
  visible: boolean;
  anchorEl: HTMLElement | null;
  onSelect: (item: MentionItem) => void;
  onDismiss: () => void;
}

function NodeIcon({ component, isTrigger }: { component?: string; isTrigger?: boolean }) {
  const iconSrc = component ? getHeaderIconSrc(component) : undefined;
  if (iconSrc) return <img src={iconSrc} alt="" className="size-4 object-contain shrink-0" />;

  const slug = component ? BUILTIN_COMPONENT_ICON_SLUGS[component] : undefined;
  if (slug) return createElement(resolveIcon(slug), { className: "size-4 shrink-0" });

  return <span className={cn("size-2.5 rounded-full shrink-0", isTrigger ? "bg-violet-500" : "bg-blue-500")} />;
}

function RunStatusDot({ status }: { status?: string }) {
  const color =
    status === "RESULT_PASSED"
      ? "bg-green-500"
      : status === "RESULT_FAILED"
        ? "bg-red-500"
        : status === "RESULT_CANCELLED"
          ? "bg-slate-400"
          : "bg-amber-500";
  return <span className={cn("size-2.5 rounded-full shrink-0", color)} />;
}

function NodeItemList({
  items,
  highlightedIndex,
  onMouseEnter,
  onMouseDown,
  showHeader,
}: {
  items: MentionCandidate[];
  highlightedIndex: number;
  onMouseEnter: (idx: number) => void;
  onMouseDown: (item: MentionCandidate) => void;
  showHeader: boolean;
}) {
  if (items.length === 0) return null;
  return (
    <>
      {showHeader && (
        <div className="px-3 py-1 text-[10px] font-medium uppercase tracking-wider text-slate-400 dark:text-gray-500">
          Nodes
        </div>
      )}
      {items.map((item, idx) => (
        <button
          key={`node-${item.id}`}
          type="button"
          data-index={idx}
          className={cn(
            "flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm transition-colors",
            idx === highlightedIndex ? "bg-slate-100 dark:bg-gray-700" : "hover:bg-slate-50 dark:hover:bg-gray-700",
          )}
          onMouseEnter={() => onMouseEnter(idx)}
          onMouseDown={(e) => {
            e.preventDefault();
            onMouseDown(item);
          }}
        >
          <NodeIcon component={item.meta} isTrigger={item.isTrigger} />
          <span className="flex-1 truncate font-medium text-slate-700 dark:text-gray-200">{item.label}</span>
          {item.meta && <span className="text-[10px] text-slate-400 dark:text-gray-500">{item.meta}</span>}
        </button>
      ))}
    </>
  );
}

function RunItemList({
  items,
  baseIndex,
  highlightedIndex,
  onMouseEnter,
  onMouseDown,
  showHeader,
}: {
  items: MentionCandidate[];
  baseIndex: number;
  highlightedIndex: number;
  onMouseEnter: (idx: number) => void;
  onMouseDown: (item: MentionCandidate) => void;
  showHeader: boolean;
}) {
  if (items.length === 0) return null;
  return (
    <>
      {showHeader && (
        <div className="mt-1 border-t border-slate-100 px-3 py-1 pt-2 text-[10px] font-medium uppercase tracking-wider text-slate-400 dark:border-gray-700 dark:text-gray-500">
          Recent Runs
        </div>
      )}
      {items.map((item, runIndex) => {
        const idx = baseIndex + runIndex;
        return (
          <button
            key={`run-${item.id}`}
            type="button"
            data-index={idx}
            className={cn(
              "flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm transition-colors",
              idx === highlightedIndex ? "bg-slate-100 dark:bg-gray-700" : "hover:bg-slate-50 dark:hover:bg-gray-700",
            )}
            onMouseEnter={() => onMouseEnter(idx)}
            onMouseDown={(e) => {
              e.preventDefault();
              onMouseDown(item);
            }}
          >
            <RunStatusDot status={item.meta} />
            <span className="flex-1 truncate font-medium text-slate-700 dark:text-gray-200">{item.label}</span>
            {item.timeAgo && <span className="text-[10px] text-slate-400 dark:text-gray-500">{item.timeAgo}</span>}
          </button>
        );
      })}
    </>
  );
}

export function MentionDropdown({
  items,
  visible,
  anchorEl,
  onSelect,
  onDismiss,
  keyboardRef,
}: MentionDropdownProps & { keyboardRef?: React.MutableRefObject<((e: React.KeyboardEvent) => boolean) | null> }) {
  const [highlightedIndex, setHighlightedIndex] = useState(0);
  const listRef = useRef<HTMLDivElement>(null);

  // Reset highlight when items change
  useEffect(() => {
    setHighlightedIndex(0);
  }, [items]);

  // Expose keyboard handler via ref so textarea can call it directly
  useEffect(() => {
    if (!keyboardRef) return;
    if (!visible || items.length === 0) {
      keyboardRef.current = null;
      return;
    }
    const handleKeyboard = (e: React.KeyboardEvent): boolean => {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setHighlightedIndex((i) => (i < items.length - 1 ? i + 1 : 0));
        return true;
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        setHighlightedIndex((i) => (i > 0 ? i - 1 : items.length - 1));
        return true;
      } else if (e.key === "Enter" || e.key === "Tab") {
        e.preventDefault();
        if (items[highlightedIndex]) {
          const item = items[highlightedIndex];
          onSelect({ type: item.type, id: item.id, label: item.label, meta: item.meta });
        }
        return true;
      } else if (e.key === "Escape") {
        e.preventDefault();
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
  }, [visible, items, highlightedIndex, onSelect, onDismiss, keyboardRef]);

  // Scroll highlighted item into view
  useEffect(() => {
    if (!listRef.current) return;
    const el = listRef.current.querySelector(`[data-index="${highlightedIndex}"]`);
    el?.scrollIntoView({ block: "nearest" });
  }, [highlightedIndex]);

  const handleClick = useCallback(
    (item: MentionCandidate) => {
      onSelect({ type: item.type, id: item.id, label: item.label, meta: item.meta });
    },
    [onSelect],
  );

  if (!visible || items.length === 0 || !anchorEl) return null;

  const nodes = items.filter((i) => i.type === "node");
  const runs = items.filter((i) => i.type === "run");

  // Position above the textarea
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
        <NodeItemList
          items={nodes}
          highlightedIndex={highlightedIndex}
          onMouseEnter={setHighlightedIndex}
          onMouseDown={handleClick}
          showHeader={runs.length > 0}
        />
        <RunItemList
          items={runs}
          baseIndex={nodes.length}
          highlightedIndex={highlightedIndex}
          onMouseEnter={setHighlightedIndex}
          onMouseDown={handleClick}
          showHeader={nodes.length > 0}
        />
      </div>
    </div>,
    document.body,
  );
}
