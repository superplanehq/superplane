import { useCallback, useMemo, useState } from "react";
import {
  CodeXml,
  FileText,
  Gauge,
  Hash,
  LayoutGrid,
  LineChart,
  Loader2,
  Network,
  Plus,
  Table2,
  Workflow,
} from "lucide-react";

import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { ConsolePanel, ConsoleLayoutItem } from "@/hooks/useCanvasData";
import type { DraftConsoleDiffItem, DraftConsoleDiffSummary } from "../draftConsoleDiff";

import { ConsoleGrid } from "./ConsoleGrid";
import { CONSOLE_PANEL_SHELL_SURFACE } from "./consolePanelStyles";
import { useConsolePanelState } from "./useConsolePanelState";
import { CREATABLE_PANEL_TYPES, PANEL_TYPE_META, type PanelType } from "./panelTypes";

export interface ConsoleViewProps {
  panels: ConsolePanel[];
  layout: ConsoleLayoutItem[];
  isLoading: boolean;
  errorMessage?: string;
  readOnly: boolean;
  onChange: (next: { panels: ConsolePanel[]; layout: ConsoleLayoutItem[] }) => void;
  onEffectiveChange?: (next: { panels: ConsolePanel[]; layout: ConsoleLayoutItem[] }) => void;
  /** When provided with `onAddPanelDialogOpenChange`, the add-panel dialog is controlled by the parent (e.g. header). */
  addPanelDialogOpen?: boolean;
  onAddPanelDialogOpenChange?: (open: boolean) => void;
  visualDiff?: {
    enabled: boolean;
    summary?: DraftConsoleDiffSummary;
  };
}

export function ConsoleView({
  panels,
  layout,
  isLoading,
  errorMessage,
  readOnly,
  onChange,
  onEffectiveChange,
  addPanelDialogOpen: addPanelDialogOpenProp,
  onAddPanelDialogOpenChange,
  visualDiff,
}: ConsoleViewProps) {
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

  const { localPanels, localLayout, handleAddPanel, handleDeletePanel, handlePanelContentChange, handleLayoutChange } =
    useConsolePanelState(panels, layout, onChange, onEffectiveChange);
  const visualDiffWithLocalDeletes = useMemo(
    () => withLocalDeletedPanels(visualDiff, panels, layout, localPanels, localLayout),
    [visualDiff, panels, layout, localPanels, localLayout],
  );

  const confirmAddPanel = useCallback(
    (name: string, type: PanelType) => {
      handleAddPanel(name, type);
      setAddPanelOpen(false);
    },
    [handleAddPanel, setAddPanelOpen],
  );

  if (errorMessage) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-2 p-8 text-sm text-red-600 dark:text-red-400">
        <p className="font-medium">Failed to load console</p>
        <p className="text-slate-500 dark:text-gray-400">{errorMessage}</p>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <Loader2 className="h-5 w-5 animate-spin text-slate-400 dark:text-gray-500" />
      </div>
    );
  }

  if (localPanels.length === 0 && !hasRemovedDiffPanels(visualDiffWithLocalDeletes)) {
    return (
      <>
        <EmptyState onAddFirstPanel={() => setAddPanelOpen(true)} />
        <AddPanelDialog open={addPanelOpen} onConfirm={confirmAddPanel} onCancel={() => setAddPanelOpen(false)} />
      </>
    );
  }

  return (
    <>
      <ConsoleGrid
        panels={localPanels}
        layout={localLayout}
        readOnly={readOnly}
        visualDiff={visualDiffWithLocalDeletes}
        onDeletePanel={handleDeletePanel}
        onPanelContentChange={handlePanelContentChange}
        onLayoutChange={handleLayoutChange}
      />
      <AddPanelDialog open={addPanelOpen} onConfirm={confirmAddPanel} onCancel={() => setAddPanelOpen(false)} />
    </>
  );
}

function hasRemovedDiffPanels(visualDiff?: ConsoleViewProps["visualDiff"]): boolean {
  return !!visualDiff?.enabled && !!visualDiff.summary?.items.some((item) => item.changeType === "removed");
}

