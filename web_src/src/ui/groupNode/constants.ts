export type GroupColor = "purple" | "blue" | "green" | "cyan" | "orange" | "rose" | "amber";

const GROUP_COLOR_KEYS: GroupColor[] = ["purple", "blue", "green", "cyan", "orange", "rose", "amber"];

/**
 * Minimum Y (relative to the group parent) for child nodes so they cannot overlap the
 * title/description header strip. Keep in sync with header layout in groupNode/index.tsx.
 */
export const GROUP_CHILD_MIN_Y_OFFSET = 104;

/** Inset from the group border on left, right, and bottom (flow px). Top inset is {@link GROUP_CHILD_MIN_Y_OFFSET}. */
export const GROUP_CHILD_EDGE_PADDING = 12;

/** Maps saved configuration values (including legacy `gray`) to a valid palette key. */
export function normalizeGroupColor(raw?: string): GroupColor {
  if (raw && GROUP_COLOR_KEYS.includes(raw as GroupColor)) {
    return raw as GroupColor;
  }
  return "purple";
}
