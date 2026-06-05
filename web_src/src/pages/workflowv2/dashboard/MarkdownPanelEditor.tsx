import { useMemo, useState, type RefObject } from "react";
import { ChevronDown, ChevronUp } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";

import { MarkdownBody } from "./MarkdownBody";
import { MarkdownVariablesPanel } from "./MarkdownVariablesPanel";
import { useMarkdownVariables } from "./useMarkdownVariables";
import type { MarkdownVariable } from "./panelTypes";

/**
 * In-card editor for a Markdown panel: title input, body textarea, live
 * preview, and the variables manager on the right rail. Kept in its own
 * module so the read-only render path in `MarkdownPanelCard.tsx` can stay
 * compact.
 */
export function MarkdownPanelEditor(props: {
  panelId: string;
  canvasId: string;
  draftTitle: string;
  setDraftTitle: (value: string) => void;
  draftBody: string;
  setDraftBody: (value: string) => void;
  draftVariables: MarkdownVariable[];
  setDraftVariables: (next: MarkdownVariable[]) => void;
  titleInputRef: RefObject<HTMLInputElement | null>;
  textareaRef: RefObject<HTMLTextAreaElement | null>;
  /** Set when the last save attempt was blocked by variable validation. */
  saveError?: string | null;
  onCancel: () => void;
  onCommit: () => void;
}) {
  const {
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
  } = props;

  const handleKeyDown = (e: React.KeyboardEvent) => {
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

  // Resolve the draft variables against live data so the inline preview
  // mirrors what the saved panel will show. We pass the draft title + body so
  // the hook only side-loads run node executions when either references `$[`
  // (both are interpolated with the same variables).
  const textForSideload = useMemo(() => `${draftTitle}\n${draftBody}`, [draftTitle, draftBody]);
  const { vars: previewVars, errors, isLoading } = useMarkdownVariables(canvasId, draftVariables, textForSideload);

  // Preview is collapsible — when collapsed the textarea reclaims the freed
  // vertical space, useful on shorter panel cards.
  const [previewCollapsed, setPreviewCollapsed] = useState(false);

  const insertAtCursor = (snippet: string) => {
    const el = textareaRef.current;
    if (!el) {
      setDraftBody(draftBody + snippet);
      return;
    }
    const start = el.selectionStart ?? draftBody.length;
    const end = el.selectionEnd ?? draftBody.length;
    const next = draftBody.slice(0, start) + snippet + draftBody.slice(end);
    setDraftBody(next);
    requestAnimationFrame(() => {
      el.focus();
      const caret = start + snippet.length;
      el.setSelectionRange(caret, caret);
    });
  };

  return (
    <div className="flex h-full w-full flex-col gap-0 overflow-hidden rounded-lg border border-slate-950/15 bg-white">
      <div className="flex items-center gap-2 rounded-t-lg px-2 py-1">
        <Input
          ref={titleInputRef}
          value={draftTitle}
          onChange={(e) => setDraftTitle(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder={panelId}
          aria-label="Panel title"
          className="h-7 border-0 bg-transparent px-2 text-xs font-medium text-slate-800 shadow-none focus-visible:ring-0 focus-visible:ring-offset-0"
          data-testid="dashboard-markdown-title-editor"
        />
      </div>
      <div className="grid min-h-0 flex-1 grid-cols-1 gap-0 lg:grid-cols-[minmax(0,1fr)_minmax(260px,360px)]">
        <div className="flex min-h-0 min-w-0 flex-col border-r border-slate-950/10">
          <Textarea
            ref={textareaRef}
            value={draftBody}
            onChange={(e) => setDraftBody(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Write **markdown** here. Use {{ name.field }} to reference variables."
            className="min-h-[120px] flex-1 resize-none rounded-none border-0 bg-white font-mono text-sm shadow-none focus-visible:ring-0 focus-visible:ring-offset-0"
            data-testid="dashboard-markdown-editor"
          />
          <MarkdownLivePreview
            body={draftBody}
            vars={previewVars}
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
          isLoading={isLoading}
          onInsertSnippet={insertAtCursor}
        />
      </div>
      <div className="flex items-center justify-between gap-2 rounded-b-lg border-t border-slate-950/10 bg-slate-50/50 px-3 py-1.5">
        {saveError ? (
          <span className="text-[11px] text-red-600" role="alert" data-testid="dashboard-markdown-save-error">
            {saveError}
          </span>
        ) : (
          <span className="text-[11px] text-slate-500">
            <kbd className="rounded border border-slate-200 bg-white px-1 font-mono">Esc</kbd> cancel &middot;{" "}
            <kbd className="rounded border border-slate-200 bg-white px-1 font-mono">Cmd+Enter</kbd> save
          </span>
        )}
        <div className="flex items-center gap-1">
          <Button type="button" size="sm" variant="outline" onClick={onCancel}>
            Cancel
          </Button>
          <Button type="button" size="sm" onClick={onCommit} data-testid="dashboard-markdown-save">
            Save
          </Button>
        </div>
      </div>
    </div>
  );
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
  body,
  vars,
  collapsed,
  onToggle,
}: {
  body: string;
  vars: Record<string, unknown>;
  collapsed: boolean;
  onToggle: () => void;
}) {
  return (
    <div
      className={cn(
        "flex flex-col overflow-hidden border-t border-slate-950/10 bg-slate-50/40",
        collapsed ? "shrink-0" : "min-h-[120px] flex-1",
      )}
      data-testid="dashboard-markdown-editor-preview"
    >
      <button
        type="button"
        onClick={onToggle}
        aria-expanded={!collapsed}
        className="flex items-center justify-between gap-2 border-b border-slate-950/10 px-3 py-1.5 text-left hover:bg-slate-100/60"
        data-testid="dashboard-markdown-editor-preview-toggle"
      >
        <span className="text-[11px] font-semibold uppercase tracking-wide text-slate-500">Preview</span>
        {collapsed ? (
          <ChevronUp className="size-3.5 text-slate-500" />
        ) : (
          <ChevronDown className="size-3.5 text-slate-500" />
        )}
      </button>
      {!collapsed ? (
        <div className="min-h-0 flex-1 overflow-auto px-3 py-2">
          {body.trim() ? (
            <MarkdownBody body={body} vars={vars} />
          ) : (
            <p className="text-[12px] text-slate-400">Preview will appear once you write markdown above.</p>
          )}
        </div>
      ) : null}
    </div>
  );
}
