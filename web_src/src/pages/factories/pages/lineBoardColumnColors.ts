/**
 * Lane colours for the line board. The picker circle and the lane use the
 * same fill so the colour you tap is the colour you see on the board.
 */

export type LineBoardColumnColorId = "lime" | "yellow" | "red" | "sky" | "purple" | "slate";

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
  { id: "red", label: "Red", className: "bg-rose-300 dark:bg-rose-800" },
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
