/**
 * Lane colours for the line board. Light mode keeps a pastel fill. Dark mode
 * uses a dim wash so the board stays quiet.
 */

export type LineBoardColumnColorId = "lime" | "yellow" | "teal" | "sky" | "purple" | "slate";

export interface LineBoardColumnColor {
  id: LineBoardColumnColorId;
  /** Accessible name for the swatch button. */
  label: string;
  /** Picker swatch fill. Stays readable at a small size. */
  className: string;
  /** Lane fill. Dark mode is dimmer than the swatch. */
  laneClassName: string;
}

export const LINE_BOARD_COLUMN_COLORS: LineBoardColumnColor[] = [
  {
    id: "lime",
    label: "Lime",
    className: "bg-lime-300 dark:bg-lime-800",
    laneClassName: "bg-lime-300 dark:bg-lime-950/40",
  },
  {
    id: "yellow",
    label: "Yellow",
    className: "bg-amber-300 dark:bg-amber-800",
    laneClassName: "bg-amber-300 dark:bg-amber-950/40",
  },
  {
    id: "teal",
    label: "Teal",
    className: "bg-teal-300 dark:bg-teal-800",
    laneClassName: "bg-teal-300 dark:bg-teal-950/40",
  },
  {
    id: "sky",
    label: "Sky",
    className: "bg-sky-300 dark:bg-sky-800",
    laneClassName: "bg-sky-300 dark:bg-sky-950/40",
  },
  {
    id: "purple",
    label: "Purple",
    className: "bg-violet-300 dark:bg-violet-800",
    laneClassName: "bg-violet-300 dark:bg-violet-950/40",
  },
  {
    id: "slate",
    label: "Slate",
    className: "bg-slate-300 dark:bg-slate-600",
    laneClassName: "bg-slate-300 dark:bg-slate-800/50",
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
