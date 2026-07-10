/**
 * Inline monospace badge used for SHAs, IDs, and other short code snippets
 * across console panels. `bg-gray-950/5` keeps contrast on light and dark
 * row/panel backgrounds without theme-specific fill colors.
 */
export const CONSOLE_CODE_BADGE_CLASSES =
  "rounded bg-gray-950/5 px-1 py-0.5 font-mono text-xs text-slate-800 dark:text-gray-100";

/** Selector utilities for `<code>` inside html/markdown roots (`MarkdownContent`, HTML panels). */
export const CONSOLE_CODE_BADGE_ANCHOR_SELECTOR_CLASSES =
  "[&_code]:rounded [&_code]:bg-gray-950/5 [&_code]:px-1 [&_code]:py-0.5 [&_code]:font-mono [&_code]:text-xs [&_code]:text-slate-800 dark:[&_code]:text-gray-100";