function withLocalDeletedPanels(
  visualDiff: ConsoleViewProps["visualDiff"],
  persistedPanels: ConsolePanel[],
  persistedLayout: ConsoleLayoutItem[],
  localPanels: ConsolePanel[],
  localLayout: ConsoleLayoutItem[],
): ConsoleViewProps["visualDiff"] {
  if (!visualDiff?.enabled) return visualDiff;

  const localDeletedItems = buildLocalDeletedDiffItems(
    visualDiff.summary,
    persistedPanels,
    persistedLayout,
    localPanels,
    localLayout,
  );
  if (localDeletedItems.length === 0) return visualDiff;

  const localDeletedIds = new Set(localDeletedItems.map((item) => item.id));
  const existingItems = visualDiff.summary?.items ?? [];
  const locallyRemovedItems = localDeletedItems.filter((item) => {
    const existingItem = existingItems.find((existing) => existing.id === item.id);
    return existingItem?.changeType !== "added";
  });
  const items = [...existingItems.filter((item) => !localDeletedIds.has(item.id)), ...locallyRemovedItems].sort(
    (left, right) => left.id.localeCompare(right.id),
  );

  return {
    ...visualDiff,
    summary: {
      items,
      addedCount: countDiffItems(items, "added"),
      updatedCount: countDiffItems(items, "updated"),
      removedCount: countDiffItems(items, "removed"),
    },
  };
}

function buildLocalDeletedDiffItems(
  summary: DraftConsoleDiffSummary | undefined,
  persistedPanels: ConsolePanel[],
  persistedLayout: ConsoleLayoutItem[],
  localPanels: ConsolePanel[],
  localLayout: ConsoleLayoutItem[],
): DraftConsoleDiffItem[] {
  const persistedPanelsById = new Map(persistedPanels.map((panel) => [panel.id, panel]));
  const persistedLayoutById = new Map(persistedLayout.map((item) => [item.i, item]));
  const localIds = new Set([...localPanels.map((panel) => panel.id), ...localLayout.map((item) => item.i)]);
  const existingItemsById = new Map((summary?.items ?? []).map((item) => [item.id, item]));

  return Array.from(new Set([...persistedPanelsById.keys(), ...persistedLayoutById.keys()]))
    .filter((id) => id && !localIds.has(id))
    .map((id) => {
      const existingItem = existingItemsById.get(id);
      const panel = persistedPanelsById.get(id);
      return {
        id,
        title: existingItem?.title ?? panelTitle(panel, id),
        changeType: "removed",
        panel,
        layout: persistedLayoutById.get(id),
        lines: existingItem?.lines ?? [],
      };
    });
}

function panelTitle(panel: ConsolePanel | undefined, id: string): string {
  const title = panel?.content.title;
  return typeof title === "string" && title.trim() ? title.trim() : id || "Untitled panel";
}

function countDiffItems(items: DraftConsoleDiffItem[], changeType: DraftConsoleDiffItem["changeType"]): number {
  return items.filter((item) => item.changeType === changeType).length;
}

