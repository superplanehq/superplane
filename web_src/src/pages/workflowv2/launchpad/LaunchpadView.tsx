import "react-grid-layout/css/styles.css";
import "react-resizable/css/styles.css";
import "./launchpad.css";

import { useCallback, useEffect, useMemo, useRef, useState, type ComponentType } from "react";
import { ReactGridLayout, type Layout, type LayoutItem } from "react-grid-layout/legacy";
import {
  Plus,
  GripVertical,
  Loader2,
  Pencil,
  Trash2,
  ChevronsUpDown,
  LayoutDashboard,
  AtSign,
  BarChart3,
  Play,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import type { NodeChipContext } from "@/ui/Markdown/CanvasMarkdown";
import { cn } from "@/lib/utils";
import type { LaunchpadLayoutItem, LaunchpadPanel } from "@/hooks/useCanvasData";
import { useLaunchpadHeaderSlotSetter } from "@/ui/CanvasPage/LaunchpadHeaderSlotContext";
import {
  getPanelDef,
  listPanelDefs,
  type PanelDef,
  type PanelImperativeHandle,
  type PanelRenderCtx,
} from "./panelRegistry";

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
  readOnly: boolean;
  nodeRefs?: NodeChipContext;
  canvasId?: string;
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

const AUTO_HEIGHT_DEBOUNCE_MS = 100;

/**
 * Observes the panel body and reports its natural content height (in pixels)
 * back to the parent so it can resize the grid item to fit. Only does work
 * when `enabled` is true.
 *
 * Uses `scrollHeight` rather than `contentRect.height` so we get the natural
 * content size even when the parent grid item visually clips the body. Both
 * `ResizeObserver` (catches direct size changes / window resize) and
 * `MutationObserver` (catches descendant additions like new table rows) are
 * wired so dynamic content keeps the panel in sync.
 */
function useAutoHeightObserver(enabled: boolean, onMeasured: (contentHeightPx: number) => void) {
  const ref = useRef<HTMLDivElement | null>(null);
  // Stash the latest callback in a ref so we don't reset the observers every
  // time `onMeasured` changes identity (it depends on layout state).
  const callbackRef = useRef(onMeasured);
  useEffect(() => {
    callbackRef.current = onMeasured;
  }, [onMeasured]);

  useEffect(() => {
    if (!enabled) return;
    const node = ref.current;
    if (!node) return;
    if (typeof ResizeObserver === "undefined" || typeof MutationObserver === "undefined") return;

    let timer: ReturnType<typeof setTimeout> | null = null;
    const measure = () => {
      if (timer !== null) clearTimeout(timer);
      timer = setTimeout(() => {
        if (!ref.current) return;
        const h = ref.current.scrollHeight;
        // Skip zero-height measurements — they happen during initial mount
        // before content has rendered, and snapping the panel to a single
        // row is worse than leaving its current height alone.
        if (h <= 0) return;
        callbackRef.current(h);
      }, AUTO_HEIGHT_DEBOUNCE_MS);
    };

    const ro = new ResizeObserver(measure);
    ro.observe(node);
    const mo = new MutationObserver(measure);
    mo.observe(node, { childList: true, subtree: true, characterData: true });
    measure();

    return () => {
      if (timer !== null) clearTimeout(timer);
      ro.disconnect();
      mo.disconnect();
    };
  }, [enabled]);

  return ref;
}

export function LaunchpadView({
  panels,
  layout,
  isLoading,
  errorMessage,
  readOnly,
  nodeRefs,
  canvasId,
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

  const ctx: PanelRenderCtx = useMemo(() => ({ nodeRefs, canvasId }), [nodeRefs, canvasId]);

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

  // Render the Add panel control into the secondary header slot when one is
  // available. Falls back to the empty-state button if no provider is mounted
  // (e.g. during isolated component tests). The setter is stable across
  // renders, and we route through a ref so handleAddPanel's identity churn
  // (it depends on the live panel/layout state) doesn't re-fire the effect.
  const setLaunchpadHeaderNode = useLaunchpadHeaderSlotSetter();
  const handleAddPanelRef = useRef(handleAddPanel);
  handleAddPanelRef.current = handleAddPanel;
  const stableHandleAddPanel = useCallback((def: PanelDef) => {
    handleAddPanelRef.current(def);
  }, []);
  useEffect(() => {
    if (!setLaunchpadHeaderNode) return;
    if (readOnly) {
      setLaunchpadHeaderNode(null);
      return;
    }
    setLaunchpadHeaderNode(<AddPanelButton onAdd={stableHandleAddPanel} />);
    return () => {
      setLaunchpadHeaderNode(null);
    };
  }, [setLaunchpadHeaderNode, readOnly, stableHandleAddPanel]);

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
      //
      // For panels with autoHeight=true, the ResizeObserver in PanelChrome
      // owns the `h` dimension; ignore whatever react-grid-layout proposes
      // here so a transient drag doesn't stomp the auto-fit value.
      const nextLayout: LaunchpadLayoutItem[] = newLayout.map((item: LayoutItem) => {
        const existing = localLayout.find((l) => l.i === item.i);
        const autoHeight = existing?.autoHeight === true;
        return {
          i: item.i,
          x: item.x,
          y: item.y,
          w: item.w,
          h: autoHeight ? (existing?.h ?? item.h) : item.h,
          ...(existing?.minW !== undefined ? { minW: existing.minW } : {}),
          ...(existing?.minH !== undefined ? { minH: existing.minH } : {}),
          ...(autoHeight ? { autoHeight: true } : {}),
        };
      });
      const before = localLayout
        .map((l) => `${l.i}:${l.x},${l.y},${l.w},${l.h},${l.autoHeight ? 1 : 0}`)
        .sort()
        .join("|");
      const after = nextLayout
        .map((l) => `${l.i}:${l.x},${l.y},${l.w},${l.h},${l.autoHeight ? 1 : 0}`)
        .sort()
        .join("|");
      if (before === after) return;
      setLocalLayout(nextLayout);
      queueSave(localPanels, nextLayout);
    },
    [localLayout, localPanels, queueSave],
  );

  const handleToggleAutoHeight = useCallback(
    (id: string) => {
      const nextLayout = localLayout.map((l) => (l.i === id ? { ...l, autoHeight: !l.autoHeight } : l));
      setLocalLayout(nextLayout);
      queueSave(localPanels, nextLayout);
    },
    [localLayout, localPanels, queueSave],
  );

  // Translate a measured pixel height into row units, snapping up so we never
  // clip content. Mirrors react-grid-layout's row math: each row is
  // `ROW_HEIGHT + MARGIN[1]` tall (the trailing margin is shared with the
  // next row, hence the `+ MARGIN[1]` in the numerator).
  const handleAutoHeightMeasured = useCallback(
    (id: string, contentHeightPx: number) => {
      const item = localLayout.find((l) => l.i === id);
      if (!item || item.autoHeight !== true) return;
      const minH = item.minH ?? 1;
      const rowsNeeded = Math.max(minH, Math.ceil((contentHeightPx + MARGIN[1]) / (ROW_HEIGHT + MARGIN[1])));
      if (rowsNeeded === item.h) return;
      const nextLayout = localLayout.map((l) => (l.i === id ? { ...l, h: rowsNeeded } : l));
      setLocalLayout(nextLayout);
      queueSave(localPanels, nextLayout);
    },
    [localLayout, localPanels, queueSave],
  );

  if (errorMessage) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-2 p-8 text-sm text-red-600">
        <p className="font-medium">Failed to load Apps</p>
        <p className="text-slate-500">{errorMessage}</p>
      </div>
    );
  }

  const showEmptyState = !isLoading && localPanels.length === 0;

  return (
    <div ref={containerRef} className="relative flex h-full w-full flex-col overflow-auto">
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
              resizeHandles={["s", "e", "se"]}
              compactType="vertical"
              preventCollision={false}
              draggableHandle={DRAG_HANDLE_SELECTOR}
              onLayoutChange={handleLayoutChange}
            >
              {localPanels.map((panel) => {
                const layoutItem = localLayout.find((l) => l.i === panel.id);
                const autoHeight = layoutItem?.autoHeight === true;
                return (
                  <div
                    key={panel.id}
                    data-testid={`launchpad-panel-${panel.id}`}
                    data-auto-height={autoHeight ? "true" : "false"}
                  >
                    <PanelChrome
                      panel={panel}
                      readOnly={readOnly}
                      ctx={ctx}
                      autoHeight={autoHeight}
                      onDelete={() => handleDeletePanel(panel.id)}
                      onChange={(content) => handlePanelContentChange(panel.id, content)}
                      onToggleAutoHeight={() => handleToggleAutoHeight(panel.id)}
                      onAutoHeightMeasured={(h) => handleAutoHeightMeasured(panel.id, h)}
                    />
                  </div>
                );
              })}
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
  autoHeight,
  onDelete,
  onChange,
  onToggleAutoHeight,
  onAutoHeightMeasured,
}: {
  panel: PanelInternal;
  readOnly: boolean;
  ctx: PanelRenderCtx;
  autoHeight: boolean;
  onDelete: () => void;
  onChange: (content: Record<string, unknown>) => void;
  onToggleAutoHeight: () => void;
  onAutoHeightMeasured: (contentHeightPx: number) => void;
}) {
  const def = getPanelDef(panel.type);
  const handleRef = useRef<PanelImperativeHandle | null>(null);
  const [confirmingDelete, setConfirmingDelete] = useState(false);
  const autoHeightRef = useAutoHeightObserver(autoHeight, onAutoHeightMeasured);

  // Build a stable per-panel ctx so the inner renderer can register an
  // imperative handle (used by the chrome's Edit button). Reusing the parent
  // ctx avoids losing the nodeRefs / canvasId fields.
  const panelCtx: PanelRenderCtx = useMemo(
    () => ({
      ...ctx,
      registerImperativeHandle: (handle) => {
        handleRef.current = handle;
      },
    }),
    [ctx],
  );

  if (!def) {
    return (
      <div className="flex h-full w-full flex-col items-center justify-center rounded-md border border-dashed border-amber-400 bg-amber-50 p-3 text-xs text-amber-700">
        <p className="font-medium">Unknown panel type</p>
        <p>"{panel.type}"</p>
      </div>
    );
  }

  const content = def.normalize(panel.content);
  const supportsEdit = !readOnly && def.supportsEdit === true;

  const triggerEdit = () => {
    if (readOnly) return;
    handleRef.current?.startEdit();
  };

  return (
    <>
      <Card
        data-panel-chrome
        className={cn(
          "group/panel relative flex h-full w-full flex-col overflow-hidden gap-0 p-0 rounded-lg",
          "border-slate-200/80 bg-white",
          "shadow-[0_1px_2px_rgb(15_23_42_/_0.04)]",
          "transition-[box-shadow,border-color,transform] duration-150",
          "hover:border-slate-300 hover:shadow-[0_4px_12px_-2px_rgb(15_23_42_/_0.08)]",
        )}
      >
        <div ref={autoHeightRef} className="min-h-0 flex-1 overflow-hidden">
          {def.render({
            content,
            readOnly,
            ctx: panelCtx,
            onChange: (next) => onChange(next as Record<string, unknown>),
          })}
        </div>
        {!readOnly ? (
          <PanelOverlay
            supportsEdit={supportsEdit}
            autoHeight={autoHeight}
            onEdit={triggerEdit}
            onToggleAutoHeight={onToggleAutoHeight}
            onRequestDelete={() => setConfirmingDelete(true)}
          />
        ) : null}
      </Card>
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

function PanelOverlay({
  supportsEdit,
  autoHeight,
  onEdit,
  onToggleAutoHeight,
  onRequestDelete,
}: {
  supportsEdit: boolean;
  autoHeight: boolean;
  onEdit: () => void;
  onToggleAutoHeight: () => void;
  onRequestDelete: () => void;
}) {
  // Single hover-revealed cluster anchored top-right. Order: drag, edit,
  // grow-with-content toggle, delete. The drag icon keeps the
  // `.launchpad-drag-handle` class so react-grid-layout's `draggableHandle`
  // selector still resolves to it.
  return (
    <div
      className="absolute right-1 top-1 z-20 flex items-center gap-0.5 rounded-md bg-white/85 opacity-0 shadow-[0_1px_2px_rgb(15_23_42_/_0.04)] backdrop-blur transition-opacity duration-150 group-hover/panel:opacity-100 focus-within:opacity-100"
      data-testid="launchpad-panel-actions"
      // Stop the drag-handle drag from initiating when interacting with the
      // action cluster — clicks on these buttons should not start a drag.
      onMouseDown={(e) => e.stopPropagation()}
      onTouchStart={(e) => e.stopPropagation()}
    >
      <span
        className="launchpad-drag-handle inline-flex h-6 w-6 cursor-grab items-center justify-center rounded text-slate-500 transition-colors hover:text-slate-700 active:cursor-grabbing"
        data-testid="launchpad-drag-handle"
        role="button"
        aria-label="Drag panel"
        title="Drag to move"
        // The drag handle isn't focusable on its own — the rest of the cluster
        // already provides keyboard access — but we still want to swallow
        // mousedowns so the drag (rather than focus shift) starts.
        onMouseDown={(e) => e.stopPropagation()}
      >
        <GripVertical className="h-3.5 w-3.5" />
      </span>
      {supportsEdit ? (
        <Button
          type="button"
          size="icon-xs"
          variant="ghost"
          onClick={onEdit}
          aria-label="Edit panel"
          title="Edit"
          data-testid="launchpad-edit-panel"
          className="h-6 w-6 text-slate-500 hover:text-slate-700"
        >
          <Pencil className="h-3.5 w-3.5" />
        </Button>
      ) : null}
      <Button
        type="button"
        size="icon-xs"
        variant="ghost"
        onClick={onToggleAutoHeight}
        aria-pressed={autoHeight}
        aria-label="Grow with content"
        title={autoHeight ? "Stop growing with content" : "Grow with content"}
        data-testid="launchpad-toggle-auto-height"
        data-active={autoHeight ? "true" : "false"}
        className={cn(
          "h-6 w-6 transition-colors",
          autoHeight ? "bg-slate-100 text-slate-700 hover:bg-slate-200" : "text-slate-500 hover:text-slate-700",
        )}
      >
        <ChevronsUpDown className="h-3.5 w-3.5" />
      </Button>
      <Button
        type="button"
        size="icon-xs"
        variant="ghost"
        onClick={onRequestDelete}
        aria-label="Delete panel"
        title="Delete"
        data-testid="launchpad-delete-panel-button"
        className="h-6 w-6 text-slate-500 hover:bg-red-50 hover:text-red-600"
      >
        <Trash2 className="h-3.5 w-3.5" />
      </Button>
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
      <DialogContent data-testid="launchpad-delete-confirm">
        <DialogHeader>
          <DialogTitle>Delete this panel?</DialogTitle>
          <DialogDescription>
            This panel and its contents will be removed from the Apps page. You can add it back later, but the content
            isn't recoverable.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button type="button" variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button type="button" variant="destructive" onClick={onConfirm} data-testid="launchpad-delete-confirm-action">
            Delete panel
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
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
      <div className="flex w-full max-w-2xl flex-col items-center gap-5 rounded-xl border border-dashed border-slate-300 bg-white/70 px-8 py-10 text-center shadow-[0_1px_2px_rgb(15_23_42_/_0.04)] backdrop-blur">
        <div className="flex h-14 w-14 items-center justify-center rounded-full bg-slate-100">
          <LayoutDashboard className="h-7 w-7 text-slate-500" />
        </div>
        <div className="flex flex-col gap-1.5">
          <h3 className="text-lg font-semibold text-slate-800">Build your Apps page</h3>
          <p className="mx-auto max-w-md text-sm leading-relaxed text-slate-500">
            Apps panels surface the most important docs, links, and live data for this canvas. Drag and resize panels to
            lay them out the way your team works.
          </p>
        </div>

        <div className="grid w-full grid-cols-1 gap-3 sm:grid-cols-3">
          <FeatureTile
            icon={AtSign}
            title="Reference nodes"
            description="Drop in @my-node chips that show live status."
          />
          <FeatureTile
            icon={BarChart3}
            title="Show live data"
            description="Embed canvas memory or executions with widget blocks."
          />
          <FeatureTile
            icon={Play}
            title="Trigger runs"
            description="Add Run buttons for manual triggers right in markdown."
          />
        </div>

        {!readOnly ? (
          <div className="flex items-center gap-3">
            <AddPanelButton onAdd={onAdd} />
          </div>
        ) : null}
      </div>
    </div>
  );
}

function FeatureTile({
  icon: Icon,
  title,
  description,
}: {
  icon: ComponentType<{ className?: string }>;
  title: string;
  description: string;
}) {
  return (
    <Card className="gap-2 border-slate-200/80 bg-white px-3 py-3 text-left shadow-none">
      <div className="flex items-center gap-2">
        <span className="flex h-6 w-6 items-center justify-center rounded-md bg-sky-50 text-sky-600">
          <Icon className="h-3.5 w-3.5" />
        </span>
        <span className="text-xs font-semibold text-slate-700">{title}</span>
      </div>
      <p className="text-xs leading-snug text-slate-500">{description}</p>
    </Card>
  );
}
