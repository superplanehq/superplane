import { useMemo, useState, type RefObject } from "react";
import { ChevronDown, ChevronUp } from "lucide-react";

import { ExpressionEditor } from "@/components/ExpressionEditor";
import { Button } from "@/components/ui/button";
import { useResponsiveRailCollapse } from "@/hooks/useResponsiveRailCollapse";
import { cn } from "@/lib/utils";

import { interpolateMarkdownTemplate } from "./markdownInterpolation";
import { MarkdownBody, MarkdownBodyLoading } from "./MarkdownBody";
import { markdownTextIsLoading } from "./markdownInterpolation";
import { MarkdownVariablesPanel } from "./MarkdownVariablesPanel";
import { useMarkdownVariables } from "./useMarkdownVariables";
import type { MarkdownVariable } from "./panelTypes";

/**
 * In-card editor for a Markdown panel: title input, body textarea, live
 * preview, and the variables manager on the right rail. Kept in its own
 * module so the read-only render path in `MarkdownPanelCard.tsx` can stay
 * compact.
 */
/**
 * Resolve the draft variables against live data for the editor preview, and
 * derive the loading gate the preview shares with the read-only view.
 *
 * The draft title + body are passed as the side-load text so run-node
 * executions are only fetched when either interpolates a `$[` reference (both
 * use the same variable map). `previewLoading` mirrors `markdownTextIsLoading`
 * so the preview shows a spinner instead of flashing empty
 * `{{ run.$["Node"]... }}` fields while the per-run execution side-load is in
 * flight — keeping it consistent with the saved panel.
 */
function useMarkdownEditorPreview(
  canvasId: string,
  draftTitle: string,
  draftBody: string,
  draftVariables: MarkdownVariable[],
) {
  const textForSideload = useMemo(() => `${draftTitle}\n${draftBody}`, [draftTitle, draftBody]);
  const { vars, errors, baseLoading, sideloadLoading, searchingNames } = useMarkdownVariables(
    canvasId,
    draftVariables,
    textForSideload,
  );
  const bodyPreviewLoading = markdownTextIsLoading(draftBody, baseLoading, sideloadLoading, searchingNames);
  const titlePreviewLoading = markdownTextIsLoading(draftTitle, baseLoading, sideloadLoading, searchingNames);
  return {
    previewVars: vars,
    errors,
    baseLoading,
    sideloadLoading,
    searchingNames,
    bodyPreviewLoading,
    titlePreviewLoading,
  };
}

interface MarkdownPanelEditorProps {
  panelId: string;
  canvasId: string;
  draftTitle: string;
  setDraftTitle: (value: string) => void;
  draftBody: string;
  setDraftBody: (value: string) => void;
  draftVariables: MarkdownVariable[];
  setDraftVariables: (next: MarkdownVariable[]) => void;
  titleInputRef: RefObject<HTMLTextAreaElement | null>;
  textareaRef: RefObject<HTMLTextAreaElement | null>;
  /** Set when the last save attempt was blocked by variable validation. */
  saveError?: string | null;
  onCancel: () => void;
  onCommit: () => void;
}

