import { useCallback, useEffect, useRef, useState } from "react";
import { Plus, Loader2, LayoutDashboard, Pencil, Trash2, FileText } from "lucide-react";
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
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { DashboardPanel, DashboardLayoutItem } from "@/hooks/useCanvasData";
import { WorkflowMarkdownPreview } from "../WorkflowMarkdownPreview";

const SAVE_DEBOUNCE_MS = 500;

export interface DashboardViewProps {
  panels: DashboardPanel[];
  layout: DashboardLayoutItem[];
  isLoading: boolean;
  errorMessage?: string;
  readOnly: boolean;
  onChange: (next: { panels: DashboardPanel[]; layout: DashboardLayoutItem[] }) => void;
}

export function DashboardView({ panels, layout, isLoading, errorMessage, readOnly, onChange }: DashboardViewProps) {
  const [localPanels, setLocalPanels] = useState<DashboardPanel[]>(panels);
  const [localLayout, setLocalLayout] = useState<DashboardLayoutItem[]>(layout);
  const lastPropsHashRef = useRef<string>("");

  useEffect(() => {
    const next = JSON.stringify({ panels, layout });
    if (next !== lastPropsHashRef.current) {
      lastPropsHashRef.current = next;
      setLocalPanels(panels);
      setLocalLayout(layout);
    }
  }, [panels, layout]);

  const saveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pendingRef = useRef<{ panels: DashboardPanel[]; layout: DashboardLayoutItem[] } | null>(null);
  const queueSave = useCallback(
    (nextPanels: DashboardPanel[], nextLayout: DashboardLayoutItem[]) => {
      pendingRef.current = { panels: nextPanels, layout: nextLayout };
      if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
      saveTimerRef.current = setTimeout(() => {
        const pending = pendingRef.current;
        if (!pending) return;
        onChange(pending);
        pendingRef.current = null;
      }, SAVE_DEBOUNCE_MS);
    },
    [onChange],
  );

  useEffect(() => {
    return () => {
      if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
      const pending = pendingRef.current;
      if (pending) onChange(pending);
    };
  }, [onChange]);

  const [addDialogOpen, setAddDialogOpen] = useState(false);

  const handleAddPanel = useCallback(
    (name: string) => {
      const slug = name
        .toLowerCase()
        .trim()
        .replace(/\s+/g, "-")
        .replace(/[^a-z0-9-]/g, "")
        .replace(/-+/g, "-")
        .replace(/^-|-$/g, "");
      const id = slug || `panel-${Math.random().toString(36).slice(2, 10)}`;
      const newPanel: DashboardPanel = { id, type: "markdown", content: { body: "" } };
      const maxBottom = localLayout.reduce((acc, item) => Math.max(acc, item.y + item.h), 0);
      const newLayoutItem: DashboardLayoutItem = { i: id, x: 0, y: maxBottom, w: 12, h: 6, minW: 2, minH: 2 };
      const nextPanels = [...localPanels, newPanel];
      const nextLayout = [...localLayout, newLayoutItem];
      setLocalPanels(nextPanels);
      setLocalLayout(nextLayout);
      queueSave(nextPanels, nextLayout);
    },
    [localLayout, localPanels, queueSave],
  );

  const handleDeletePanel = useCallback(
    (id: string) => {
      const nextPanels = localPanels.filter((p) => p.id !== id);
      const nextLayout = localLayout.filter((l) => l.i !== id);
      setLocalPanels(nextPanels);
      setLocalLayout(nextLayout);
      queueSave(nextPanels, nextLayout);
    },
    [localLayout, localPanels, queueSave],
  );

  const handlePanelContentChange = useCallback(
    (id: string, content: Record<string, unknown>) => {
      const nextPanels = localPanels.map((p) => (p.id === id ? { ...p, content } : p));
      setLocalPanels(nextPanels);
      queueSave(nextPanels, localLayout);
    },
    [localLayout, localPanels, queueSave],
  );

  if (errorMessage) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-2 p-8 text-sm text-red-600">
        <p className="font-medium">Failed to load dashboard</p>
        <p className="text-slate-500">{errorMessage}</p>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <Loader2 className="h-5 w-5 animate-spin text-slate-400" />
      </div>
    );
  }

  if (localPanels.length === 0) {
    return (
      <>
        <EmptyState readOnly={readOnly} onAdd={() => setAddDialogOpen(true)} />
        <AddPanelDialog
          open={addDialogOpen}
          onConfirm={(name: string) => {
            handleAddPanel(name);
            setAddDialogOpen(false);
          }}
          onCancel={() => setAddDialogOpen(false)}
        />
      </>
    );
  }

  return (
    <div className="flex h-full w-full flex-col overflow-auto">
      {!readOnly ? (
        <div className="flex items-center justify-end px-4 pt-3 pb-1">
          <Button size="sm" variant="default" onClick={() => setAddDialogOpen(true)} data-testid="dashboard-add-panel">
            <Plus className="mr-1 h-3.5 w-3.5" />
            Add panel
          </Button>
        </div>
      ) : null}
      <div className="flex flex-col gap-3 px-4 py-3">
        {localPanels.map((panel) => (
          <MarkdownPanelCard
            key={panel.id}
            panel={panel}
            readOnly={readOnly}
            onDelete={() => handleDeletePanel(panel.id)}
            onChange={(content) => handlePanelContentChange(panel.id, content)}
          />
        ))}
      </div>
      <AddPanelDialog
        open={addDialogOpen}
        onConfirm={(name: string) => {
          handleAddPanel(name);
          setAddDialogOpen(false);
        }}
        onCancel={() => setAddDialogOpen(false)}
      />
    </div>
  );
}

