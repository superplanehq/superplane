import { useEffect, useRef, useState, type RefObject } from "react";
import { Pencil, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { DashboardPanel } from "@/hooks/useCanvasData";
import { WorkflowMarkdownPreview } from "../WorkflowMarkdownPreview";

export function MarkdownPanelCard({
  panel,
  readOnly,
  onDelete,
  onChange,
}: {
  panel: DashboardPanel;
  readOnly: boolean;
  onDelete: () => void;
  onChange: (content: Record<string, unknown>) => void;
}) {
  const [isEditing, setIsEditing] = useState(false);
  const body = typeof panel.content?.body === "string" ? panel.content.body : "";
  const [draft, setDraft] = useState(body);
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);
  const [confirmingDelete, setConfirmingDelete] = useState(false);

  useEffect(() => {
    if (!isEditing) setDraft(body);
  }, [body, isEditing]);

  useEffect(() => {
    if (isEditing) textareaRef.current?.focus();
  }, [isEditing]);

  const commit = () => {
    if (!isEditing) return;
    setIsEditing(false);
    if (draft !== body) onChange({ body: draft });
  };

  const cancel = () => {
    setIsEditing(false);
    setDraft(body);
  };

  if (isEditing && !readOnly) {
    return (
      <MarkdownPanelEditor
        panelId={panel.id}
        draft={draft}
        setDraft={setDraft}
        textareaRef={textareaRef}
        onCancel={cancel}
        onCommit={commit}
      />
    );
  }

  return (
    <>
      <div className="group/panel relative flex flex-col gap-0 overflow-hidden rounded-lg border border-slate-200 bg-white shadow-sm">
        <div className="flex items-center justify-between border-b border-slate-100 bg-slate-50/80 px-3 py-1.5">
          <span className="text-xs font-medium text-slate-500">{panel.id}</span>
          {!readOnly ? (
            <div className="flex items-center gap-0.5 opacity-0 transition-opacity group-hover/panel:opacity-100">
              <Button
                type="button"
                size="icon"
                variant="ghost"
                onClick={() => setIsEditing(true)}
                aria-label="Edit panel"
                className="h-6 w-6 text-slate-500 hover:text-slate-700"
                data-testid="dashboard-edit-panel"
              >
                <Pencil className="h-3.5 w-3.5" />
              </Button>
              <Button
                type="button"
                size="icon"
                variant="ghost"
                onClick={() => setConfirmingDelete(true)}
                aria-label="Delete panel"
                className="h-6 w-6 text-slate-500 hover:bg-red-50 hover:text-red-600"
                data-testid="dashboard-delete-panel"
              >
                <Trash2 className="h-3.5 w-3.5" />
              </Button>
            </div>
          ) : null}
        </div>
        {body.trim() ? (
          <div
            className="px-4 py-3"
            onDoubleClick={readOnly ? undefined : () => setIsEditing(true)}
            data-testid="dashboard-markdown-view"
          >
            <WorkflowMarkdownPreview content={body} />
          </div>
        ) : (
          <button
            type="button"
            onClick={readOnly ? undefined : () => setIsEditing(true)}
            disabled={readOnly}
            className="flex h-24 w-full flex-col items-center justify-center gap-1.5 bg-transparent text-slate-400 transition-colors hover:bg-sky-50/40 hover:text-sky-600 disabled:cursor-default disabled:hover:bg-transparent disabled:hover:text-slate-400"
            data-testid="dashboard-markdown-empty"
          >
            <Pencil className="h-4 w-4" />
            <span className="text-sm">{readOnly ? "Empty panel" : "Click to edit"}</span>
          </button>
        )}
      </div>
      <DeleteConfirmDialog
        open={confirmingDelete}
        onClose={() => setConfirmingDelete(false)}
        onConfirm={() => {
          setConfirmingDelete(false);
          onDelete();
        }}
      />
    </>
  );
}

function MarkdownPanelEditor({
  panelId,
  draft,
  setDraft,
  textareaRef,
  onCancel,
  onCommit,
}: {
  panelId: string;
  draft: string;
  setDraft: (value: string) => void;
  textareaRef: RefObject<HTMLTextAreaElement | null>;
  onCancel: () => void;
  onCommit: () => void;
}) {
  return (
    <div className="flex flex-col gap-0 overflow-hidden rounded-lg border border-slate-200 bg-white shadow-sm">
      <div className="flex items-center justify-between border-b border-slate-100 bg-slate-50/80 px-3 py-1.5">
        <span className="text-xs font-medium text-slate-500">{panelId}</span>
      </div>
      <Textarea
        ref={textareaRef}
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Escape") {
            e.preventDefault();
            onCancel();
          }
          if ((e.metaKey || e.ctrlKey) && e.key === "Enter") {
            e.preventDefault();
            onCommit();
          }
        }}
        placeholder="Write **markdown** here..."
        className="min-h-[120px] resize-none rounded-none border-0 bg-transparent font-mono text-sm shadow-none focus-visible:ring-0 focus-visible:ring-offset-0"
        data-testid="dashboard-markdown-editor"
      />
      <div className="flex items-center justify-between gap-2 border-t border-slate-100 bg-slate-50/50 px-3 py-1.5">
        <span className="text-[11px] text-slate-500">
          <kbd className="rounded border border-slate-200 bg-white px-1 font-mono">Esc</kbd> cancel &middot;{" "}
          <kbd className="rounded border border-slate-200 bg-white px-1 font-mono">Cmd+Enter</kbd> save
        </span>
        <div className="flex items-center gap-1">
          <Button type="button" size="sm" variant="ghost" onClick={onCancel}>
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

function DeleteConfirmDialog({
  open,
  onClose,
  onConfirm,
}: {
  open: boolean;
  onClose: () => void;
  onConfirm: () => void;
}) {
  return (
    <Dialog open={open} onOpenChange={(next) => (next ? null : onClose())}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete this panel?</DialogTitle>
          <DialogDescription>
            This panel and its contents will be removed from the dashboard. The content is not recoverable.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button type="button" variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button type="button" variant="destructive" onClick={onConfirm} data-testid="dashboard-delete-confirm">
            Delete panel
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
