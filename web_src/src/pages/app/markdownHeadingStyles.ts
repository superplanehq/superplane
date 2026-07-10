import { cn } from "@/lib/utils";

/** Vertical rhythm for markdown/HTML headings — Tailwind `my-4` (= `1rem 0`). */
export const MARKDOWN_HEADING_MARGIN_CLASSES = "my-4 first:mt-0";

const MARKDOWN_HEADING_TYPOGRAPHY = {
  h1: "text-lg font-semibold leading-tight",
  h2: "text-base font-semibold leading-tight",
  h3: "text-sm font-semibold leading-tight",
  h4: "text-sm font-medium leading-tight",
} as const;

export type MarkdownHeadingLevel = keyof typeof MARKDOWN_HEADING_TYPOGRAPHY;

export function markdownHeadingClassName(level: MarkdownHeadingLevel): string {
  return cn(MARKDOWN_HEADING_MARGIN_CLASSES, MARKDOWN_HEADING_TYPOGRAPHY[level]);
}

/** Descendant selectors for raw HTML panel headings (same `my-4` rhythm). */
export const MARKDOWN_HEADING_MARGIN_SELECTOR_CLASSES =
  "[&_h1]:my-4 [&_h1]:first:mt-0 [&_h2]:my-4 [&_h2]:first:mt-0 [&_h3]:my-4 [&_h3]:first:mt-0 [&_h4]:my-4 [&_h4]:first:mt-0";

export const MARKDOWN_HEADING_TYPOGRAPHY_SELECTOR_CLASSES =
  "[&_h1]:text-lg [&_h1]:font-semibold [&_h1]:leading-tight " +
  "[&_h2]:text-base [&_h2]:font-semibold [&_h2]:leading-tight " +
  "[&_h3]:text-sm [&_h3]:font-semibold [&_h3]:leading-tight " +
  "[&_h4]:text-sm [&_h4]:font-medium [&_h4]:leading-tight";
