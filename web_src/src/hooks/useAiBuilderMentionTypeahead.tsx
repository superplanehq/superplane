import { aiBuilderNodeDisplayName, type AiBuilderMentionNode } from "@/lib/aiBuilderNodeMentions";
import {
  computeMentionMenuPlacement,
  filterMentionNodesByQuery,
  getActiveMentionSegment,
  isMentionSegmentComplete,
  mentionQueryHasAnyMatch,
  type MentionMenuPlacement,
} from "@/lib/aiBuilderMentionTypeahead";
import { cn } from "@/lib/utils";
import type { KeyboardEvent as ReactKeyboardEvent, RefObject } from "react";
import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";

export type AiBuilderMentionListPortalProps = {
  open: boolean;
  placement: MentionMenuPlacement | null;
  nodes: AiBuilderMentionNode[];
  selectedIndex: number;
  onHoverIndex: (index: number) => void;
  onPick: (node: AiBuilderMentionNode) => void;
};

export function AiBuilderMentionListPortal({
  open,
  placement,
  nodes,
  selectedIndex,
  onHoverIndex,
  onPick,
}: AiBuilderMentionListPortalProps) {
  if (!open || !placement || nodes.length === 0) {
    return null;
  }

  return createPortal(
    <div
      className="fixed z-[300] overflow-y-auto rounded-md border border-slate-200 bg-popover py-1 shadow-lg"
      role="listbox"
      style={{
        left: placement.left,
        width: placement.width,
        bottom: placement.bottom,
        maxHeight: placement.maxHeight,
      }}
    >
      {nodes.map((node, index) => {
        const label = aiBuilderNodeDisplayName(node);
        return (
          <button
            key={node.id}
            type="button"
            role="option"
            aria-selected={index === selectedIndex}
            className={cn(
              "block w-full px-2 py-1.5 text-left text-sm text-slate-800",
              index === selectedIndex ? "bg-slate-100" : "hover:bg-slate-50",
            )}
            onMouseDown={(ev) => ev.preventDefault()}
            onMouseEnter={() => onHoverIndex(index)}
            onClick={() => onPick(node)}
          >
            {label}
          </button>
        );
      })}
    </div>,
    document.body,
  );
}

type UseAiBuilderMentionTypeaheadArgs = {
  aiInput: string;
  aiInputRef: RefObject<HTMLTextAreaElement | null>;
  canvasNodes?: AiBuilderMentionNode[];
  onAiInputChange: (value: string) => void;
  onSendPrompt: () => void;
};

export function useAiBuilderMentionTypeahead({
  aiInput,
  aiInputRef,
  canvasNodes,
  onAiInputChange,
  onSendPrompt,
}: UseAiBuilderMentionTypeaheadArgs) {
  const mentionAnchorRef = useRef<HTMLDivElement>(null);
  const [mentionMenuPlacement, setMentionMenuPlacement] = useState<MentionMenuPlacement | null>(null);
  const [mentionOpen, setMentionOpen] = useState(false);
  const [mentionStart, setMentionStart] = useState(0);
  const [mentionQuery, setMentionQuery] = useState("");
  const [mentionSelectedIndex, setMentionSelectedIndex] = useState(0);

  const syncMentionUi = useCallback(
    (value: string, cursor: number) => {
      if (!canvasNodes?.length) {
        setMentionOpen(false);
        return;
      }
      const seg = getActiveMentionSegment(value, cursor);
      if (!seg) {
        setMentionOpen(false);
        return;
      }
      if (isMentionSegmentComplete(seg.query, canvasNodes)) {
        setMentionOpen(false);
        return;
      }
      if (!mentionQueryHasAnyMatch(seg.query, canvasNodes)) {
        setMentionOpen(false);
        return;
      }
      setMentionOpen(true);
      setMentionStart(seg.start);
      setMentionQuery(seg.query);
      setMentionSelectedIndex(0);
    },
    [canvasNodes],
  );

  const filteredMentionNodes = useMemo(
    () => (canvasNodes?.length ? filterMentionNodesByQuery(canvasNodes, mentionQuery) : []),
    [canvasNodes, mentionQuery],
  );

  const applyMention = useCallback(
    (node: AiBuilderMentionNode) => {
      const el = aiInputRef.current;
      if (!el) {
        return;
      }
      const cursor = el.selectionStart ?? aiInput.length;
      const name = aiBuilderNodeDisplayName(node);
      const before = aiInput.slice(0, mentionStart);
      const after = aiInput.slice(cursor);
      const insert = `@${name} `;
      const next = before + insert + after;
      onAiInputChange(next);
      setMentionOpen(false);
      const pos = before.length + insert.length;
      requestAnimationFrame(() => {
        el.focus();
        el.setSelectionRange(pos, pos);
        syncMentionUi(next, pos);
      });
    },
    [aiInput, aiInputRef, mentionStart, onAiInputChange, syncMentionUi],
  );

  const updateMentionMenuPlacement = useCallback(() => {
    if (!mentionOpen || filteredMentionNodes.length === 0) {
      setMentionMenuPlacement(null);
      return;
    }
    const anchor = mentionAnchorRef.current;
    if (!anchor) {
      setMentionMenuPlacement(null);
      return;
    }
    setMentionMenuPlacement(computeMentionMenuPlacement(anchor.getBoundingClientRect()));
  }, [mentionOpen, filteredMentionNodes.length]);

  useLayoutEffect(() => {
    updateMentionMenuPlacement();
  }, [updateMentionMenuPlacement, aiInput, mentionOpen, filteredMentionNodes.length]);

  useEffect(() => {
    if (!mentionOpen || filteredMentionNodes.length === 0) {
      return;
    }
    const onReposition = () => updateMentionMenuPlacement();
    window.addEventListener("scroll", onReposition, true);
    window.addEventListener("resize", onReposition);
    return () => {
      window.removeEventListener("scroll", onReposition, true);
      window.removeEventListener("resize", onReposition);
    };
  }, [mentionOpen, filteredMentionNodes.length, updateMentionMenuPlacement]);

  const handleTextareaChange = useCallback(
    (v: string, selectionStart: number) => {
      onAiInputChange(v);
      requestAnimationFrame(() => syncMentionUi(v, selectionStart));
    },
    [onAiInputChange, syncMentionUi],
  );

  const handleTextareaKeyDown = useCallback(
    (e: ReactKeyboardEvent<HTMLTextAreaElement>) => {
      if (mentionOpen && filteredMentionNodes.length > 0) {
        if (e.key === "ArrowDown") {
          e.preventDefault();
          setMentionSelectedIndex((i) => Math.min(i + 1, filteredMentionNodes.length - 1));
          return;
        }
        if (e.key === "ArrowUp") {
          e.preventDefault();
          setMentionSelectedIndex((i) => Math.max(i - 1, 0));
          return;
        }
        if (e.key === "Enter" && !e.shiftKey) {
          e.preventDefault();
          const pick = filteredMentionNodes[mentionSelectedIndex];
          if (pick) {
            applyMention(pick);
          }
          return;
        }
        if (e.key === "Escape") {
          e.preventDefault();
          setMentionOpen(false);
          return;
        }
      }

      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        onSendPrompt();
      }
    },
    [applyMention, filteredMentionNodes, mentionOpen, mentionSelectedIndex, onSendPrompt],
  );

  return {
    mentionAnchorRef,
    mentionOpen,
    mentionMenuPlacement,
    filteredMentionNodes,
    mentionSelectedIndex,
    setMentionSelectedIndex,
    syncMentionUi,
    applyMention,
    handleTextareaChange,
    handleTextareaKeyDown,
  };
}
