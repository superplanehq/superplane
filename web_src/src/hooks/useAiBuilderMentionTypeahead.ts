import { aiBuilderNodeDisplayName, type AiBuilderMentionNode } from "@/lib/aiBuilderNodeMentions";
import {
  filterMentionNodesByQuery,
  getActiveMentionSegment,
  isMentionSegmentComplete,
  mentionQueryHasAnyMatch,
} from "@/lib/aiBuilderMentionTypeahead";
import { handleMentionTextareaKeyDown as dispatchMentionTextareaKeyDown } from "@/lib/aiBuilderMentionTextareaKeydown";
import { useMentionMenuPlacement } from "@/hooks/useMentionMenuPlacement";
import type { KeyboardEvent as ReactKeyboardEvent, RefObject } from "react";
import { useCallback, useMemo, useRef, useState } from "react";

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

  const mentionMenuPlacement = useMentionMenuPlacement(
    mentionOpen,
    filteredMentionNodes.length,
    mentionAnchorRef,
    aiInput,
  );

  const handleTextareaChange = useCallback(
    (v: string, selectionStart: number) => {
      onAiInputChange(v);
      requestAnimationFrame(() => syncMentionUi(v, selectionStart));
    },
    [onAiInputChange, syncMentionUi],
  );

  const handleTextareaKeyDown = useCallback(
    (e: ReactKeyboardEvent<HTMLTextAreaElement>) => {
      dispatchMentionTextareaKeyDown(e, {
        mentionOpen,
        filteredMentionNodes,
        mentionSelectedIndex,
        setMentionSelectedIndex,
        applyMention,
        setMentionOpen,
        onSendPrompt,
      });
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
