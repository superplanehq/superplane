import { useEffect, useRef, useState } from "react";
import { Textarea } from "@/components/ui/textarea";
import { CanvasMarkdown } from "@/ui/Markdown/CanvasMarkdown";
import { Pencil } from "lucide-react";
import type { PanelRenderProps } from "./panelRegistry";

export type MarkdownPanelContent = {
  body: string;
};

const PLACEHOLDER = `# New panel

Write **markdown** here. You can reference canvas nodes with \`@node-name\`.`;

export function MarkdownPanel({ content, readOnly, onChange, ctx }: PanelRenderProps<MarkdownPanelContent>) {
  const [isEditing, setIsEditing] = useState(false);
  const [draft, setDraft] = useState(content.body);
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);

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
          onBlur={commit}
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
          className="h-full w-full resize-none border-none bg-transparent font-mono text-sm focus-visible:ring-0"
          data-testid="launchpad-markdown-editor"
        />
        <div className="flex items-center justify-end gap-2 border-t border-slate-200 px-2 py-1 text-[11px] text-slate-500">
          <span>Esc to cancel, Cmd/Ctrl+Enter to save</span>
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
        className="flex h-full w-full flex-col items-center justify-center gap-1 rounded-md border border-dashed border-slate-300 text-slate-500 hover:border-sky-400 hover:bg-sky-50 disabled:hover:border-slate-300 disabled:hover:bg-transparent"
        data-testid="launchpad-markdown-empty"
      >
        <Pencil className="h-4 w-4" />
        <span className="text-sm">{readOnly ? "Empty panel" : "Click to edit"}</span>
      </button>
    );
  }

  return (
    <div className="h-full w-full overflow-auto p-3" onDoubleClick={startEdit} data-testid="launchpad-markdown-view">
      <CanvasMarkdown nodeRefs={ctx.nodeRefs} canvasId={ctx.canvasId}>
        {content.body}
      </CanvasMarkdown>
    </div>
  );
}
