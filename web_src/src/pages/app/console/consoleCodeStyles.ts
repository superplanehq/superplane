/**
 * Inline monospace badge used for SHAs, IDs, and other short code snippets
 * across console panels.
 */
export const CONSOLE_CODE_BADGE_CLASSES =
  "rounded bg-action-neutral px-1 py-0.5 font-mono text-xs text-content-primary";

/** Selector utilities for `<code>` inside html/markdown roots (`MarkdownContent`, HTML panels). */
export const CONSOLE_CODE_BADGE_ANCHOR_SELECTOR_CLASSES =
  "[&_code]:rounded [&_code]:bg-action-neutral [&_code]:px-1 [&_code]:py-0.5 [&_code]:font-mono [&_code]:text-xs [&_code]:text-content-primary";
