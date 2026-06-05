import { useEffect, useMemo, useRef, useState } from "react";
import { Pencil, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
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

import { useDashboardContext } from "./DashboardContext";
import { useMarkdownVariables } from "./useMarkdownVariables";
import { interpolateMarkdownTemplate } from "./markdownInterpolation";
import { MarkdownBody } from "./MarkdownBody";
import { MarkdownPanelEditor } from "./MarkdownPanelEditor";
import type { MarkdownVariable } from "./panelTypes";

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
  const variables = useMemo(() => readVariables(panel.content), [panel.content]);

  const [editFocus, setEditFocus] = useState<EditFocus>(null);
  const isEditing = editFocus !== null;
  const [draftBody, setDraftBody] = useState(body);
  const [draftTitle, setDraftTitle] = useState(persistedTitle);
  const [draftVariables, setDraftVariables] = useState<MarkdownVariable[]>(variables);
  const titleInputRef = useRef<HTMLInputElement | null>(null);
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);
  const [confirmingDelete, setConfirmingDelete] = useState(false);

  const ctx = useDashboardContext();
  const canvasId = ctx?.canvasId ?? "";
  // Resolve persisted variables for the read-only / display path. The editor
  // re-resolves the draft variables on its own so the preview stays in lockstep
  // with whatever the user just typed.
  const { vars: displayVars } = useMarkdownVariables(canvasId, variables, body);
  const displayTitle = useMemo(() => {
    const interpolated = interpolateMarkdownTemplate(persistedTitle, displayVars).trim();
    return interpolated || persistedTitle.trim() || panel.id;
  }, [persistedTitle, displayVars, panel.id]);

  // Sync drafts from props when we're not editing, so external updates
  // (YAML import, websocket invalidation, etc.) flow into the rendered view.
  useEffect(() => {
    if (!isEditing) {
      setDraftBody(body);
      setDraftTitle(persistedTitle);
      setDraftVariables(variables);
    }
  }, [body, persistedTitle, variables, isEditing]);

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
    const normalizedVars = normalizeDraftVariables(draftVariables);
    const bodyChanged = draftBody !== body;
    const titleChanged = trimmedTitle !== persistedTitle;
    const varsChanged = !variablesEqual(normalizedVars, variables);
    if (!bodyChanged && !titleChanged && !varsChanged) return;
    const nextContent: Record<string, unknown> = { ...(panel.content ?? {}), body: draftBody };
    if (trimmedTitle) nextContent.title = trimmedTitle;
    else delete nextContent.title;
    if (normalizedVars.length > 0) nextContent.variables = normalizedVars;
    else delete nextContent.variables;
    onChange(nextContent);
  };

  const cancel = () => {
    setEditFocus(null);
    setDraftBody(body);
    setDraftTitle(persistedTitle);
    setDraftVariables(variables);
  };

  const startEditing = (focus: EditFocus) => {
    if (readOnly || focus === null) return;
    setEditFocus(focus);
  };

  if (isEditing && !readOnly) {
    return (
      <MarkdownPanelEditor
        panelId={panel.id}
        canvasId={canvasId}
        draftTitle={draftTitle}
        setDraftTitle={setDraftTitle}
        draftBody={draftBody}
        setDraftBody={setDraftBody}
        draftVariables={draftVariables}
        setDraftVariables={setDraftVariables}
        titleInputRef={titleInputRef}
        textareaRef={textareaRef}
        onCancel={cancel}
        onCommit={commit}
      />
    );
  }

  return (
    <MarkdownPanelView
      body={body}
      displayTitle={displayTitle}
      displayVars={displayVars}
      readOnly={readOnly}
      onEditBody={() => startEditing("body")}
      onEditTitle={() => startEditing("title")}
      confirmingDelete={confirmingDelete}
      onRequestDelete={() => setConfirmingDelete(true)}
      onCancelDelete={() => setConfirmingDelete(false)}
      onConfirmDelete={() => {
        setConfirmingDelete(false);
        onDelete();
      }}
    />
  );
}

