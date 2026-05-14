import { useEffect, useMemo, useRef, useState } from "react";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "@/components/ui/button";
import { CanvasMarkdown } from "@/ui/Markdown/CanvasMarkdown";
import { AppsPanelMarkdownLinksScope } from "@/ui/Markdown/appsPanelMarkdownLinksContext";
import { WidgetBlock } from "@/ui/Markdown/WidgetBlock";
import { Pencil } from "lucide-react";
import type { PanelRenderProps } from "./panelRegistry";

export type MarkdownPanelContent = {
  body: string;
};

// A panel whose body contains exactly one fenced ```widget (or legacy
// ```query) block — possibly with surrounding markdown like a `# Heading`
// above it — is rendered in fill mode: the markdown above and below keeps
// its natural height while the widget stretches to consume whatever vertical
// space is left. This covers the common authoring shape of "title + chart".
//
// Two-or-more fences fall through to the regular markdown wrapper to avoid
// silently picking one widget to grow at the expense of the others.
const WIDGET_FENCE_RE = /```(?:widget|query)\s*\n([\s\S]*?)\n```/g;

function parseWidgetPanelBody(body: string): { leading: string; widgetBody: string; trailing: string } | null {
  const matches = Array.from(body.matchAll(WIDGET_FENCE_RE));
  if (matches.length !== 1) return null;
  const m = matches[0];
  const start = m.index ?? 0;
  return {
    leading: body.slice(0, start),
    widgetBody: m[1],
    trailing: body.slice(start + m[0].length),
  };
}

const PLACEHOLDER = `# New panel

Write **markdown** here. You can reference canvas nodes with \`@node-name\`.`;

export function MarkdownPanel(props: PanelRenderProps<MarkdownPanelContent>) {
  return (
    <AppsPanelMarkdownLinksScope>
      <MarkdownPanelInner {...props} />
    </AppsPanelMarkdownLinksScope>
  );
}

function MarkdownPanelInner({ content, readOnly, onChange, ctx }: PanelRenderProps<MarkdownPanelContent>) {
  const [isEditing, setIsEditing] = useState(false);
  const [draft, setDraft] = useState(content.body);
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);
  const widgetPanel = useMemo(() => parseWidgetPanelBody(content.body), [content.body]);

  // Reset local draft when the upstream content changes outside of an active
  // edit (e.g. another tab updated the launchpad). Avoid clobbering an active
  // edit by checking isEditing.
  useEffect(() => {
    if (!isEditing) {
      setDraft(content.body);
    }
  }, [content.body, isEditing]);

  useEffect(() => {
    if (isEditing) {
      textareaRef.current?.focus();
    }
  }, [isEditing]);

  // Expose `startEdit` to the panel chrome so the visible Edit button can
  // enter edit mode. The chrome guards on readOnly itself; we re-check here
  // defensively so a stale handle can never bypass it.
  const register = ctx.registerImperativeHandle;
  useEffect(() => {
    if (!register) return;
    register({
      startEdit: () => {
        if (readOnly) return;
        setIsEditing(true);
      },
    });
    return () => register(null);
  }, [register, readOnly]);

  const commit = () => {
    if (!isEditing) return;
    setIsEditing(false);
    if (draft !== content.body) {
      onChange({ body: draft });
    }
  };

  const cancel = () => {
    setIsEditing(false);
    setDraft(content.body);
  };

  const startEdit = () => {
    if (readOnly) return;
    setIsEditing(true);
  };

  if (isEditing && !readOnly) {
    return (
      <div className="flex h-full w-full flex-col">
        <Textarea
          ref={textareaRef}
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Escape") {
              e.preventDefault();
              cancel();
            }
            // Cmd/Ctrl+Enter commits without losing focus state.
            if ((e.metaKey || e.ctrlKey) && e.key === "Enter") {
              e.preventDefault();
              commit();
            }
          }}
          placeholder={PLACEHOLDER}
          className="h-full w-full resize-none rounded-none border-0 bg-transparent font-mono text-sm shadow-none focus-visible:ring-0 focus-visible:ring-offset-0"
          data-testid="launchpad-markdown-editor"
        />
        <div className="flex items-center justify-between gap-2 border-t border-slate-100 bg-slate-50/50 px-2 py-1.5">
          <span className="text-[11px] text-slate-500">
            <kbd className="rounded border border-slate-200 bg-white px-1 font-mono">Esc</kbd> to cancel,{" "}
            <kbd className="rounded border border-slate-200 bg-white px-1 font-mono">Cmd/Ctrl</kbd>+
            <kbd className="rounded border border-slate-200 bg-white px-1 font-mono">Enter</kbd> to save
          </span>
          <div className="flex items-center gap-1">
            <Button type="button" size="sm" variant="ghost" onClick={cancel} data-testid="launchpad-markdown-cancel">
              Cancel
            </Button>
            <Button type="button" size="sm" onClick={commit} data-testid="launchpad-markdown-save">
              Save
            </Button>
          </div>
        </div>
      </div>
    );
  }

  if (!content.body.trim()) {
    return (
      <button
        type="button"
        onClick={startEdit}
        disabled={readOnly}
        className="flex h-full w-full flex-col items-center justify-center gap-1.5 bg-transparent text-slate-400 transition-colors hover:bg-sky-50/40 hover:text-sky-600 disabled:cursor-default disabled:hover:bg-transparent disabled:hover:text-slate-400"
        data-testid="launchpad-markdown-empty"
      >
        <Pencil className="h-4 w-4" />
        <span className="text-sm">{readOnly ? "Empty panel" : "Click to edit"}</span>
      </button>
    );
  }

  if (widgetPanel && ctx.canvasId) {
    const leading = widgetPanel.leading.trim() ? widgetPanel.leading : null;
    const trailing = widgetPanel.trailing.trim() ? widgetPanel.trailing : null;
    return (
      <div
        className="flex h-full w-full flex-col overflow-hidden px-3 pb-3 pt-2"
        onDoubleClick={startEdit}
        data-testid="launchpad-markdown-view"
        data-fill="true"
      >
        {leading ? (
          <CanvasMarkdown nodeRefs={ctx.nodeRefs} canvasId={ctx.canvasId}>
            {leading}
          </CanvasMarkdown>
        ) : null}
        <div className="min-h-0 flex-1">
          <WidgetBlock body={widgetPanel.widgetBody} canvasId={ctx.canvasId} nodeRefs={ctx.nodeRefs} fill />
        </div>
        {trailing ? (
          <CanvasMarkdown nodeRefs={ctx.nodeRefs} canvasId={ctx.canvasId}>
            {trailing}
          </CanvasMarkdown>
        ) : null}
      </div>
    );
  }

  return (
    <div
      className="h-full w-full overflow-auto px-3 pb-3 pt-2"
      onDoubleClick={startEdit}
      data-testid="launchpad-markdown-view"
      data-fill="false"
    >
      <CanvasMarkdown nodeRefs={ctx.nodeRefs} canvasId={ctx.canvasId}>
        {content.body}
      </CanvasMarkdown>
    </div>
  );
}
