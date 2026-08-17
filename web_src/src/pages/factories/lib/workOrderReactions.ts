/** Curated reaction set — a short list keeps the picker compact and avoids a full emoji keyboard. */
export const WORK_ORDER_REACTION_EMOJIS = ["👍", "🎉", "❤️", "👀", "🚀", "😕"] as const;

export interface WorkOrderReaction {
  emoji: string;
  count: number;
  /** Whether the current viewer is one of the reactors. */
  mine: boolean;
  /** Display names of everyone who reacted with this emoji, for the "seen by" tooltip. */
  reactorNames?: string[];
}

const EMOJI_SLUGS: Record<string, string> = {
  "👍": "thumbs-up",
  "🎉": "party",
  "❤️": "heart",
  "👀": "eyes",
  "🚀": "rocket",
  "😕": "confused",
};

export function emojiSlug(emoji: string): string {
  return EMOJI_SLUGS[emoji] ?? emoji;
}

/**
 * Prototype-only local toggle: adds/removes "You" from the given emoji's
 * reactors. No network call — stories use this to simulate optimistic add
 * or remove without a backend.
 */
export function toggleWorkOrderReaction(reactions: WorkOrderReaction[], emoji: string): WorkOrderReaction[] {
  const existing = reactions.find((reaction) => reaction.emoji === emoji);

  if (!existing) {
    return [...reactions, { emoji, count: 1, mine: true, reactorNames: ["You"] }];
  }

  if (existing.mine) {
    const nextCount = existing.count - 1;
    if (nextCount <= 0) {
      return reactions.filter((reaction) => reaction.emoji !== emoji);
    }
    return reactions.map((reaction) =>
      reaction.emoji === emoji
        ? {
            ...reaction,
            count: nextCount,
            mine: false,
            reactorNames: (reaction.reactorNames ?? []).filter((n) => n !== "You"),
          }
        : reaction,
    );
  }

  return reactions.map((reaction) =>
    reaction.emoji === emoji
      ? { ...reaction, count: reaction.count + 1, mine: true, reactorNames: [...(reaction.reactorNames ?? []), "You"] }
      : reaction,
  );
}
