import { aiBuilderNodeDisplayName, type AiBuilderMentionNode } from "@/lib/aiBuilderNodeMentions";

export const AI_BUILDER_MENTION_LIST_MAX = 20;

/** `@` mention: from @ through cursor; no newlines or second @ in the segment. */
export function getActiveMentionSegment(value: string, cursor: number): { start: number; query: string } | null {
  const before = value.slice(0, cursor);
  const at = before.lastIndexOf("@");
  if (at < 0) {
    return null;
  }
  const afterAt = before.slice(at + 1);
  if (afterAt.includes("\n") || afterAt.includes("@")) {
    return null;
  }
  if (at > 0 && !/\s/.test(value.charAt(at - 1))) {
    return null;
  }
  return { start: at, query: afterAt };
}

/** `@Name` is finished (typed fully or picked); stop treating it as an active filter. */
export function isMentionSegmentComplete(query: string, nodes: AiBuilderMentionNode[]): boolean {
  const names = nodes
    .map((n) => aiBuilderNodeDisplayName(n).trim())
    .filter((name) => name.length > 0)
    .sort((a, b) => b.length - a.length);
  for (const name of names) {
    if (query === name || query.startsWith(`${name} `)) {
      return true;
    }
  }
  return false;
}

export function mentionQueryHasAnyMatch(query: string, nodes: AiBuilderMentionNode[]): boolean {
  const q = query.trim().toLowerCase();
  return nodes.some((n) => {
    const name = aiBuilderNodeDisplayName(n).toLowerCase();
    return !q || name.includes(q);
  });
}

export function filterMentionNodesByQuery(nodes: AiBuilderMentionNode[], query: string): AiBuilderMentionNode[] {
  const q = query.trim().toLowerCase();
  return nodes
    .filter((n) => {
      const name = aiBuilderNodeDisplayName(n).toLowerCase();
      return !q || name.includes(q);
    })
    .slice(0, AI_BUILDER_MENTION_LIST_MAX);
}

export type MentionMenuPlacement = {
  left: number;
  width: number;
  top: number;
  maxHeight: number;
};

/** Fixed menu below the anchor; height capped by viewport and list max (~max-h-48). */
export function computeMentionMenuPlacement(anchorRect: DOMRect): MentionMenuPlacement {
  const margin = 8;
  const maxListHeightPx = 192;
  const top = anchorRect.bottom + 4;
  const availableBelow = window.innerHeight - top - margin;
  const maxHeight = Math.min(maxListHeightPx, Math.max(0, availableBelow));
  let left = anchorRect.left;
  const width = anchorRect.width;
  const maxLeft = window.innerWidth - width - margin;
  if (left > maxLeft) {
    left = Math.max(margin, maxLeft);
  }
  return { left, width, top, maxHeight };
}
