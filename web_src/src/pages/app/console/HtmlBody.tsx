import { useId, useMemo } from "react";
import { Loader2 } from "lucide-react";

import { cn } from "@/lib/utils";

import { CONSOLE_CODE_BADGE_ANCHOR_SELECTOR_CLASSES } from "./consoleCodeStyles";
import { CONSOLE_LINK_ANCHOR_SELECTOR_CLASSES } from "./consoleLinkStyles";
import { HTML_WIDGET_ROOT_ATTR, sanitizeHtml } from "./htmlSanitize";
import { interpolateMarkdownTemplate } from "./markdownInterpolation";
import {
  MARKDOWN_HEADING_MARGIN_SELECTOR_CLASSES,
  MARKDOWN_HEADING_TYPOGRAPHY_SELECTOR_CLASSES,
} from "../markdownHeadingStyles";
import { MARKDOWN_TABLE_SELECTOR_CLASSES } from "../markdownTableStyles";

/**
 * Tailwind class string applied to the html widget's root element. Sets a
 * sensible default typography so authors get readable text out of the box
 * without having to add classes for every paragraph; user-authored CSS can
 * override these via the scoped `<style>` block (or `class=`-based overrides
 * that survive the curated Tailwind safelist).
 */
const HTML_ROOT_CLASSES =
  "max-w-none text-[13px] text-slate-800 " +
  `${MARKDOWN_HEADING_MARGIN_SELECTOR_CLASSES} ${MARKDOWN_HEADING_TYPOGRAPHY_SELECTOR_CLASSES}` +
  "[&_p]:mb-2 [&_p]:leading-relaxed " +
  "[&_ol]:mb-2 [&_ol]:ml-5 [&_ol]:list-decimal " +
  "[&_ul]:mb-2 [&_ul]:ml-5 [&_ul]:list-disc [&_li]:mb-1 " +
  "[&_blockquote]:my-2 [&_blockquote]:border-l-2 [&_blockquote]:border-slate-300 [&_blockquote]:pl-3 " +
  "[&_pre]:my-2 [&_pre]:overflow-auto [&_pre]:rounded [&_pre]:bg-slate-100 [&_pre]:p-2 " +
  "[&_pre_code]:bg-transparent [&_pre_code]:p-0 " +
  `${MARKDOWN_TABLE_SELECTOR_CLASSES} `;

/**
 * Render a sanitized HTML string with `{{ name.field }}` variable
 * interpolation applied first. The output is rendered inside a scoped root
 * element so that any `<style>` blocks the author included are anchored at
 * `[data-console-html-root="<rootId>"]` and cannot leak styles outside the
 * widget.
 *
 * Returns `null` when the sanitized output is empty so the caller can show
 * its own empty state (mirrors `MarkdownBody`).
 */
export function HtmlBody({ body, vars }: { body: string; vars: Record<string, unknown> }) {
  // useId gives a unique, stable id per mounted instance so multiple html
  // panels on the same dashboard cannot accidentally share scoped styles.
  // React's generated ids contain `:` which is invalid inside a CSS
  // attribute selector value when un-escaped; we encode it to keep CSSOM
  // parsing happy in both the scoped-selector rewrite and the rendered DOM.
  const reactId = useId();
  const rootId = useMemo(() => reactId.replace(/[^a-zA-Z0-9_-]/g, "_"), [reactId]);

  const sanitized = useMemo(() => {
    const interpolated = interpolateMarkdownTemplate(body, vars);
    return sanitizeHtml(interpolated, rootId);
  }, [body, vars, rootId]);

  if (!sanitized.trim()) return null;

  return (
    // Link colors live on an outer wrapper so `dark:` variants apply even though
    // the sanitized html root uses `dark-mode-disabled` (which opts out of dark:).
    <div className={cn(CONSOLE_LINK_ANCHOR_SELECTOR_CLASSES, CONSOLE_CODE_BADGE_ANCHOR_SELECTOR_CLASSES)}>
      <div
        className={cn("dark-mode-disabled", HTML_ROOT_CLASSES)}
        data-testid="console-html"
        {...{ [HTML_WIDGET_ROOT_ATTR]: rootId }}
        dangerouslySetInnerHTML={{ __html: sanitized }}
      />
    </div>
  );
}

/**
 * Loading placeholder shown while the variables backing a templated body are
 * still resolving. Mirrors `MarkdownBodyLoading` so live-data panels share a
 * consistent loading affordance.
 */
export function HtmlBodyLoading() {
  return (
    <div className="flex h-full min-h-[3rem] items-center justify-center" data-testid="console-html-loading">
      <Loader2 className="size-4 animate-spin text-slate-400 dark:text-gray-500" />
    </div>
  );
}
