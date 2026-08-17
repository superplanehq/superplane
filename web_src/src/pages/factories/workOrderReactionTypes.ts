/** One emoji group on a work order: the emoji and everyone who reacted with it. */
export interface WorkOrderReactionGroup {
  emoji: string;
  reactorNames: string[];
}

/** Curated set for the picker — a full emoji library is out of scope for the prototype. */
export const REACTION_EMOJI_OPTIONS = ["👍", "🎉", "❤️", "👀", "🚀", "✅"] as const;
