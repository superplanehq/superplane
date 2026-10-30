/**
 * Lane colours for the line board. Vivid uses the full accent in light mode
 * and a dark surface mix. Dim uses a quieter wash in both light and dark mode.
 */

import {
  DEFAULT_LINE_BOARD_COLUMN_COLOR_VIEW,
  type LineBoardColumnColorView,
} from "../lib/lineBoardColumnColorViewPreference";

export type LineBoardColumnColorId = "lime" | "yellow" | "teal" | "sky" | "purple" | "slate";

export interface LineBoardColumnColor {
  id: LineBoardColumnColorId;
  /** Accessible name for the swatch button. */
  label: string;
  /** Picker swatch and vivid lane fill. */
  className: string;
  /** Dim lane fill. Quieter than the swatch in both themes. */
  laneClassName: string;
  /** Lane outline when the board view is Bordered. */
  borderClassName: string;
}

export const LINE_BOARD_COLUMN_COLORS: LineBoardColumnColor[] = [
  {
    id: "lime",
    label: "Mint",
    className: "line-board-color-lime line-board-column-vivid bg-[var(--board-column-vivid)]",
    laneClassName: "line-board-color-lime line-board-column-soft bg-[var(--board-column-soft)]",
    borderClassName: "line-board-color-lime line-board-column-bordered border-[var(--board-column-border)]",
  },
  {
    id: "yellow",
    label: "Peach",
    className: "line-board-color-yellow line-board-column-vivid bg-[var(--board-column-vivid)]",
    laneClassName: "line-board-color-yellow line-board-column-soft bg-[var(--board-column-soft)]",
    borderClassName: "line-board-color-yellow line-board-column-bordered border-[var(--board-column-border)]",
  },
  {
    id: "teal",
    label: "Teal",
    className: "bg-teal-300 dark:bg-teal-800",
    laneClassName: "bg-teal-100 dark:bg-teal-950/40",
    borderClassName: "border-teal-400 dark:border-teal-800/45",
  },
  {
    id: "sky",
    label: "Sky",
    className: "line-board-color-sky line-board-column-vivid bg-[var(--board-column-vivid)]",
    laneClassName: "line-board-color-sky line-board-column-soft bg-[var(--board-column-soft)]",
    borderClassName: "line-board-color-sky line-board-column-bordered border-[var(--board-column-border)]",
  },
  {
    id: "purple",
    label: "Purple",
    className: "bg-violet-300 dark:bg-violet-800",
    laneClassName: "bg-violet-100 dark:bg-violet-950/40",
    borderClassName: "border-violet-400 dark:border-violet-800/45",
  },
  {
    id: "slate",
    label: "Slate",
    className: "line-board-color-slate line-board-column-vivid bg-[var(--board-column-vivid)]",
    laneClassName: "line-board-color-slate line-board-column-soft bg-[var(--board-column-soft)]",
    borderClassName: "line-board-color-slate line-board-column-bordered border-[var(--board-column-border)]",
  },
];

export function lineBoardColumnColorById(id: string | null | undefined): LineBoardColumnColor | undefined {
  if (!id) {
    return undefined;
  }
  return LINE_BOARD_COLUMN_COLORS.find((color) => color.id === id);
}

export function lineBoardColumnLaneClassName(id: string | null | undefined): string | undefined {
  return lineBoardColumnColorById(id)?.laneClassName;
}

export type { LineBoardColumnColorView };

/**
 * Maps a stored color and the board view to lane chrome. Vivid and dim
 * replace the lane background. Bordered keeps the default fill and colors
 * the outline.
 */
export function lineBoardColumnLaneProps(
  id: string | null | undefined,
  view: LineBoardColumnColorView = DEFAULT_LINE_BOARD_COLUMN_COLOR_VIEW,
  options?: { mutedFallback?: boolean },
): { surfaceClassName?: string; className?: string } {
  const color = lineBoardColumnColorById(id);
  const muted = options?.mutedFallback ? "bg-muted" : undefined;

  if (!color || view === "off") {
    return muted ? { className: muted } : {};
  }
  if (view === "borders") {
    return { surfaceClassName: "bg-card", className: color.borderClassName };
  }
  if (view === "vivid") {
    return { surfaceClassName: color.className };
  }
  return { surfaceClassName: color.laneClassName };
}

/**
 * Board column colors are persisted on the line as a map of column key to
 * color id. Unknown ids (e.g. saved by a newer client) are dropped rather
 * than shown as an invalid color.
 */
export function normalizeColumnColors(
  columnColors: Record<string, string> | undefined,
): Record<string, LineBoardColumnColorId | null> {
  const normalized: Record<string, LineBoardColumnColorId | null> = {};
  for (const [key, value] of Object.entries(columnColors ?? {})) {
    if (lineBoardColumnColorById(value)) {
      normalized[key] = value as LineBoardColumnColorId;
    }
  }
  return normalized;
}

/**
 * Inverse of normalizeColumnColors: drops cleared (null) entries so the
 * persisted map only contains columns that have an explicit color.
 */
export function serializeColumnColors(
  columnColors: Record<string, LineBoardColumnColorId | null>,
): Record<string, string> {
  const serialized: Record<string, string> = {};
  for (const [key, value] of Object.entries(columnColors)) {
    if (value) {
      serialized[key] = value;
    }
  }
  return serialized;
}
