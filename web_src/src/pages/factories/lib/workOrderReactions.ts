/**
 * Fixed, GitHub-style reaction vocabulary for work orders. Reusing this
 * exact set (rather than a free-form emoji picker) keeps scope small and
 * matches the reaction values already supported elsewhere in the product
 * (see pkg/integrations/github/components/pulls/add_reaction.go and
 * web_src/src/pages/app/mappers/github/add_reaction.ts, which map the same
 * content values for canvas-automation "add reaction" nodes).
 */
export const REACTION_CONTENTS = ["+1", "-1", "laugh", "hooray", "confused", "heart", "rocket", "eyes"] as const;

export type ReactionContent = (typeof REACTION_CONTENTS)[number];

const REACTION_EMOJI: Record<ReactionContent, string> = {
  "+1": "👍",
  "-1": "👎",
  laugh: "😄",
  hooray: "🎉",
  confused: "😕",
  heart: "❤️",
  rocket: "🚀",
  eyes: "👀",
};

const REACTION_LABEL: Record<ReactionContent, string> = {
  "+1": "Thumbs up",
  "-1": "Thumbs down",
  laugh: "Laugh",
  hooray: "Hooray",
  confused: "Confused",
  heart: "Heart",
  rocket: "Rocket",
  eyes: "Eyes",
};

export function isReactionContent(content: string | undefined): content is ReactionContent {
  return Boolean(content) && (REACTION_CONTENTS as readonly string[]).includes(content as string);
}

export function reactionEmoji(content?: string): string {
  if (isReactionContent(content)) {
    return REACTION_EMOJI[content];
  }
  return content || "❓";
}

export function reactionLabel(content?: string): string {
  if (isReactionContent(content)) {
    return REACTION_LABEL[content];
  }
  return content || "Reaction";
}