export function MarkdownPanelEditor({
  panelId,
  canvasId,
  draftTitle,
  setDraftTitle,
  draftBody,
  setDraftBody,
  draftVariables,
  setDraftVariables,
  titleInputRef,
  textareaRef,
  saveError,
  onCancel,
  onCommit,
}: MarkdownPanelEditorProps) {
  const handleShortcutKeys = (e: React.KeyboardEvent) => {
    if (e.key === "Escape") {
      e.preventDefault();
      onCancel();
      return;
    }
    if ((e.metaKey || e.ctrlKey) && e.key === "Enter") {
      e.preventDefault();
      onCommit();
    }
  };

  const handleTitleKeyDown = (e: React.KeyboardEvent) => {
    // Titles are conceptually single-line — swallow bare Enter so the field
    // acts like an <input> and doesn't accumulate stray newlines.
    if (e.key === "Enter" && !(e.metaKey || e.ctrlKey || e.shiftKey || e.altKey)) {
      e.preventDefault();
      return;
    }
    handleShortcutKeys(e);
  };

  const { previewVars, errors, baseLoading, sideloadLoading, searchingNames, bodyPreviewLoading, titlePreviewLoading } =
    useMarkdownEditorPreview(canvasId, draftTitle, draftBody, draftVariables);

  const previewTitle = useMemo(() => {
    if (titlePreviewLoading) return "";
    return interpolateMarkdownTemplate(draftTitle, previewVars).trim();
  }, [draftTitle, previewVars, titlePreviewLoading]);

  // Preview is collapsible — when collapsed the textarea reclaims the freed
  // vertical space, useful on shorter panel cards.
  const [previewCollapsed, setPreviewCollapsed] = useState(false);

  // Variables rail collapses automatically when the parent widget is narrow.
  // The manual toggle wins until the breakpoint flips again, at which point
  // the auto behavior takes over so resizes always honor the current width.
  const {
    containerRef: gridRef,
    collapsed: variablesCollapsed,
    toggle: toggleVariablesCollapsed,
  } = useResponsiveRailCollapse();

  const insertAtCursor = (snippet: string) =>
    insertSnippetAtTextareaCursor(textareaRef.current, draftBody, snippet, setDraftBody);

  return (
    <div className="flex h-full w-full flex-col gap-0 overflow-hidden rounded-lg border border-slate-950/15 bg-white dark:border-gray-700/70 dark:bg-gray-900">
      <div className="flex items-center gap-2 rounded-t-lg px-2 py-1" onKeyDown={handleTitleKeyDown}>
        <ExpressionEditor
          ref={titleInputRef}
          dialect="cel"
          syntaxProfile="wrapped"
          value={draftTitle}
          onChange={setDraftTitle}
          exampleObj={previewVars}
          placeholder={panelId}
          aria-label="Panel title"
          inputSize="xs"
          className="border-0 bg-transparent px-2 shadow-none focus-within:ring-0"
          data-testid="console-markdown-title-editor"
          quickTip="Tip: type `{{` to reference a variable."
        />
      </div>
      <div
        ref={gridRef}
        className={cn(
          "grid min-h-0 flex-1 grid-cols-1 gap-0",
          variablesCollapsed ? "grid-cols-[minmax(0,1fr)_auto]" : "grid-cols-[minmax(0,1fr)_minmax(220px,320px)]",
        )}
      >
        <div className="flex min-h-0 min-w-0 flex-col border-r border-slate-950/10 dark:border-gray-800">
          <div className="flex min-h-[120px] flex-1 flex-col" onKeyDown={handleShortcutKeys}>
            <ExpressionEditor
              ref={textareaRef}
              dialect="cel"
              syntaxProfile="wrapped"
              value={draftBody}
              onChange={setDraftBody}
              exampleObj={previewVars}
              placeholder="Write **markdown** here. Use {{ name.field }} to reference variables."
              className="flex-1 rounded-none border-0 bg-white font-mono shadow-none focus-within:ring-0 dark:bg-gray-900"
              fullHeight
              data-testid="console-markdown-editor"
              quickTip="Tip: type `{{` to reference a variable."
            />
          </div>
          <MarkdownLivePreview
            title={previewTitle}
            body={draftBody}
            vars={previewVars}
            loading={bodyPreviewLoading}
            collapsed={previewCollapsed}
            onToggle={() => setPreviewCollapsed((prev) => !prev)}
          />
        </div>
        <MarkdownVariablesPanel
          canvasId={canvasId}
          draftBody={draftBody}
          draftVariables={draftVariables}
          setDraftVariables={setDraftVariables}
          previewVars={previewVars}
          errors={errors}
          baseLoading={baseLoading}
          sideloadLoading={sideloadLoading}
          searchingNames={searchingNames}
          onInsertSnippet={insertAtCursor}
          collapsed={variablesCollapsed}
          onToggleCollapsed={toggleVariablesCollapsed}
        />
      </div>
      <MarkdownEditorFooter saveError={saveError} onCancel={onCancel} onCommit={onCommit} />
    </div>
  );
}

