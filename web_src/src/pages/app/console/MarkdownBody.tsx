import { useMemo } from "react";
import { Loader2 } from "lucide-react";

import { MarkdownContent } from "../Markdown";

import { useConsoleContext } from "./ConsoleContext";
import { interpolateMarkdownTemplate } from "./markdownInterpolation";

/**
 * Render a markdown string with the dashboard's GFM + sanitize pipeline and
 * `{{ name.field }}` variable interpolation applied first. Returns `null`
 * when the resulting markdown is empty so the caller can decide whether to
 * show its own empty state.
 *
 * Canvas ids come from `ConsoleContext` so `[label](node:…)` links render as
 * node chips the same way they do in the Files markdown preview. Link and
 * inline-code styling live on `MarkdownContent` so Files and Console match.
 */
export function MarkdownBody({
  body,
  vars,
  canvasId,
  organizationId,
}: {
  body: string;
  vars: Record<string, unknown>;
  canvasId?: string;
  organizationId?: string;
}) {
  const ctx = useConsoleContext();
  const resolvedCanvasId = canvasId ?? ctx?.canvasId;
  const resolvedOrganizationId = organizationId ?? ctx?.organizationId;

  // Interpolate `{{ name.field }}` (and `$["Node"]` run-node references) before
  // delegating to the shared markdown renderer so live values flow through the
  // same sanitize + GFM pipeline as static markdown. Trim here (not in
  // MarkdownContent) so panel-authored content keeps its historical "ignore
  // leading/trailing whitespace" behavior while the file viewer still sees
  // its content byte-for-byte.
  const interpolated = useMemo(() => interpolateMarkdownTemplate(body, vars).trim(), [body, vars]);
  return (
    <MarkdownContent
      content={interpolated}
      canvasId={resolvedCanvasId}
      organizationId={resolvedOrganizationId}
      data-testid="console-markdown"
    />
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
    <div className="flex h-full min-h-[3rem] items-center justify-center" data-testid="console-markdown-loading">
      <Loader2 className="size-4 animate-spin text-slate-400 dark:text-gray-500" />
    </div>
  );
}