function MarkdownPanelCard({
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
  const body = typeof panel.content?.body === "string" ? (panel.content.body as string) : "";
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
      <div className="flex flex-col gap-0 overflow-hidden rounded-lg border border-slate-200 bg-white shadow-sm">
        <div className="flex items-center justify-between border-b border-slate-100 bg-slate-50/80 px-3 py-1.5">
          <span className="text-xs font-medium text-slate-500">{panel.id}</span>
        </div>
        <Textarea
          ref={textareaRef}
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Escape") {
              e.preventDefault();
              cancel();
            }
            if ((e.metaKey || e.ctrlKey) && e.key === "Enter") {
              e.preventDefault();
              commit();
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
            <Button type="button" size="sm" variant="ghost" onClick={cancel}>
              Cancel
            </Button>
            <Button type="button" size="sm" onClick={commit} data-testid="dashboard-markdown-save">
              Save
            </Button>
          </div>
        </div>
      </div>
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

function EmptyState({ readOnly, onAdd }: { readOnly: boolean; onAdd: () => void }) {
  return (
    <div className="flex flex-1 items-center justify-center p-8" data-testid="dashboard-empty-state">
      <div className="flex w-full max-w-lg flex-col items-center gap-5 rounded-xl border border-dashed border-slate-300 bg-white/70 px-8 py-10 text-center shadow-sm">
        <div className="flex h-14 w-14 items-center justify-center rounded-full bg-slate-100">
          <LayoutDashboard className="h-7 w-7 text-slate-500" />
        </div>
        <div className="flex flex-col gap-1.5">
          <h3 className="text-lg font-semibold text-slate-800">Build your dashboard</h3>
          <p className="mx-auto max-w-md text-sm leading-relaxed text-slate-500">
            Add markdown panels to surface important docs, links, and notes for this canvas.
          </p>
        </div>
        {!readOnly ? (
          <Button variant="default" onClick={onAdd} data-testid="dashboard-add-first-panel">
            <Plus className="mr-1.5 h-4 w-4" />
            Add panel
          </Button>
        ) : null}
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

function AddPanelDialog({
  open,
  onConfirm,
  onCancel,
}: {
  open: boolean;
  onConfirm: (name: string) => void;
  onCancel: () => void;
}) {
  const [name, setName] = useState("");
  const slug = name
    .toLowerCase()
    .trim()
    .replace(/\s+/g, "-")
    .replace(/[^a-z0-9-]/g, "")
    .replace(/-+/g, "-")
    .replace(/^-|-$/g, "");
  const isValid = slug.length > 0;

  return (
    <Dialog
      open={open}
      onOpenChange={(isOpen) => {
        if (!isOpen) {
          setName("");
          onCancel();
        }
      }}
    >
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Add panel</DialogTitle>
          <DialogDescription>Give your panel a name. This will be used as its identifier.</DialogDescription>
        </DialogHeader>
        <div className="space-y-3 py-2">
          <div className="space-y-1.5">
            <Label htmlFor="panel-name">Name</Label>
            <Input
              id="panel-name"
              placeholder="e.g. Pipeline Status"
              value={name}
              onChange={(e) => setName(e.target.value)}
              onKeyDown={(e: React.KeyboardEvent<HTMLInputElement>) => {
                if (e.key === "Enter" && isValid) {
                  onConfirm(name.trim());
                  setName("");
                }
              }}
              autoFocus
              data-testid="add-panel-name-input"
            />
          </div>
          {name.trim() ? (
            <p className="text-xs text-slate-500">
              ID: <code className="rounded bg-slate-100 px-1 py-0.5">{slug || "—"}</code>
            </p>
          ) : null}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onCancel}>
            Cancel
          </Button>
          <Button
            onClick={() => {
              if (isValid) {
                onConfirm(name.trim());
                setName("");
              }
            }}
            disabled={!isValid}
            data-testid="add-panel-confirm"
          >
            <FileText className="mr-1.5 h-3.5 w-3.5" />
            Add
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
