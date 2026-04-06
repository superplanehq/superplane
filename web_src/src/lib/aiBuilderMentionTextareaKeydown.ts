import type { AiBuilderMentionNode } from "@/lib/aiBuilderNodeMentions";
import type { Dispatch, KeyboardEvent as ReactKeyboardEvent, SetStateAction } from "react";

export type MentionTextareaKeydownCtx = {
  mentionOpen: boolean;
  filteredMentionNodes: AiBuilderMentionNode[];
  mentionSelectedIndex: number;
  setMentionSelectedIndex: Dispatch<SetStateAction<number>>;
  applyMention: (node: AiBuilderMentionNode) => void;
  setMentionOpen: (open: boolean) => void;
  onSendPrompt: () => void;
};

export function handleMentionTextareaKeyDown(
  e: ReactKeyboardEvent<HTMLTextAreaElement>,
  ctx: MentionTextareaKeydownCtx,
): void {
  const {
    mentionOpen,
    filteredMentionNodes,
    mentionSelectedIndex,
    setMentionSelectedIndex,
    applyMention,
    setMentionOpen,
    onSendPrompt,
  } = ctx;

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
}
