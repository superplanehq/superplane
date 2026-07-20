import { cn } from "@/lib/utils";

import { CONSOLE_TABLE_HEAD_BORDER_DARK_CLASSES } from "./console/consoleTableStyles";

/** Shared cell border for markdown/HTML tables — one color for `th` and `td`. */
export const MARKDOWN_TABLE_CELL_BORDER_CLASSES = "border border-slate-200 px-2 py-1 dark:border-gray-800";

export const MARKDOWN_TABLE_HEAD_BORDER_CLASSES = cn(
  "border border-slate-200 px-2 py-1",
  CONSOLE_TABLE_HEAD_BORDER_DARK_CLASSES,
);

export const MARKDOWN_TABLE_CLASSES = "my-2 w-full border-collapse text-[13px]";

export const MARKDOWN_TABLE_HEAD_CLASSES = cn(MARKDOWN_TABLE_HEAD_BORDER_CLASSES, "font-semibold text-left");

export const MARKDOWN_TABLE_DATA_CLASSES = MARKDOWN_TABLE_CELL_BORDER_CLASSES;

/** Cap emphasis inside tables at semibold (600), never browser bold (700). */
export const MARKDOWN_TABLE_EMPHASIS_CLASSES = "font-semibold";

/** Descendant selectors for raw HTML panel tables. */
export const MARKDOWN_TABLE_SELECTOR_CLASSES =
  "[&_table]:my-2 [&_table]:w-full [&_table]:border-collapse [&_table]:text-[13px] " +
  "[&_th]:border [&_th]:border-slate-200 [&_th]:px-2 [&_th]:py-1 [&_th]:font-semibold [&_th]:text-left dark:[&_th]:border-gray-800 " +
  "[&_td]:border [&_td]:border-slate-200 [&_td]:px-2 [&_td]:py-1 dark:[&_td]:border-gray-800 " +
  "[&_table_strong]:font-semibold [&_table_b]:font-semibold";
