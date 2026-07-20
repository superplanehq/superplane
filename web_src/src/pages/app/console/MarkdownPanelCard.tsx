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
import type { ConsolePanel } from "@/hooks/useCanvasData";

import { useConsoleContext } from "./ConsoleContext";
import { CONSOLE_PANEL_BODY_SURFACE, CONSOLE_PANEL_SHELL_SURFACE } from "./consolePanelStyles";
import { useMarkdownVariables } from "./useMarkdownVariables";
import {
  interpolateMarkdownTemplate,
  markdownTemplateHasExpressions,
  markdownTextIsLoading,
} from "./markdownInterpolation";
import { MarkdownBody, MarkdownBodyLoading } from "./MarkdownBody";
import { MarkdownPanelEditor } from "./MarkdownPanelEditor";
import {
  normalizeRunStatuses,
  normalizeRunTriggers,
  validateMarkdownVariables,
  type MarkdownVariable,
} from "./panelTypes";

/**
 * Stable empty list passed to `useMarkdownVariables` while the panel is being
 * edited. The editor mounts its own hook over the draft variables, so the
 * read-path hook here is disabled to avoid running the same memory / run /
 * execution queries twice for a single panel.
 */
const EMPTY_VARIABLES: MarkdownVariable[] = [];

/**
 * Which field auto-focuses when the user enters edit mode. Driven by which
 * affordance the user activated:
 *  - pencil icon → title input
 *  - double-click on the body / "click to edit" empty state → body textarea
 */
type EditFocus = "title" | "body" | null;

/**
 * Resolve the persisted variables for the read-only / display path and derive
 * the display title plus the body loading gate.
 *
 * The editor re-resolves the draft variables on its own so its preview stays
 * in lockstep with whatever the user just typed; while editing this hook is
 * disabled (empty list, no side-load text) so the editor's hook is the single
 * owner of the queries for this panel. Title + body are both interpolated, so
 * the combined text drives the run-node side-load gate.
 */
function useMarkdownDisplay({
  panelId,
  body,
  persistedTitle,
  variables,
  isEditing,
}: {
  panelId: string;
  body: string;
  persistedTitle: string;
  variables: MarkdownVariable[];
  isEditing: boolean;
}) {
  const ctx = useConsoleContext();
  const canvasId = ctx?.canvasId ?? "";
  const textForSideload = useMemo(() => `${persistedTitle}\n${body}`, [persistedTitle, body]);
  const { vars, baseLoading, sideloadLoading, searchingNames } = useMarkdownVariables(
    canvasId,
    isEditing ? EMPTY_VARIABLES : variables,
    isEditing ? "" : textForSideload,
  );

  // While the backing variable queries (including the per-run execution
  // side-load behind `{{ run.$["Node"]... }}`) are still in flight, the var map
  // is only partially resolved. Interpolating against it now would render those
  // references as empty fields and then flash to the real values once the
  // side-load settles. Mirror the table run widget, which treats execution
  // side-load as part of initial loading, and hold a loading state instead.
  // Gating is per-text and per-phase: text that doesn't reference a run node
  // resolves without the execution side-load, so it isn't held on that phase.
  const titleLoading = markdownTextIsLoading(persistedTitle, baseLoading, sideloadLoading, searchingNames);
  const bodyLoading = markdownTextIsLoading(body, baseLoading, sideloadLoading, searchingNames);

  const displayTitle = useMemo(() => {
    // A templated title can't be shown verbatim (it'd leak raw `{{ }}` syntax)
    // and can't be interpolated yet, so fall back to the stable panel id while
    // the variables it actually depends on load.
    if (titleLoading) return panelId;
    const interpolated = interpolateMarkdownTemplate(persistedTitle, vars).trim();
    if (interpolated) return interpolated;
    // A templated title that resolves to an empty string must not fall back to
    // its raw source (that would leak the unparsed `{{ }}` syntax). Only a
    // static title is safe to show verbatim; otherwise use the stable panel id.
    if (markdownTemplateHasExpressions(persistedTitle)) return panelId;
    return persistedTitle.trim() || panelId;
  }, [titleLoading, persistedTitle, vars, panelId]);

  return { canvasId, displayVars: vars, bodyLoading, displayTitle };
}

interface MarkdownPanelCardProps {
  panel: ConsolePanel;
  readOnly: boolean;
  onDelete: () => void;
  onChange: (content: Record<string, unknown>) => void;
  onEditingChange?: (editing: boolean) => void;
}

