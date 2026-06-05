import { useMemo } from "react";
import { Loader2 } from "lucide-react";
import ReactMarkdown from "react-markdown";
import rehypeRaw from "rehype-raw";
import rehypeSanitize, { defaultSchema } from "rehype-sanitize";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";

import { cn } from "@/lib/utils";

import { interpolateMarkdownTemplate } from "./markdownInterpolation";

/**
 * Tailwind class string used to style the rendered markdown body. We don't use
 * the official `prose` plugin so panels stay visually consistent with the rest
 * of the canvas chrome at small panel sizes.
 */
const MARKDOWN_CLASSES =
  "max-w-none text-sm text-slate-800 " +
  "[&_h1]:mb-1.5 [&_h1]:mt-1 [&_h1]:text-lg [&_h1]:font-semibold [&_h1]:leading-tight [&_h1:first-child]:mt-0 " +
  "[&_h2]:mb-1 [&_h2]:mt-1 [&_h2]:text-base [&_h2]:font-semibold [&_h2]:leading-tight [&_h2:first-child]:mt-0 " +
  "[&_h3]:mb-0.5 [&_h3]:mt-1 [&_h3]:text-sm [&_h3]:font-semibold [&_h3]:leading-tight [&_h3:first-child]:mt-0 " +
  "[&_h4]:mb-0.5 [&_h4]:mt-1 [&_h4]:text-sm [&_h4]:font-medium [&_h4]:leading-tight [&_h4:first-child]:mt-0 " +
  "[&_p]:mb-2 [&_p]:leading-relaxed " +
  "[&_ol]:mb-2 [&_ol]:ml-5 [&_ol]:list-decimal " +
  "[&_ul]:mb-2 [&_ul]:ml-5 [&_ul]:list-disc [&_li]:mb-1 " +
  "[&_blockquote]:my-2 [&_blockquote]:border-l-2 [&_blockquote]:border-slate-300 [&_blockquote]:pl-3 " +
  "[&_code]:rounded [&_code]:bg-slate-100 [&_code]:px-1 [&_code]:py-0.5 [&_code]:text-xs " +
  "[&_pre]:my-2 [&_pre]:overflow-auto [&_pre]:rounded [&_pre]:bg-slate-100 [&_pre]:p-2 " +
  "[&_pre_code]:bg-transparent [&_pre_code]:p-0 " +
  "[&_a]:underline [&_a]:underline-offset-2 [&_a]:decoration-current " +
  "[&_table]:my-2 [&_table]:text-xs [&_table]:border-collapse [&_th]:border [&_th]:border-slate-200 [&_th]:px-2 [&_th]:py-1 " +
  "[&_td]:border [&_td]:border-slate-100 [&_td]:px-2 [&_td]:py-1 " +
  "[&_details]:my-3 [&_details]:rounded-md [&_details]:border [&_details]:border-slate-200 [&_details]:bg-slate-50/60 [&_details]:p-3 " +
  "[&_details>summary]:flex [&_details>summary]:items-center [&_details>summary]:cursor-pointer [&_details>summary]:select-none [&_details>summary]:text-sm [&_details>summary]:font-semibold [&_details>summary]:text-slate-900 [&_details>summary]:list-none [&_details>summary]:marker:hidden [&_details>summary]:hover:text-sky-700 " +
  "[&_details>summary]:before:content-['▸'] [&_details>summary]:before:mr-2 [&_details>summary]:before:text-slate-500 [&_details>summary]:before:transition-transform [&_details>summary]:before:duration-200 " +
  "[&_details[open]>summary]:mb-3 [&_details[open]>summary]:before:rotate-90 " +
  "[&_details>*:last-child]:mb-0";

/**
 * Sanitize schema extending the rehype-sanitize defaults with `<details>` /
 * `<summary>` (plus the `open` attribute) so collapsible sections can be
 * authored directly in markdown without weakening the rest of the policy
 * around scripts, event handlers, and inline styles.
 */
const MARKDOWN_SANITIZE_SCHEMA = {
  ...defaultSchema,
  tagNames: [...(defaultSchema.tagNames ?? []), "details", "summary"],
  attributes: {
    ...(defaultSchema.attributes ?? {}),
    details: [...(defaultSchema.attributes?.details ?? []), "open"],
  },
};

/**
 * Render a markdown string with the dashboard's GFM + sanitize pipeline and
 * `{{ name.field }}` variable interpolation applied first. Returns `null`
 * when the resulting markdown is empty so the caller can decide whether to
 * show its own empty state.
 */
export function MarkdownBody({ body, vars }: { body: string; vars: Record<string, unknown> }) {
  // Interpolate `{{ name.field }}` (and `$["Node"]` run-node references) before
  // normalizing line endings so the resulting text flows through the same
  // sanitize + GFM pipeline as static markdown.
  const interpolated = useMemo(() => interpolateMarkdownTemplate(body, vars), [body, vars]);
  const normalized = useMemo(() => interpolated.replace(/\r\n/g, "\n").trim(), [interpolated]);
  if (!normalized) return null;
  return (
    <div className={cn(MARKDOWN_CLASSES)} data-testid="dashboard-markdown">
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkBreaks]}
        rehypePlugins={[rehypeRaw, [rehypeSanitize, MARKDOWN_SANITIZE_SCHEMA]]}
      >
        {normalized}
      </ReactMarkdown>
    </div>
  );
}

/**
 * Loading placeholder shown in place of the rendered markdown body while the
 * panel's variables (notably the per-run execution side-load behind
 * `{{ run.$["Node"]... }}`) are still resolving. Mirrors `WidgetTable`'s
 * spinner so live-data panels share a consistent loading affordance instead of
 * flashing empty interpolated fields. Shared between the read-only card view
 * and the in-card editor preview so both gate on the same loading state.
 */
export function MarkdownBodyLoading() {
  return (
    <div className="flex h-full min-h-[3rem] items-center justify-center" data-testid="dashboard-markdown-loading">
      <Loader2 className="size-4 animate-spin text-slate-400" />
    </div>
  );
}
