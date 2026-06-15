import { useMemo } from "react";
import { Loader2 } from "lucide-react";

import { MarkdownContent } from "../Markdown";

import { interpolateMarkdownTemplate } from "./markdownInterpolation";

/**
 * Render a markdown string with the dashboard's GFM + sanitize pipeline and
 * `{{ name.field }}` variable interpolation applied first. Returns `null`
 * when the resulting markdown is empty so the caller can decide whether to
 * show its own empty state.
 */
export function MarkdownBody({ body, vars }: { body: string; vars: Record<string, unknown> }) {
  // Interpolate `{{ name.field }}` (and `$["Node"]` run-node references) before
  // delegating to the shared markdown renderer so live values flow through the
  // same sanitize + GFM pipeline as static markdown. Trim here (not in
  // MarkdownContent) so panel-authored content keeps its historical "ignore
  // leading/trailing whitespace" behavior while the file viewer still sees
  // its content byte-for-byte.
  const interpolated = useMemo(() => interpolateMarkdownTemplate(body, vars).trim(), [body, vars]);
  return <MarkdownContent content={interpolated} data-testid="console-markdown" />;
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
    <div className="flex h-full min-h-[3rem] items-center justify-center" data-testid="console-markdown-loading">
      <Loader2 className="size-4 animate-spin text-slate-400" />
    </div>
  );
}
