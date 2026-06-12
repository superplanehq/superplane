import { useId, useMemo } from "react";
import { Loader2 } from "lucide-react";

import { cn } from "@/lib/utils";

import { HTML_WIDGET_ROOT_ATTR, sanitizeHtml } from "./htmlSanitize";
import { interpolateMarkdownTemplate } from "./markdownInterpolation";

/**
 * Tailwind class string applied to the html widget's root element. Sets a
 * sensible default typography so authors get readable text out of the box
 * without having to add classes for every paragraph; user-authored CSS can
 * override these via the scoped `<style>` block (or `class=`-based overrides
 * that survive the curated Tailwind safelist).
 */
const HTML_ROOT_CLASSES =
  "max-w-none text-sm text-slate-800 " +
  "[&_h1]:mb-1.5 [&_h1]:mt-1 [&_h1]:text-lg [&_h1]:font-semibold [&_h1]:leading-tight [&_h1:first-child]:mt-0 " +
  "[&_h2]:mb-1 [&_h2]:mt-1 [&_h2]:text-base [&_h2]:font-semibold [&_h2]:leading-tight [&_h2:first-child]:mt-0 " +
  "[&_h3]:mb-0.5 [&_h3]:mt-1 [&_h3]:text-sm [&_h3]:font-semibold [&_h3]:leading-tight [&_h3:first-child]:mt-0 " +
  "[&_p]:mb-2 [&_p]:leading-relaxed " +
  "[&_ol]:mb-2 [&_ol]:ml-5 [&_ol]:list-decimal " +
  "[&_ul]:mb-2 [&_ul]:ml-5 [&_ul]:list-disc [&_li]:mb-1 " +
  "[&_blockquote]:my-2 [&_blockquote]:border-l-2 [&_blockquote]:border-slate-300 [&_blockquote]:pl-3 " +
  "[&_code]:rounded [&_code]:bg-slate-100 [&_code]:px-1 [&_code]:py-0.5 [&_code]:text-xs " +
  "[&_pre]:my-2 [&_pre]:overflow-auto [&_pre]:rounded [&_pre]:bg-slate-100 [&_pre]:p-2 " +
  "[&_pre_code]:bg-transparent [&_pre_code]:p-0 " +
  "[&_a]:underline [&_a]:underline-offset-2 [&_a]:decoration-current " +
  "[&_table]:my-2 [&_table]:text-xs [&_table]:border-collapse [&_th]:border [&_th]:border-slate-200 [&_th]:px-2 [&_th]:py-1 " +
  "[&_td]:border [&_td]:border-slate-100 [&_td]:px-2 [&_td]:py-1";

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
    <div
      className={cn(HTML_ROOT_CLASSES)}
      data-testid="console-html"
      {...{ [HTML_WIDGET_ROOT_ATTR]: rootId }}
      dangerouslySetInnerHTML={{ __html: sanitized }}
    />
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
      <Loader2 className="size-4 animate-spin text-slate-400" />
    </div>
  );
}