function EmptyState({ onAddFirstPanel }: { onAddFirstPanel?: () => void }) {
  return (
    <div className="flex flex-1 items-center justify-center p-6 sm:p-8" data-testid="console-empty-state">
      <div
        className={cn(
          "flex w-full max-w-3xl flex-col items-center overflow-hidden rounded-2xl border border-slate-950/15 bg-white dark:border-gray-700/70",
          CONSOLE_PANEL_SHELL_SURFACE,
        )}
      >
        <div className="flex flex-col items-center px-4 pb-6 pt-8 text-center sm:px-6">
          <div className="flex h-12 w-12 items-center justify-center rounded-full bg-sky-50 dark:bg-gray-800">
            <LayoutGrid className="h-6 w-6 text-sky-600 dark:text-gray-300" />
          </div>
          <h3 className="mt-4 text-lg font-medium tracking-tight text-slate-900 dark:text-gray-100">
            Build your console
          </h3>
          <p className="mx-auto mt-1 max-w-xs text-sm leading-normal text-gray-500 dark:text-gray-400">
            Console panels surface the most important docs, links, and live data for this canvas.
          </p>
          {onAddFirstPanel ? (
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="mt-7"
              onClick={onAddFirstPanel}
              data-testid="console-add-first-panel"
            >
              <Plus className="mr-1 h-3.5 w-3.5" aria-hidden />
              Add First Panel
            </Button>
          ) : null}
        </div>

        <div className="mt-6 grid w-full gap-3 border-t border-slate-950/15 px-4 pb-8 pt-6 dark:border-gray-700/70 sm:grid-cols-3 sm:px-6">
          <div className="rounded-xl px-2 py-3 text-left">
            <FileText className="h-5 w-5 text-sky-600 dark:text-gray-300" aria-hidden />
            <h4 className="mt-3 text-sm font-medium text-slate-900 dark:text-gray-100">Document with markdown</h4>
            <p className="mt-1 text-sm leading-normal text-gray-500 dark:text-gray-400">
              Write runbooks, links, and notes in markdown.
            </p>
          </div>
          <div className="rounded-xl px-2 py-3 text-left">
            <Table2 className="h-5 w-5 text-sky-600 dark:text-gray-300" aria-hidden />
            <h4 className="mt-3 text-sm font-medium text-slate-900 dark:text-gray-100">Show live data</h4>
            <p className="mt-1 text-sm leading-normal text-gray-500 dark:text-gray-400">
              Tables, charts, and KPIs over executions or memory.
            </p>
          </div>
          <div className="rounded-xl px-2 py-3 text-left">
            <Workflow className="h-5 w-5 text-sky-600 dark:text-gray-300" aria-hidden />
            <h4 className="mt-3 text-sm font-medium text-slate-900 dark:text-gray-100">Surface key nodes</h4>
            <p className="mt-1 text-sm leading-normal text-gray-500 dark:text-gray-400">
              Pin a node with its live status and an optional Run button.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}

const PANEL_TYPE_ICONS: Record<PanelType, typeof FileText> = {
  markdown: FileText,
  html: CodeXml,
  node: Workflow,
  nodes: Network,
  table: Table2,
  chart: LineChart,
  number: Hash,
  scorecard: Gauge,
};

/**
 * Add Panel dialog with a type picker. The user picks one of the five panel
 * kinds, names the panel, and confirms — the resulting panel is seeded with a
 * sensible per-type template so the editor opens straight into a working form.
 */
function AddPanelDialog({
  open,
  onConfirm,
  onCancel,
}: {
  open: boolean;
  onConfirm: (name: string, type: PanelType) => void;
  onCancel: () => void;
}) {
  const [name, setName] = useState("");
  const [type, setType] = useState<PanelType>("markdown");
  const slug = name
    .toLowerCase()
    .trim()
    .replace(/\s+/g, "-")
    .replace(/[^a-z0-9-]/g, "")
    .replace(/-+/g, "-")
    .replace(/^-|-$/g, "");
  const isValid = slug.length > 0;

  const reset = () => {
    setName("");
    setType("markdown");
  };

  const submit = () => {
    if (!isValid) return;
    onConfirm(name.trim(), type);
    reset();
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(isOpen) => {
        if (!isOpen) {
          reset();
          onCancel();
        }
      }}
    >
      <DialogContent className="sm:max-w-2xl dark:border-gray-700/70 dark:bg-gray-900">
        <DialogHeader>
          <DialogTitle className="text-base font-medium">Add panel</DialogTitle>
        </DialogHeader>
        <div className="space-y-4 py-2">
          <div className="space-y-1.5">
            <Label htmlFor="panel-name" className="mb-3">
              Name
            </Label>
            <Input
              id="panel-name"
              placeholder="Panel name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              onKeyDown={(e: React.KeyboardEvent<HTMLInputElement>) => {
                if (e.key === "Enter") {
                  e.preventDefault();
                  submit();
                }
              }}
              autoFocus
              data-testid="add-panel-name-input"
            />
          </div>
          <div className="space-y-1.5">
            <Label className="mb-3">Type</Label>
            <div className="grid grid-cols-2 gap-4 sm:grid-cols-3" role="radiogroup" aria-label="Panel type">
              {CREATABLE_PANEL_TYPES.map((t) => {
                const meta = PANEL_TYPE_META[t];
                const Icon = PANEL_TYPE_ICONS[t];
                const selected = type === t;
                return (
                  <button
                    key={t}
                    type="button"
                    role="radio"
                    aria-checked={selected}
                    onClick={() => setType(t)}
                    className={cn(
                      "flex flex-col items-start gap-1 rounded-md border bg-white p-3 text-left transition-colors dark:bg-gray-900",
                      "hover:border-sky-400 hover:bg-sky-50 dark:hover:border-gray-500 dark:hover:bg-gray-800",
                      selected
                        ? "border-sky-500 bg-sky-50 dark:border-gray-500 dark:bg-gray-800"
                        : "border-slate-200 dark:border-gray-700/70",
                    )}
                    data-testid={`add-panel-type-${t}`}
                  >
                    <div className="flex items-center gap-1.5">
                      <Icon className="h-4 w-4 text-slate-600 dark:text-gray-400" aria-hidden />
                      <span className="text-sm font-medium text-slate-800 dark:text-gray-100">{meta.label}</span>
                    </div>
                    <span className="text-xs leading-normal text-gray-500 dark:text-gray-400">{meta.description}</span>
                  </button>
                );
              })}
            </div>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onCancel}>
            Cancel
          </Button>
          <Button onClick={submit} disabled={!isValid} data-testid="add-panel-confirm">
            Add
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
