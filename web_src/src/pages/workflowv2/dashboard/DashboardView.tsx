import { useCallback, useState } from "react";
import { Plus, Loader2, LayoutGrid, AtSign, BarChart3, Play } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { DashboardPanel, DashboardLayoutItem } from "@/hooks/useCanvasData";

import { MarkdownPanelCard } from "./MarkdownPanelCard";
import { useDashboardPanelState } from "./useDashboardPanelState";

export interface DashboardViewProps {
  panels: DashboardPanel[];
  layout: DashboardLayoutItem[];
  isLoading: boolean;
  errorMessage?: string;
  readOnly: boolean;
  onChange: (next: { panels: DashboardPanel[]; layout: DashboardLayoutItem[] }) => void;
  /** When provided with `onAddPanelDialogOpenChange`, the add-panel dialog is controlled by the parent (e.g. header). */
  addPanelDialogOpen?: boolean;
  onAddPanelDialogOpenChange?: (open: boolean) => void;
}

export function DashboardView({
  panels,
  layout,
  isLoading,
  errorMessage,
  readOnly,
  onChange,
  addPanelDialogOpen: addPanelDialogOpenProp,
  onAddPanelDialogOpenChange,
}: DashboardViewProps) {
  const [internalAddPanelOpen, setInternalAddPanelOpen] = useState(false);
  const isAddPanelControlled = onAddPanelDialogOpenChange != null;
  const addPanelOpen = isAddPanelControlled ? Boolean(addPanelDialogOpenProp) : internalAddPanelOpen;
  const setAddPanelOpen = useCallback(
    (next: boolean) => {
      if (isAddPanelControlled) onAddPanelDialogOpenChange!(next);
      else setInternalAddPanelOpen(next);
    },
    [isAddPanelControlled, onAddPanelDialogOpenChange],
  );

  const { localPanels, handleAddPanel, handleDeletePanel, handlePanelContentChange } = useDashboardPanelState(
    panels,
    layout,
    onChange,
  );

  const confirmAddPanel = (name: string) => {
    handleAddPanel(name);
    setAddPanelOpen(false);
  };

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
        <EmptyState readOnly={readOnly} onAdd={() => setAddPanelOpen(true)} />
        <AddPanelDialog open={addPanelOpen} onConfirm={confirmAddPanel} onCancel={() => setAddPanelOpen(false)} />
      </>
    );
  }

  return (
    <div className="flex h-full w-full flex-col overflow-auto">
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
      <AddPanelDialog open={addPanelOpen} onConfirm={confirmAddPanel} onCancel={() => setAddPanelOpen(false)} />
    </div>
  );
}

function EmptyState({ readOnly, onAdd }: { readOnly: boolean; onAdd: () => void }) {
  return (
    <div className="flex flex-1 items-center justify-center p-6 sm:p-8" data-testid="dashboard-empty-state">
      <div className="flex w-full max-w-4xl flex-col items-center rounded-2xl border border-dashed border-slate-300 bg-white px-6 py-10 shadow-sm sm:px-10">
        <div className="flex flex-col items-center text-center">
          <div className="flex h-14 w-14 items-center justify-center rounded-full bg-slate-100">
            <LayoutGrid className="h-7 w-7 text-slate-600" />
          </div>
          <h3 className="mt-5 text-xl font-semibold tracking-tight text-slate-900">Build your dashboard</h3>
          <p className="mx-auto mt-2 max-w-2xl text-sm leading-relaxed text-slate-500">
            Dashboard panels surface the most important docs, links, and live data for this canvas. Drag panels into
            place and resize from the bottom-right corner to lay them out the way your team works.
          </p>
        </div>

        <div className="mt-8 grid w-full gap-4 sm:grid-cols-3">
          <div className="rounded-xl border border-slate-200 bg-slate-50/50 p-4 text-left shadow-sm">
            <AtSign className="h-5 w-5 text-sky-600" aria-hidden />
            <h4 className="mt-3 text-sm font-semibold text-slate-900">Reference nodes</h4>
            <p className="mt-1 text-xs leading-relaxed text-slate-500">Drop in @my-node chips that show live status.</p>
          </div>
          <div className="rounded-xl border border-slate-200 bg-slate-50/50 p-4 text-left shadow-sm">
            <BarChart3 className="h-5 w-5 text-sky-600" aria-hidden />
            <h4 className="mt-3 text-sm font-semibold text-slate-900">Show live data</h4>
            <p className="mt-1 text-xs leading-relaxed text-slate-500">
              Embed canvas memory or executions with widget blocks.
            </p>
          </div>
          <div className="rounded-xl border border-slate-200 bg-slate-50/50 p-4 text-left shadow-sm">
            <Play className="h-5 w-5 text-sky-600" aria-hidden />
            <h4 className="mt-3 text-sm font-semibold text-slate-900">Trigger runs</h4>
            <p className="mt-1 text-xs leading-relaxed text-slate-500">
              Add Run buttons for manual triggers right in markdown.
            </p>
          </div>
        </div>

        {!readOnly ? (
          <Button variant="default" className="mt-10" onClick={onAdd} data-testid="dashboard-add-first-panel">
            <Plus className="mr-1.5 h-4 w-4" />
            Add panel
          </Button>
        ) : null}
      </div>
    </div>
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
        </DialogHeader>
        <div className="space-y-3 py-2">
          <div className="space-y-1.5">
            <Label htmlFor="panel-name">Name</Label>
            <Input
              id="panel-name"
              placeholder="Panel name"
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
            Add
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
