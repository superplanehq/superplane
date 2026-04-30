import "react-grid-layout/css/styles.css";
import "react-resizable/css/styles.css";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ReactGridLayout, type Layout, type LayoutItem } from "react-grid-layout/legacy";
import { Plus, Trash2, GripVertical, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import type { NodeChipContext } from "@/ui/Markdown/CanvasMarkdown";
import { cn } from "@/lib/utils";
import type { LaunchpadLayoutItem, LaunchpadPanel } from "@/hooks/useCanvasData";
import { getPanelDef, listPanelDefs, type PanelDef, type PanelRenderCtx } from "./panelRegistry";

const GRID_COLS = 12;
const ROW_HEIGHT = 40;
const MARGIN: [number, number] = [12, 12];
const CONTAINER_PADDING: [number, number] = [16, 16];
const SAVE_DEBOUNCE_MS = 350;
const DRAG_HANDLE_SELECTOR = ".launchpad-drag-handle";

export interface LaunchpadViewProps {
  panels: LaunchpadPanel[];
  layout: LaunchpadLayoutItem[];
  isLoading: boolean;
  errorMessage?: string;
  isSaving?: boolean;
  readOnly: boolean;
  nodeRefs?: NodeChipContext;
  /**
   * Persists the next launchpad state. Called whenever the user makes any
   * change (drag, resize, add, delete, edit). Implementations should debounce
   * upstream if needed; this view dedupes drag-induced churn locally.
   */
  onChange: (next: { panels: LaunchpadPanel[]; layout: LaunchpadLayoutItem[] }) => void;
}

interface PanelInternal {
  id: string;
  type: string;
  content: Record<string, unknown>;
}

export function LaunchpadView({
  panels,
  layout,
  isLoading,
  errorMessage,
  isSaving,
  readOnly,
  nodeRefs,
  onChange,
}: LaunchpadViewProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const [containerWidth, setContainerWidth] = useState<number>(0);

  // Keep a local working copy of panels + layout so the user sees instant
  // updates when dragging/typing, even before the upstream mutation settles.
  // We sync from props only when the upstream identity changes (deep
  // structural sync would clobber in-flight edits).
  const [localPanels, setLocalPanels] = useState<PanelInternal[]>(panels);
  const [localLayout, setLocalLayout] = useState<LaunchpadLayoutItem[]>(layout);
  const lastPropsHashRef = useRef<string>("");
  useEffect(() => {
    const next = JSON.stringify({ panels, layout });
    if (next !== lastPropsHashRef.current) {
      lastPropsHashRef.current = next;
      setLocalPanels(panels);
      setLocalLayout(layout);
    }
  }, [panels, layout]);

  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    const update = () => setContainerWidth(el.clientWidth);
    update();
    if (typeof ResizeObserver === "undefined") return;
    const ro = new ResizeObserver(update);
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  // Debounce save so a continuous drag/resize collapses into a single network
  // call. The latest state always wins.
  const saveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pendingChangeRef = useRef<{ panels: PanelInternal[]; layout: LaunchpadLayoutItem[] } | null>(null);
  const queueSave = useCallback(
    (nextPanels: PanelInternal[], nextLayout: LaunchpadLayoutItem[]) => {
      pendingChangeRef.current = { panels: nextPanels, layout: nextLayout };
      if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
      saveTimerRef.current = setTimeout(() => {
        const pending = pendingChangeRef.current;
        if (!pending) return;
        onChange(pending);
        pendingChangeRef.current = null;
      }, SAVE_DEBOUNCE_MS);
    },
    [onChange],
  );
  // Flush pending changes on unmount so a tab switch doesn't drop work.
  useEffect(() => {
    return () => {
      if (saveTimerRef.current) {
        clearTimeout(saveTimerRef.current);
      }
      const pending = pendingChangeRef.current;
      if (pending) {
        onChange(pending);
      }
    };
  }, [onChange]);

  const ctx: PanelRenderCtx = useMemo(() => ({ nodeRefs }), [nodeRefs]);

  const handleAddPanel = useCallback(
    (def: PanelDef) => {
      const id = `panel-${Math.random().toString(36).slice(2, 10)}`;
      const newPanel: PanelInternal = {
        id,
        type: def.type,
        content: { ...def.defaultContent },
      };
      // Find the first empty row at the bottom of the existing layout so the
      // new panel doesn't overlap with anything.
      const maxBottom = localLayout.reduce((acc, item) => Math.max(acc, item.y + item.h), 0);
      const newLayoutItem: LaunchpadLayoutItem = {
        i: id,
        x: 0,
        y: maxBottom,
        w: def.defaultSize.w,
        h: def.defaultSize.h,
        ...(def.defaultSize.minW !== undefined ? { minW: def.defaultSize.minW } : {}),
        ...(def.defaultSize.minH !== undefined ? { minH: def.defaultSize.minH } : {}),
      };
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

  const handleLayoutChange = useCallback(
    (newLayout: Layout) => {
      // Only persist when something actually moved — react-grid-layout fires
      // onLayoutChange on every render including the initial mount.
      const nextLayout: LaunchpadLayoutItem[] = newLayout.map((item: LayoutItem) => {
        const existing = localLayout.find((l) => l.i === item.i);
        return {
          i: item.i,
          x: item.x,
          y: item.y,
          w: item.w,
          h: item.h,
          ...(existing?.minW !== undefined ? { minW: existing.minW } : {}),
          ...(existing?.minH !== undefined ? { minH: existing.minH } : {}),
        };
      });
      const before = localLayout
        .map((l) => `${l.i}:${l.x},${l.y},${l.w},${l.h}`)
        .sort()
        .join("|");
      const after = nextLayout
        .map((l) => `${l.i}:${l.x},${l.y},${l.w},${l.h}`)
        .sort()
        .join("|");
      if (before === after) return;
      setLocalLayout(nextLayout);
      queueSave(localPanels, nextLayout);
    },
    [localLayout, localPanels, queueSave],
  );

  if (errorMessage) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-2 p-8 text-sm text-red-600">
        <p className="font-medium">Failed to load launchpad</p>
        <p className="text-slate-500">{errorMessage}</p>
      </div>
    );
  }

  const showEmptyState = !isLoading && localPanels.length === 0;

  return (
    <div ref={containerRef} className="relative flex h-full w-full flex-col overflow-auto">
      <div className="sticky top-0 z-10 flex h-12 shrink-0 items-center justify-between border-b border-slate-200 bg-white/80 px-4 backdrop-blur">
        <h2 className="text-sm font-semibold text-slate-800">Launchpad</h2>
        <div className="flex items-center gap-2">
          {isSaving ? (
            <span className="inline-flex items-center gap-1 text-xs text-slate-500">
              <Loader2 className="h-3 w-3 animate-spin" />
              Saving
            </span>
          ) : null}
          {!readOnly ? <AddPanelButton onAdd={handleAddPanel} /> : null}
        </div>
      </div>

      {isLoading ? (
        <div className="flex flex-1 items-center justify-center">
          <Loader2 className="h-5 w-5 animate-spin text-slate-400" />
        </div>
      ) : showEmptyState ? (
        <EmptyState readOnly={readOnly} onAdd={handleAddPanel} />
      ) : (
        <div className="flex-1">
          {containerWidth > 0 ? (
            <ReactGridLayout
              className="launchpad-grid"
              cols={GRID_COLS}
              rowHeight={ROW_HEIGHT}
              width={containerWidth}
              margin={MARGIN}
              containerPadding={CONTAINER_PADDING}
              layout={localLayout.map((l) => ({
                i: l.i,
                x: l.x,
                y: l.y,
                w: l.w,
                h: l.h,
                minW: l.minW,
                minH: l.minH,
              }))}
              isDraggable={!readOnly}
              isResizable={!readOnly}
              compactType="vertical"
              preventCollision={false}
              draggableHandle={DRAG_HANDLE_SELECTOR}
              onLayoutChange={handleLayoutChange}
            >
              {localPanels.map((panel) => (
                <div key={panel.id} data-testid={`launchpad-panel-${panel.id}`}>
                  <PanelChrome
                    panel={panel}
                    readOnly={readOnly}
                    ctx={ctx}
                    onDelete={() => handleDeletePanel(panel.id)}
                    onChange={(content) => handlePanelContentChange(panel.id, content)}
                  />
                </div>
              ))}
            </ReactGridLayout>
          ) : null}
        </div>
      )}
    </div>
  );
}

function PanelChrome({
  panel,
  readOnly,
  ctx,
  onDelete,
  onChange,
}: {
  panel: PanelInternal;
  readOnly: boolean;
  ctx: PanelRenderCtx;
  onDelete: () => void;
  onChange: (content: Record<string, unknown>) => void;
}) {
  const def = getPanelDef(panel.type);

  if (!def) {
    return (
      <div className="flex h-full w-full flex-col items-center justify-center rounded-md border border-dashed border-amber-400 bg-amber-50 p-3 text-xs text-amber-700">
        <p className="font-medium">Unknown panel type</p>
        <p>"{panel.type}"</p>
      </div>
    );
  }

  const content = def.normalize(panel.content);

  return (
    <div className="group/panel relative flex h-full w-full flex-col overflow-hidden rounded-md border border-slate-200 bg-white shadow-sm">
      {!readOnly ? (
        <div
          className={cn(
            "absolute right-1.5 top-1.5 z-10 flex items-center gap-1 opacity-0 transition-opacity",
            "group-hover/panel:opacity-100 focus-within:opacity-100",
          )}
        >
          <button
            type="button"
            onClick={onDelete}
            className="inline-flex h-6 w-6 items-center justify-center rounded text-slate-500 hover:bg-red-50 hover:text-red-600"
            aria-label="Delete panel"
            data-testid="launchpad-delete-panel"
          >
            <Trash2 className="h-3.5 w-3.5" />
          </button>
          <span
            className={cn(
              "launchpad-drag-handle inline-flex h-6 w-6 cursor-move items-center justify-center rounded text-slate-500 hover:bg-slate-100",
            )}
            aria-label="Drag panel"
            data-testid="launchpad-drag-handle"
          >
            <GripVertical className="h-3.5 w-3.5" />
          </span>
        </div>
      ) : null}
      <div className="min-h-0 flex-1">
        {def.render({
          content,
          readOnly,
          ctx,
          onChange: (next) => onChange(next as Record<string, unknown>),
        })}
      </div>
    </div>
  );
}

function AddPanelButton({ onAdd }: { onAdd: (def: PanelDef) => void }) {
  const defs = listPanelDefs();
  // With one panel type today, show a single-action button. The dropdown is
  // wired up so adding a second panel type later doesn't require UI changes.
  if (defs.length === 1) {
    const only = defs[0];
    return (
      <Button size="sm" variant="default" onClick={() => onAdd(only)} data-testid="launchpad-add-panel">
        <Plus className="mr-1 h-3.5 w-3.5" />
        Add panel
      </Button>
    );
  }
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button size="sm" variant="default" data-testid="launchpad-add-panel">
          <Plus className="mr-1 h-3.5 w-3.5" />
          Add panel
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {defs.map((def) => (
          <DropdownMenuItem key={def.type} onClick={() => onAdd(def)}>
            <def.icon className="h-4 w-4" />
            {def.label}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function EmptyState({ readOnly, onAdd }: { readOnly: boolean; onAdd: (def: PanelDef) => void }) {
  return (
    <div className="flex flex-1 items-center justify-center p-8" data-testid="launchpad-empty-state">
      <div className="flex max-w-md flex-col items-center gap-3 rounded-lg border border-dashed border-slate-300 bg-white p-8 text-center">
        <h3 className="text-base font-semibold text-slate-800">Build your Launchpad</h3>
        <p className="text-sm text-slate-500">
          Add panels to surface the most important docs, links, and notes for this canvas. Panels can be dragged and
          resized into a grid that suits your team.
        </p>
        {!readOnly ? <AddPanelButton onAdd={onAdd} /> : null}
      </div>
    </div>
  );
}
