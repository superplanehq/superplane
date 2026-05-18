import { useEffect, useMemo, useRef, useState, type RefObject } from "react";
import { Pencil, Trash2 } from "lucide-react";
import ReactMarkdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { cn } from "@/lib/utils";
import type { DashboardPanel } from "@/hooks/useCanvasData";

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
  "[&_td]:border [&_td]:border-slate-100 [&_td]:px-2 [&_td]:py-1";

/**
 * Which field auto-focuses when the user enters edit mode. Driven by which
 * affordance the user activated:
 *  - pencil icon → title input
 *  - double-click on the body / "click to edit" empty state → body textarea
 */
type EditFocus = "title" | "body" | null;

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
  const body = typeof panel.content?.body === "string" ? panel.content.body : "";
  const persistedTitle = typeof panel.content?.title === "string" ? panel.content.title : "";
  const displayTitle = persistedTitle.trim() || panel.id;

  const [editFocus, setEditFocus] = useState<EditFocus>(null);
  const isEditing = editFocus !== null;
  const [draftBody, setDraftBody] = useState(body);
  const [draftTitle, setDraftTitle] = useState(persistedTitle);
  const titleInputRef = useRef<HTMLInputElement | null>(null);
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);
  const [confirmingDelete, setConfirmingDelete] = useState(false);

  // Sync drafts from props when we're not editing, so external updates
  // (YAML import, websocket invalidation, etc.) flow into the rendered view.
  useEffect(() => {
    if (!isEditing) {
      setDraftBody(body);
      setDraftTitle(persistedTitle);
    }
  }, [body, persistedTitle, isEditing]);

  useEffect(() => {
    if (editFocus === "title") {
      titleInputRef.current?.focus();
      titleInputRef.current?.select();
    } else if (editFocus === "body") {
      textareaRef.current?.focus();
    }
  }, [editFocus]);

  const commit = () => {
    if (!isEditing) return;
    setEditFocus(null);
    const trimmedTitle = draftTitle.trim();
    const bodyChanged = draftBody !== body;
    const titleChanged = trimmedTitle !== persistedTitle;
    if (!bodyChanged && !titleChanged) return;
    const nextContent: Record<string, unknown> = { ...(panel.content ?? {}), body: draftBody };
    if (trimmedTitle) nextContent.title = trimmedTitle;
    else delete nextContent.title;
    onChange(nextContent);
  };

  const cancel = () => {
    setEditFocus(null);
    setDraftBody(body);
    setDraftTitle(persistedTitle);
  };

  const startEditing = (focus: EditFocus) => {
    if (readOnly || focus === null) return;
    setEditFocus(focus);
  };

  if (isEditing && !readOnly) {
    return (
      <MarkdownPanelEditor
        panelId={panel.id}
        draftTitle={draftTitle}
        setDraftTitle={setDraftTitle}
        draftBody={draftBody}
        setDraftBody={setDraftBody}
        titleInputRef={titleInputRef}
        textareaRef={textareaRef}
        onCancel={cancel}
        onCommit={commit}
      />
    );
  }

  return (
    <>
      <div className="group/panel relative flex h-full w-full flex-col gap-0 overflow-hidden rounded-lg border border-slate-200 bg-white shadow-sm">
        <MarkdownPanelHeader
          displayTitle={displayTitle}
          readOnly={readOnly}
          onEditTitle={() => startEditing("title")}
          onRequestDelete={() => setConfirmingDelete(true)}
        />
        {body.trim() ? (
          <div
            className="min-h-0 flex-1 overflow-auto px-4 py-3"
            onDoubleClick={readOnly ? undefined : () => startEditing("body")}
            data-testid="dashboard-markdown-view"
          >
            <MarkdownBody body={body} />
          </div>
        ) : (
          <button
            type="button"
            onClick={readOnly ? undefined : () => startEditing("body")}
            disabled={readOnly}
            className="dashboard-grid-no-drag flex h-full min-h-[6rem] w-full flex-col items-center justify-center gap-1.5 bg-transparent text-slate-400 transition-colors hover:bg-sky-50/40 hover:text-sky-600 disabled:cursor-default disabled:hover:bg-transparent disabled:hover:text-slate-400"
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

function MarkdownBody({ body }: { body: string }) {
  const normalized = useMemo(() => body.replace(/\r\n/g, "\n").trim(), [body]);
  if (!normalized) return null;
  return (
    <div className={cn(MARKDOWN_CLASSES)} data-testid="dashboard-markdown">
      <ReactMarkdown remarkPlugins={[remarkGfm, remarkBreaks]}>{normalized}</ReactMarkdown>
    </div>
  );
}

function MarkdownPanelHeader({
  displayTitle,
  readOnly,
  onEditTitle,
  onRequestDelete,
}: {
  displayTitle: string;
  readOnly: boolean;
  onEditTitle: () => void;
  onRequestDelete: () => void;
}) {
  return (
    <div
      className={
        "flex items-center justify-between border-b border-slate-100 bg-slate-50/80 px-3 py-1.5 " +
        (readOnly ? "" : "dashboard-grid-drag-handle cursor-grab active:cursor-grabbing")
      }
      onDoubleClick={readOnly ? undefined : onEditTitle}
    >
      <span className="truncate text-xs font-medium text-slate-700" title={displayTitle}>
        {displayTitle}
      </span>
      {!readOnly ? (
        // The action buttons sit inside the drag-handle header, but
        // react-grid-layout's draggableCancel selector excludes the
        // `dashboard-grid-no-drag` class so clicks land on the buttons
        // instead of starting a drag.
        <div className="dashboard-grid-no-drag flex items-center gap-0.5 opacity-0 transition-opacity group-hover/panel:opacity-100">
          <Button
            type="button"
            size="icon"
            variant="ghost"
            onClick={(e) => {
              e.stopPropagation();
              onEditTitle();
            }}
            onMouseDown={(e) => e.stopPropagation()}
            onPointerDown={(e) => e.stopPropagation()}
            aria-label="Edit panel"
            className="h-6 w-6 cursor-pointer text-slate-500 hover:text-slate-700"
            data-testid="dashboard-edit-panel"
          >
            <Pencil className="h-3.5 w-3.5" />
          </Button>
          <Button
            type="button"
            size="icon"
            variant="ghost"
            onClick={(e) => {
              e.stopPropagation();
              onRequestDelete();
            }}
            onMouseDown={(e) => e.stopPropagation()}
            onPointerDown={(e) => e.stopPropagation()}
            aria-label="Delete panel"
            className="h-6 w-6 cursor-pointer text-slate-500 hover:bg-red-50 hover:text-red-600"
            data-testid="dashboard-delete-panel"
          >
            <Trash2 className="h-3.5 w-3.5" />
          </Button>
        </div>
      ) : null}
    </div>
  );
}

function MarkdownPanelEditor({
  panelId,
  draftTitle,
  setDraftTitle,
  draftBody,
  setDraftBody,
  titleInputRef,
  textareaRef,
  onCancel,
  onCommit,
}: {
  panelId: string;
  draftTitle: string;
  setDraftTitle: (value: string) => void;
  draftBody: string;
  setDraftBody: (value: string) => void;
  titleInputRef: RefObject<HTMLInputElement | null>;
  textareaRef: RefObject<HTMLTextAreaElement | null>;
  onCancel: () => void;
  onCommit: () => void;
}) {
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

  return (
    <div className="flex h-full w-full flex-col gap-0 overflow-hidden rounded-lg border border-slate-200 bg-white shadow-sm">
      <div className="flex items-center gap-2 border-b border-slate-100 bg-slate-50/80 px-2 py-1">
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
      <Textarea
        ref={textareaRef}
        value={draftBody}
        onChange={(e) => setDraftBody(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder="Write **markdown** here..."
        className="min-h-[120px] flex-1 resize-none rounded-none border-0 bg-transparent font-mono text-sm shadow-none focus-visible:ring-0 focus-visible:ring-offset-0"
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
