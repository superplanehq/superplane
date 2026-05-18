import { useCallback, useMemo, useState } from "react";
import GridLayout, { type Layout, WidthProvider } from "react-grid-layout";
import { Plus, Loader2, LayoutGrid, FileText, Hash, LineChart, Table2, Workflow } from "lucide-react";

import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { DashboardPanel, DashboardLayoutItem } from "@/hooks/useCanvasData";

import { MarkdownPanelCard } from "./MarkdownPanelCard";
import { NodePanelCard } from "./NodePanelCard";
import { TablePanelCard } from "./TablePanelCard";
import { ChartPanelCard } from "./ChartPanelCard";
import { NumberPanelCard } from "./NumberPanelCard";
import { useDashboardPanelState } from "./useDashboardPanelState";
import { PANEL_TYPE_META, PANEL_TYPES, type PanelType } from "./panelTypes";

import "react-grid-layout/css/styles.css";
import "./dashboard-grid.css";

const ResponsiveGridLayout = WidthProvider(GridLayout);

const DASHBOARD_GRID_COLS = 12;
const DASHBOARD_ROW_HEIGHT = 40;
const DASHBOARD_DEFAULT_MIN_W = 2;
const DASHBOARD_DEFAULT_MIN_H = 2;

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

  const { localPanels, localLayout, handleAddPanel, handleDeletePanel, handlePanelContentChange, handleLayoutChange } =
    useDashboardPanelState(panels, layout, onChange);

  const confirmAddPanel = useCallback(
    (name: string, type: PanelType) => {
      handleAddPanel(name, type);
      setAddPanelOpen(false);
    },
    [handleAddPanel, setAddPanelOpen],
  );

  const layoutItems = useMemo(() => buildRGLLayout(localPanels, localLayout), [localPanels, localLayout]);

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
      <div className="px-4 py-3">
        <ResponsiveGridLayout
          className="dashboard-grid"
          layout={layoutItems}
          cols={DASHBOARD_GRID_COLS}
          rowHeight={DASHBOARD_ROW_HEIGHT}
          margin={[12, 12]}
          containerPadding={[0, 0]}
          isDraggable={!readOnly}
          isResizable={!readOnly}
          draggableHandle=".dashboard-grid-drag-handle"
          // Anything inside `dashboard-grid-no-drag` (action buttons, the
          // "click to edit" empty-state, etc.) is exempt from drag/resize so
          // clicks reach the underlying control instead of starting a drag.
          draggableCancel=".dashboard-grid-no-drag"
          resizeHandles={["se"]}
          onLayoutChange={(next) => {
            if (readOnly) return;
            handleLayoutChange(toDashboardLayout(next));
          }}
          useCSSTransforms
          compactType="vertical"
          preventCollision={false}
        >
          {localPanels.map((panel) => (
            <div key={panel.id} className="dashboard-grid-item">
              <PanelCardRouter
                panel={panel}
                readOnly={readOnly}
                onDelete={() => handleDeletePanel(panel.id)}
                onChange={(content) => handlePanelContentChange(panel.id, content)}
              />
            </div>
          ))}
        </ResponsiveGridLayout>
      </div>
      <AddPanelDialog open={addPanelOpen} onConfirm={confirmAddPanel} onCancel={() => setAddPanelOpen(false)} />
    </div>
  );
}

/**
 * Build the layout array consumed by react-grid-layout. Panels with no layout
 * entry get a sensible default position appended to the bottom of the grid, so
 * legacy or YAML-imported dashboards that omit `layout` still render.
 */
function buildRGLLayout(panels: DashboardPanel[], layout: DashboardLayoutItem[]): Layout[] {
  const byId = new Map<string, DashboardLayoutItem>();
  for (const item of layout) byId.set(item.i, item);

  let nextY = layout.reduce((acc, item) => Math.max(acc, item.y + item.h), 0);
  const result: Layout[] = [];
  for (const panel of panels) {
    const existing = byId.get(panel.id);
    if (existing) {
      result.push({
        i: existing.i,
        x: existing.x,
        y: existing.y,
        w: existing.w,
        h: existing.h,
        minW: existing.minW ?? DASHBOARD_DEFAULT_MIN_W,
        minH: existing.minH ?? DASHBOARD_DEFAULT_MIN_H,
      });
      continue;
    }
    result.push({
      i: panel.id,
      x: 0,
      y: nextY,
      w: 12,
      h: 6,
      minW: DASHBOARD_DEFAULT_MIN_W,
      minH: DASHBOARD_DEFAULT_MIN_H,
    });
    nextY += 6;
  }
  return result;
}

