/**
 * Soft lane colours for the line board. Swatches are small circles in the
 * column menu (Apple Notes / HoneyBook style). Lane fills stay pastel so
 * white cards stay readable.
 */

export type LineBoardColumnColorId =
  | "green"
  | "yellow"
  | "orange"
  | "red"
  | "purple"
  | "blue"
  | "sky"
  | "lime"
  | "pink"
  | "slate";

export interface LineBoardColumnColor {
  id: LineBoardColumnColorId;
  /** Accessible name for the swatch button. */
  label: string;
  /** Saturated circle in the picker. */
  swatchClassName: string;
  /** Soft fill for the whole lane. */
  laneClassName: string;
}

export const LINE_BOARD_COLUMN_COLORS: LineBoardColumnColor[] = [
  {
    id: "green",
    label: "Green",
    swatchClassName: "bg-emerald-500",
    laneClassName: "bg-emerald-100 dark:bg-emerald-950/50",
  },
  {
    id: "yellow",
    label: "Yellow",
    swatchClassName: "bg-amber-400",
    laneClassName: "bg-amber-100 dark:bg-amber-950/45",
  },
  {
    id: "orange",
    label: "Orange",
    swatchClassName: "bg-orange-500",
    laneClassName: "bg-orange-100 dark:bg-orange-950/45",
  },
  {
    id: "red",
    label: "Red",
    swatchClassName: "bg-rose-500",
    laneClassName: "bg-rose-100 dark:bg-rose-950/45",
  },
  {
    id: "purple",
    label: "Purple",
    swatchClassName: "bg-violet-500",
    laneClassName: "bg-violet-100 dark:bg-violet-950/45",
  },
  {
    id: "blue",
    label: "Blue",
    swatchClassName: "bg-blue-500",
    laneClassName: "bg-blue-100 dark:bg-blue-950/45",
  },
  {
    id: "sky",
    label: "Sky",
    swatchClassName: "bg-sky-500",
    laneClassName: "bg-sky-100 dark:bg-sky-950/45",
  },
  {
    id: "lime",
    label: "Lime",
    swatchClassName: "bg-lime-500",
    laneClassName: "bg-lime-100 dark:bg-lime-950/45",
  },
  {
    id: "pink",
    label: "Pink",
    swatchClassName: "bg-fuchsia-500",
    laneClassName: "bg-fuchsia-100 dark:bg-fuchsia-950/45",
  },
  {
    id: "slate",
    label: "Slate",
    swatchClassName: "bg-slate-500",
    laneClassName: "bg-slate-200 dark:bg-slate-800",
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