export function MarkdownPanelCard({ panel, readOnly, onDelete, onChange, onEditingChange }: MarkdownPanelCardProps) {
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
  // Surfaced when an in-card save is blocked by the shared variable validator
  // (e.g. invalid name, empty namespace); cleared once the author edits again.
  const [saveError, setSaveError] = useState<string | null>(null);

  const { canvasId, displayVars, bodyLoading, displayTitle } = useMarkdownDisplay({
    panelId: panel.id,
    body,
    persistedTitle,
    variables,
    isEditing,
  });

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
    const trimmedTitle = draftTitle.trim();
    const normalizedVars = normalizeDraftVariables(draftVariables);
    // Guard the in-card save with the same validator the YAML / dialog editor
    // and the backend use, so we never persist content the API would reject.
    // Keep the editor open with the message so the author can fix it.
    const validationError = validateMarkdownVariables(normalizedVars);
    if (validationError) {
      setSaveError(validationError);
      return;
    }
    setSaveError(null);
    setEditFocus(null);
    onEditingChange?.(false);
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
    setSaveError(null);
    onEditingChange?.(false);
    setDraftBody(body);
    setDraftTitle(persistedTitle);
    setDraftVariables(variables);
  };

  const startEditing = (focus: EditFocus) => {
    if (readOnly || focus === null) return;
    setSaveError(null);
    setEditFocus(focus);
    onEditingChange?.(true);
  };

  // Editing the variables clears a stale save error so the blocked-save banner
  // doesn't linger after the author addresses it.
  const updateDraftVariables = (next: MarkdownVariable[]) => {
    if (saveError) setSaveError(null);
    setDraftVariables(next);
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
        setDraftVariables={updateDraftVariables}
        titleInputRef={titleInputRef}
        textareaRef={textareaRef}
        saveError={saveError}
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
      bodyLoading={bodyLoading}
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
  bodyLoading,
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
  bodyLoading: boolean;
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
      <div
        className={cn(
          "group/panel relative flex h-full w-full flex-col gap-0 overflow-hidden rounded-lg border border-slate-950/15 bg-white dark:border-gray-700/70",
          CONSOLE_PANEL_SHELL_SURFACE,
        )}
      >
        <MarkdownPanelHeader
          displayTitle={displayTitle}
          readOnly={readOnly}
          onEditTitle={onEditTitle}
          onRequestDelete={onRequestDelete}
        />
        {body.trim() ? (
          <div
            className={cn("min-h-0 flex-1 overflow-auto rounded-b-lg bg-white px-4 py-3", CONSOLE_PANEL_BODY_SURFACE)}
            onDoubleClick={readOnly ? undefined : onEditBody}
            data-testid="console-markdown-view"
          >
            {bodyLoading ? <MarkdownBodyLoading /> : <MarkdownBody body={body} vars={displayVars} />}
          </div>
        ) : (
          <button
            type="button"
            onClick={readOnly ? undefined : onEditBody}
            disabled={readOnly}
            className={cn(
              "console-grid-no-drag flex h-full min-h-[6rem] w-full flex-col items-center justify-center gap-1.5 rounded-b-lg bg-white text-[13px] text-gray-500 transition-colors hover:text-gray-800 disabled:cursor-default disabled:hover:text-gray-500 dark:text-gray-400 dark:hover:text-gray-200 dark:disabled:hover:text-gray-400",
              CONSOLE_PANEL_BODY_SURFACE,
            )}
            data-testid="console-markdown-empty"
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
        !readOnly && "console-grid-drag-handle cursor-grab active:cursor-grabbing",
      )}
      onDoubleClick={readOnly ? undefined : onEditTitle}
    >
      <span className="truncate text-[13px] font-medium text-slate-700 dark:text-gray-300" title={displayTitle}>
        {displayTitle}
      </span>
      {!readOnly ? (
        // The action buttons sit inside the drag-handle header, but
        // react-grid-layout's draggableCancel selector excludes the
        // `console-grid-no-drag` class so clicks land on the buttons
        // instead of starting a drag.
        <div className="console-grid-no-drag -mr-0.5 flex shrink-0 items-center opacity-0 transition-opacity group-hover/panel:opacity-100">
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
            className="h-6 w-6 cursor-pointer text-slate-500 hover:text-slate-700 dark:text-gray-400 dark:hover:text-gray-200"
            data-testid="console-edit-panel"
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
            className="h-6 w-6 cursor-pointer text-slate-500 hover:bg-red-50 hover:text-red-600 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-red-400"
            data-testid="console-delete-panel"
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
function readVariables(content: ConsolePanel["content"]): MarkdownVariable[] {
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
    // `mode: single` is the implicit default, so only persist `mode`/`limit`
    // when the author opted into list mode. Dropping them otherwise keeps the
    // YAML minimal and round-trips identically.
    const isList = source.mode === "list";
    const hasLimit = isList && typeof source.limit === "number" && source.limit > 0;
    return {
      kind: "memory",
      namespace,
      ...(orderBy ? { orderBy } : {}),
      ...(source.direction ? { direction: source.direction } : {}),
      ...(matches.length > 0 ? { matches } : {}),
      ...(isList ? { mode: "list" as const } : {}),
      ...(hasLimit ? { limit: source.limit } : {}),
    };
  }
  const statuses = normalizeRunStatuses(source.statuses);
  const triggers = normalizeRunTriggers(source.triggers);
  return {
    kind: "run",
    select: source.select,
    ...(statuses ? { statuses } : {}),
    ...(triggers ? { triggers } : {}),
  };
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
      <DialogContent className="dark:border-gray-700/70 dark:bg-gray-900">
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
          <Button type="button" variant="destructive" onClick={onConfirm} data-testid="console-delete-confirm">
            Delete panel
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