function toDashboardLayout(next: Layout[]): DashboardLayoutItem[] {
  return next.map((item) => {
    const result: DashboardLayoutItem = {
      i: item.i,
      x: item.x,
      y: item.y,
      w: item.w,
      h: item.h,
    };
    if (typeof item.minW === "number") result.minW = item.minW;
    if (typeof item.minH === "number") result.minH = item.minH;
    return result;
  });
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
            <FileText className="h-5 w-5 text-sky-600" aria-hidden />
            <h4 className="mt-3 text-sm font-semibold text-slate-900">Document with markdown</h4>
            <p className="mt-1 text-xs leading-relaxed text-slate-500">Write runbooks, links, and notes in markdown.</p>
          </div>
          <div className="rounded-xl border border-slate-200 bg-slate-50/50 p-4 text-left shadow-sm">
            <Table2 className="h-5 w-5 text-sky-600" aria-hidden />
            <h4 className="mt-3 text-sm font-semibold text-slate-900">Show live data</h4>
            <p className="mt-1 text-xs leading-relaxed text-slate-500">
              Tables, charts, and KPIs over executions or memory.
            </p>
          </div>
          <div className="rounded-xl border border-slate-200 bg-slate-50/50 p-4 text-left shadow-sm">
            <Workflow className="h-5 w-5 text-sky-600" aria-hidden />
            <h4 className="mt-3 text-sm font-semibold text-slate-900">Surface key nodes</h4>
            <p className="mt-1 text-xs leading-relaxed text-slate-500">
              Pin a node with its live status and an optional Run button.
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

const PANEL_TYPE_ICONS: Record<PanelType, typeof FileText> = {
  markdown: FileText,
  node: Workflow,
  table: Table2,
  chart: LineChart,
  number: Hash,
};

function PanelCardRouter({
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
  switch (panel.type) {
    case "node":
      return <NodePanelCard panel={panel} readOnly={readOnly} onDelete={onDelete} onChange={onChange} />;
    case "table":
      return <TablePanelCard panel={panel} readOnly={readOnly} onDelete={onDelete} onChange={onChange} />;
    case "chart":
      return <ChartPanelCard panel={panel} readOnly={readOnly} onDelete={onDelete} onChange={onChange} />;
    case "number":
      return <NumberPanelCard panel={panel} readOnly={readOnly} onDelete={onDelete} onChange={onChange} />;
    case "markdown":
    default:
      return <MarkdownPanelCard panel={panel} readOnly={readOnly} onDelete={onDelete} onChange={onChange} />;
  }
}

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
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>Add panel</DialogTitle>
        </DialogHeader>
        <div className="space-y-4 py-2">
          <div className="space-y-1.5">
            <Label>Type</Label>
            <div className="grid grid-cols-2 gap-2 sm:grid-cols-3" role="radiogroup" aria-label="Panel type">
              {PANEL_TYPES.map((t) => {
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
                      "flex flex-col items-start gap-1 rounded-md border bg-white p-3 text-left transition-colors",
                      "hover:border-sky-400 hover:bg-sky-50/40",
                      selected ? "border-sky-500 bg-sky-50 ring-2 ring-sky-200" : "border-slate-200",
                    )}
                    data-testid={`add-panel-type-${t}`}
                  >
                    <div className="flex items-center gap-1.5">
                      <Icon className="h-4 w-4 text-slate-600" aria-hidden />
                      <span className="text-sm font-medium text-slate-800">{meta.label}</span>
                    </div>
                    <span className="text-[11px] leading-snug text-slate-500">{meta.description}</span>
                  </button>
                );
              })}
            </div>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="panel-name">Name</Label>
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