function MarkdownEditorFooter({
  saveError,
  onCancel,
  onCommit,
}: {
  saveError?: string | null;
  onCancel: () => void;
  onCommit: () => void;
}) {
  return (
    <div className="flex items-center justify-between gap-2 rounded-b-lg border-t border-slate-950/10 bg-slate-50/50 px-3 py-1.5 dark:border-gray-800 dark:bg-gray-800/50">
      {saveError ? (
        <span
          className="text-[11px] text-red-600 dark:text-red-400"
          role="alert"
          data-testid="console-markdown-save-error"
        >
          {saveError}
        </span>
      ) : (
        <span className="text-[11px] text-slate-500 dark:text-gray-400">
          <kbd className="rounded border border-slate-200 bg-white px-1 font-mono dark:border-gray-600 dark:bg-gray-900">
            Esc
          </kbd>{" "}
          cancel &middot;{" "}
          <kbd className="rounded border border-slate-200 bg-white px-1 font-mono dark:border-gray-600 dark:bg-gray-900">
            Cmd+Enter
          </kbd>{" "}
          save
        </span>
      )}
      <div className="flex items-center gap-1">
        <Button type="button" size="sm" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="button" size="sm" onClick={onCommit} data-testid="console-markdown-save">
          Save
        </Button>
      </div>
    </div>
  );
}

/**
 * Insert `snippet` at the textarea's current selection (replacing any
 * selected range), updating both the controlled value via `setBody` and
 * restoring focus + caret after the React re-render. Falls back to an
 * append when the element isn't mounted yet — matches the behavior the
 * inline implementation had before this was extracted.
 */
function insertSnippetAtTextareaCursor(
  el: HTMLTextAreaElement | null,
  body: string,
  snippet: string,
  setBody: (next: string) => void,
) {
  if (!el) {
    setBody(body + snippet);
    return;
  }
  const start = el.selectionStart ?? body.length;
  const end = el.selectionEnd ?? body.length;
  setBody(body.slice(0, start) + snippet + body.slice(end));
  requestAnimationFrame(() => {
    el.focus();
    const caret = start + snippet.length;
    el.setSelectionRange(caret, caret);
  });
}

/**
 * Render the live interpolated preview shown beneath the textarea while
 * editing. Uses the same `MarkdownBody` pipeline so authors see what saving
 * would actually produce.
 *
 * Collapsible so authors can reclaim vertical space for the textarea on
 * smaller cards; collapsed state keeps the header bar visible (with the
 * toggle) so the preview can always be brought back without leaving the
 * editor.
 */
function MarkdownLivePreview({
  title,
  body,
  vars,
  loading,
  collapsed,
  onToggle,
}: {
  title: string;
  body: string;
  vars: Record<string, unknown>;
  loading: boolean;
  collapsed: boolean;
  onToggle: () => void;
}) {
  return (
    <div
      className={cn(
        "flex flex-col overflow-hidden border-t border-slate-950/10 bg-slate-50/40 dark:border-gray-800 dark:bg-gray-800/40",
        collapsed ? "shrink-0" : "min-h-[120px] flex-1",
      )}
      data-testid="console-markdown-editor-preview"
    >
      <button
        type="button"
        onClick={onToggle}
        aria-expanded={!collapsed}
        className="flex items-center justify-between gap-2 border-b border-slate-950/10 px-3 py-1.5 text-left hover:bg-slate-100/60 dark:border-gray-800 dark:hover:bg-gray-800/60"
        data-testid="console-markdown-editor-preview-toggle"
      >
        <span className="text-[11px] font-semibold uppercase tracking-wide text-slate-500 dark:text-gray-400">
          Preview
        </span>
        {collapsed ? (
          <ChevronUp className="size-3.5 text-slate-500 dark:text-gray-400" />
        ) : (
          <ChevronDown className="size-3.5 text-slate-500 dark:text-gray-400" />
        )}
      </button>
      {!collapsed ? (
        <div className="min-h-0 flex-1 overflow-auto px-3 py-2">
          {title ? (
            <div
              className="mb-1.5 truncate text-[13px] font-medium text-slate-700 dark:text-gray-200"
              data-testid="console-markdown-editor-preview-title"
              title={title}
            >
              {title}
            </div>
          ) : null}
          {body.trim() ? (
            loading ? (
              <MarkdownBodyLoading />
            ) : (
              <MarkdownBody body={body} vars={vars} />
            )
          ) : (
            <p className="text-[12px] text-slate-400 dark:text-gray-500">
              Preview will appear once you write markdown above.
            </p>
          )}
        </div>
      ) : null}
    </div>
  );
}
