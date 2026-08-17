import type { WorkOrderReactionGroup } from "../../workOrderReactionTypes";

/** Storybook-only display name for the signed-in demo user. Matches `STORYBOOK_ME_USER_NAME`. */
export const CURRENT_REACTOR_NAME = "Storybook User";

/**
 * Seed reactions keyed by work order id — not sent to any API. Covers the
 * empty, single-reactor, and multi-emoji/multi-reactor states across the
 * Open/Running/Closed demo work orders.
 */
export const SEEDED_WORK_ORDER_REACTIONS: Record<string, WorkOrderReactionGroup[]> = {
  "wo-open-refunds": [
    { emoji: "👍", reactorNames: ["Alex Reviewer", "Jamie Operator", CURRENT_REACTOR_NAME] },
    { emoji: "🎉", reactorNames: ["Alex Reviewer"] },
  ],
  "wo-running-refunds": [{ emoji: "👀", reactorNames: ["Alex Reviewer", "Jamie Operator"] }],
  "wo-closed-refunds": [
    { emoji: "✅", reactorNames: ["Alex Reviewer", "Jamie Operator", CURRENT_REACTOR_NAME, "Priya Kapoor"] },
  ],
};