function MarkdownPanelView({
  body,
  displayTitle,
  displayVars,
  readOnly,
  onEditBody,
  onEditTitle,
  confirmingDelete,
  onRequestDelete,
  onCancelDelete,
  onConfirmDelete,
}: {
  body: string;
  displayTitle: string;
  displayVars: Record<string, unknown>;
  readOnly: boolean;
  onEditBody: () => void;
  onEditTitle: () => void;
  confirmingDelete: boolean;
  onRequestDelete: () => void;
  onCancelDelete: () => void;
  onConfirmDelete: () => void;
}) {
  return (
    <>
      <div className="group/panel relative flex h-full w-full flex-col gap-0 overflow-hidden rounded-lg border border-slate-950/15 bg-white">
        <MarkdownPanelHeader
          displayTitle={displayTitle}
          readOnly={readOnly}
          onEditTitle={onEditTitle}
          onRequestDelete={onRequestDelete}
        />
        {body.trim() ? (
          <div
            className="min-h-0 flex-1 overflow-auto rounded-b-lg bg-white px-4 py-3"
            onDoubleClick={readOnly ? undefined : onEditBody}
            data-testid="dashboard-markdown-view"
          >
            <MarkdownBody body={body} vars={displayVars} />
          </div>
        ) : (
          <button
            type="button"
            onClick={readOnly ? undefined : onEditBody}
            disabled={readOnly}
            className="dashboard-grid-no-drag flex h-full min-h-[6rem] w-full flex-col items-center justify-center gap-1.5 rounded-b-lg bg-white text-[13px] text-gray-500 transition-colors hover:text-gray-800 disabled:cursor-default disabled:hover:text-gray-500"
            data-testid="dashboard-markdown-empty"
          >
            <Pencil className="size-4" />
            <span>{readOnly ? "Empty panel" : "Click to edit"}</span>
          </button>
        )}
      </div>
      <DeleteConfirmDialog open={confirmingDelete} onClose={onCancelDelete} onConfirm={onConfirmDelete} />
    </>
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
      className={cn(
        "flex items-center justify-between rounded-t-lg py-1.5 pl-3 pr-1.5",
        !readOnly && "dashboard-grid-drag-handle cursor-grab active:cursor-grabbing",
      )}
      onDoubleClick={readOnly ? undefined : onEditTitle}
    >
      <span className="truncate text-[13px] font-medium text-slate-700" title={displayTitle}>
        {displayTitle}
      </span>
      {!readOnly ? (
        // The action buttons sit inside the drag-handle header, but
        // react-grid-layout's draggableCancel selector excludes the
        // `dashboard-grid-no-drag` class so clicks land on the buttons
        // instead of starting a drag.
        <div className="dashboard-grid-no-drag -mr-0.5 flex shrink-0 items-center opacity-0 transition-opacity group-hover/panel:opacity-100">
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
            <Pencil className="size-3.5" />
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
            <Trash2 className="size-3.5" />
          </Button>
        </div>
      ) : null}
    </div>
  );
}

/**
 * Read the `variables` array off a panel's persisted content while filtering
 * out malformed entries (defensive against YAML hand-edits). Returns a fresh
 * array so callers can compare with referential identity.
 */
function readVariables(content: DashboardPanel["content"]): MarkdownVariable[] {
  const raw = (content as Record<string, unknown> | undefined)?.variables;
  if (!Array.isArray(raw)) return [];
  const out: MarkdownVariable[] = [];
  for (const item of raw) {
    if (!item || typeof item !== "object") continue;
    const record = item as Record<string, unknown>;
    if (typeof record.name !== "string") continue;
    if (!record.source || typeof record.source !== "object") continue;
    out.push({ name: record.name, source: record.source as MarkdownVariable["source"] });
  }
  return out;
}

/**
 * Normalize a draft variables list for persistence: drop entries with an
 * empty name, strip blank optional fields, and trim free-text inputs. This
 * keeps persisted YAML deterministic so save/reload round-trips are stable.
 */
function normalizeDraftVariables(list: MarkdownVariable[]): MarkdownVariable[] {
  const out: MarkdownVariable[] = [];
  const seen = new Set<string>();
  for (const variable of list) {
    const name = variable?.name?.trim();
    if (!name || seen.has(name) || !variable?.source) continue;
    seen.add(name);
    out.push({ name, source: normalizeVariableSource(variable.source) });
  }
  return out;
}

function normalizeVariableSource(source: MarkdownVariable["source"]): MarkdownVariable["source"] {
  if (source.kind === "memory") {
    const namespace = source.namespace?.trim() ?? "";
    const orderBy = source.orderBy?.trim();
    const matches = (source.matches ?? [])
      .map((match) => ({ field: match?.field?.trim() ?? "", value: match?.value ?? "" }))
      .filter((match) => match.field !== "");
    return {
      kind: "memory",
      namespace,
      ...(orderBy ? { orderBy } : {}),
      ...(source.direction ? { direction: source.direction } : {}),
      ...(matches.length > 0 ? { matches } : {}),
    };
  }
  return { kind: "run", select: source.select };
}

/**
 * Cheap deep equality for two variables arrays — used to skip the
 * `onChange` write on a save with no diffs. Falls back to JSON.stringify
 * since the shape is small and entirely JSON-safe.
 */
function variablesEqual(a: MarkdownVariable[], b: MarkdownVariable[]): boolean {
  return JSON.stringify(a) === JSON.stringify(b);
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
            This panel and its contents will be removed from the console. The content is not recoverable.
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
