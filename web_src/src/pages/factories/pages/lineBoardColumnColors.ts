/**
 * Lane colours for the line board. The picker circle and the lane use the
 * same fill so the colour you tap is the colour you see on the board.
 */

export type LineBoardColumnColorId = "lime" | "yellow" | "teal" | "sky" | "purple" | "slate";

export interface LineBoardColumnColor {
  id: LineBoardColumnColorId;
  /** Accessible name for the swatch button. */
  label: string;
  /** Shared fill for the picker circle and the lane. */
  className: string;
}

export const LINE_BOARD_COLUMN_COLORS: LineBoardColumnColor[] = [
  { id: "lime", label: "Lime", className: "bg-lime-300 dark:bg-lime-800" },
  { id: "yellow", label: "Yellow", className: "bg-amber-300 dark:bg-amber-800" },
  { id: "teal", label: "Teal", className: "bg-teal-300 dark:bg-teal-800" },
  { id: "sky", label: "Sky", className: "bg-sky-300 dark:bg-sky-800" },
  { id: "purple", label: "Purple", className: "bg-violet-300 dark:bg-violet-800" },
  { id: "slate", label: "Slate", className: "bg-slate-300 dark:bg-slate-600" },
];

export function lineBoardColumnColorById(id: string | null | undefined): LineBoardColumnColor | undefined {
  if (!id) {
    return undefined;
  }
  return LINE_BOARD_COLUMN_COLORS.find((color) => color.id === id);
}

export function lineBoardColumnLaneClassName(id: string | null | undefined): string | undefined {
  return lineBoardColumnColorById(id)?.className;
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

const PHASE_COLUMN_KEY_PREFIX = "phase-";

/**
 * After a line step is removed, shift later phase color keys down so
 * `phase-2` becomes `phase-1`. Fixed bookends (backlog, verify, done)
 * keep their keys. The removed phase color is dropped.
 */
export function remapColumnColorsAfterRemovedStep(
  columnColors: Record<string, LineBoardColumnColorId | null>,
  removedStepIndex: number,
): Record<string, LineBoardColumnColorId | null> {
  const remapped: Record<string, LineBoardColumnColorId | null> = {};
  for (const [key, value] of Object.entries(columnColors)) {
    if (!key.startsWith(PHASE_COLUMN_KEY_PREFIX)) {
      remapped[key] = value;
      continue;
    }
    const index = Number.parseInt(key.slice(PHASE_COLUMN_KEY_PREFIX.length), 10);
    if (Number.isNaN(index) || index === removedStepIndex) {
      continue;
    }
    if (index > removedStepIndex) {
      remapped[`${PHASE_COLUMN_KEY_PREFIX}${index - 1}`] = value;
      continue;
    }
    remapped[key] = value;
  }
  return remapped;
}
