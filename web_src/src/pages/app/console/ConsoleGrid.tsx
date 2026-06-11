import { useCallback, useEffect, useMemo, useState, type CSSProperties } from "react";
import GridLayout, { type Layout, WidthProvider } from "react-grid-layout";

import { cn } from "@/lib/utils";
import type { ConsoleLayoutItem, ConsolePanel } from "@/hooks/useCanvasData";
import type { DraftConsoleDiffItem, DraftConsoleDiffSummary } from "../draftConsoleDiff";

import { ConsolePanelDiffBadge, ConsolePanelDiffDialog } from "./ConsolePanelDiff";
import { PanelCardRouter } from "./ConsolePanelCards";
import { consolePanelDiffBorderClassName } from "./consolePanelDiffPresentation";
import { useConsoleGridTransitionArming } from "./useConsoleGridTransitionArming";

import "react-grid-layout/css/styles.css";
import "./console-grid.css";

const ResponsiveGridLayout = WidthProvider(GridLayout);

const DASHBOARD_GRID_COLS = 12;
const DASHBOARD_ROW_HEIGHT = 40;
const DASHBOARD_GRID_MARGIN: [number, number] = [12, 12];
const DASHBOARD_CONTAINER_PADDING: [number, number] = [0, 0];
const DASHBOARD_DEFAULT_MIN_W = 2;
const DASHBOARD_DEFAULT_MIN_H = 2;

export function ConsoleGrid({
  panels,
  layout,
  readOnly,
  visualDiff,
  onDeletePanel,
  onPanelContentChange,
  onLayoutChange,
}: {
  panels: ConsolePanel[];
  layout: ConsoleLayoutItem[];
  readOnly: boolean;
  visualDiff?: {
    enabled: boolean;
    summary?: DraftConsoleDiffSummary;
  };
  onDeletePanel: (panelId: string) => void;
  onPanelContentChange: (panelId: string, content: Record<string, unknown>) => void;
  onLayoutChange: (layout: ConsoleLayoutItem[]) => void;
}) {
  const [selectedDiffItem, setSelectedDiffItem] = useState<DraftConsoleDiffItem | null>(null);
  const [activeInteractionLayout, setActiveInteractionLayout] = useState<ConsoleLayoutItem[] | null>(null);
  const [isInteractingWithLayout, setIsInteractingWithLayout] = useState(false);
  const [editingPanelIds, setEditingPanelIds] = useState<Set<string>>(() => new Set());
  const visualDiffState = useMemo(
    () => buildConsoleVisualDiffState(visualDiff?.enabled ? visualDiff.summary : undefined),
    [visualDiff?.enabled, visualDiff?.summary],
  );
  const liveLayoutForGhosts = activeInteractionLayout ?? layout;
  const visibleRemovedDiffState = useMemo(
    () => filterRemovedDiffState(visualDiffState, liveLayoutForGhosts),
    [liveLayoutForGhosts, visualDiffState],
  );
  const layoutItems = useMemo(() => buildRGLLayout(panels, layout), [layout, panels]);
  const { transitionsArmed, gridWrapperRef, gridWidth } = useConsoleGridTransitionArming(
    panels.length > 0 || visibleRemovedDiffState.removedPanels.length > 0,
  );
  const removedGhosts = useMemo(
    () =>
      buildRemovedGhosts(
        visibleRemovedDiffState.removedPanels,
        visibleRemovedDiffState.removedLayout,
        visualDiffState.itemsById,
      ),
    [visibleRemovedDiffState.removedLayout, visibleRemovedDiffState.removedPanels, visualDiffState.itemsById],
  );
  const gridMinHeight = useMemo(
    () => layoutBottomPx([...layout, ...visibleRemovedDiffState.removedLayout], gridWidth),
    [gridWidth, layout, visibleRemovedDiffState.removedLayout],
  );

  useEffect(() => {
    if (!isInteractingWithLayout) setActiveInteractionLayout(null);
  }, [isInteractingWithLayout, layout]);

  const updateActiveInteractionLayout = useCallback(
    (next: Layout[]) => {
      setActiveInteractionLayout(toConsoleLayout(next).filter((item) => !visualDiffState.removedPanelIds.has(item.i)));
    },
    [visualDiffState.removedPanelIds],
  );
  const handleInteractionStart = useCallback(() => setIsInteractingWithLayout(true), []);
  const handleInteractionStop = useCallback(() => setIsInteractingWithLayout(false), []);
  const handleDiffDialogOpenChange = useCallback((open: boolean) => {
    if (!open) setSelectedDiffItem(null);
  }, []);
  const handlePanelEditingChange = useCallback((panelId: string, editing: boolean) => {
    setEditingPanelIds((current) => {
      const next = new Set(current);
      if (editing) next.add(panelId);
      else next.delete(panelId);
      return next;
    });
  }, []);

  return (
    <div className="flex h-full w-full flex-col overflow-auto">
      <div className="px-4 py-3">
        <div ref={gridWrapperRef} className="relative" style={{ minHeight: gridMinHeight }}>
          <RemovedPanelGhostLayer ghosts={removedGhosts} gridWidth={gridWidth} onSelectDiffItem={setSelectedDiffItem} />
          <ConsoleGridLayout
            panels={panels}
            layoutItems={layoutItems}
            readOnly={readOnly}
            transitionsArmed={transitionsArmed}
            visualDiffState={visualDiffState}
            editingPanelIds={editingPanelIds}
            onDeletePanel={onDeletePanel}
            onPanelContentChange={onPanelContentChange}
            onPanelEditingChange={handlePanelEditingChange}
            onLayoutChange={onLayoutChange}
            onSelectDiffItem={setSelectedDiffItem}
            onInteractionStart={handleInteractionStart}
            onInteractionLayoutChange={updateActiveInteractionLayout}
            onInteractionStop={handleInteractionStop}
          />
        </div>
      </div>
      <ConsolePanelDiffDialog item={selectedDiffItem} onOpenChange={handleDiffDialogOpenChange} />
    </div>
  );
}

function ConsoleGridLayout({
  panels,
  layoutItems,
  readOnly,
  transitionsArmed,
  visualDiffState,
  editingPanelIds,
  onDeletePanel,
  onPanelContentChange,
  onPanelEditingChange,
  onLayoutChange,
  onSelectDiffItem,
  onInteractionStart,
  onInteractionLayoutChange,
  onInteractionStop,
}: {
  panels: ConsolePanel[];
  layoutItems: Layout[];
  readOnly: boolean;
  transitionsArmed: boolean;
  visualDiffState: ConsoleVisualDiffState;
  editingPanelIds: Set<string>;
  onDeletePanel: (panelId: string) => void;
  onPanelContentChange: (panelId: string, content: Record<string, unknown>) => void;
  onPanelEditingChange: (panelId: string, editing: boolean) => void;
  onLayoutChange: (layout: ConsoleLayoutItem[]) => void;
  onSelectDiffItem: (item: DraftConsoleDiffItem) => void;
  onInteractionStart: () => void;
  onInteractionLayoutChange: (layout: Layout[]) => void;
  onInteractionStop: () => void;
}) {
  return (
    <ResponsiveGridLayout
      className={cn("console-grid", !transitionsArmed && "console-grid--instant")}
      layout={layoutItems}
      cols={DASHBOARD_GRID_COLS}
      rowHeight={DASHBOARD_ROW_HEIGHT}
      margin={DASHBOARD_GRID_MARGIN}
      containerPadding={DASHBOARD_CONTAINER_PADDING}
      isDraggable={!readOnly}
      isResizable={!readOnly}
      draggableHandle=".console-grid-drag-handle"
      draggableCancel=".console-grid-no-drag"
      resizeHandles={["se"]}
      onLayoutChange={(next) => {
        if (readOnly) return;
        onLayoutChange(toConsoleLayout(next).filter((item) => !visualDiffState.removedPanelIds.has(item.i)));
      }}
      onDragStart={(next) => {
        onInteractionStart();
        onInteractionLayoutChange(next);
      }}
      onDrag={(next) => onInteractionLayoutChange(next)}
      onDragStop={(next) => {
        onInteractionLayoutChange(next);
        onInteractionStop();
      }}
      onResizeStart={(next) => {
        onInteractionStart();
        onInteractionLayoutChange(next);
      }}
      onResize={(next) => onInteractionLayoutChange(next)}
      onResizeStop={(next) => {
        onInteractionLayoutChange(next);
        onInteractionStop();
      }}
      useCSSTransforms
      compactType="vertical"
      preventCollision={false}
    >
      {panels.map((panel) =>
        renderConsoleGridPanel({
          panel,
          readOnly,
          diffItem: visualDiffState.itemsById.get(panel.id),
          isEditing: editingPanelIds.has(panel.id),
          onDeletePanel,
          onPanelContentChange,
          onPanelEditingChange,
          onSelectDiffItem,
        }),
      )}
    </ResponsiveGridLayout>
  );
}

function renderConsoleGridPanel({
  panel,
  readOnly,
  diffItem,
  isEditing,
  onDeletePanel,
  onPanelContentChange,
  onPanelEditingChange,
  onSelectDiffItem,
}: {
  panel: ConsolePanel;
  readOnly: boolean;
  diffItem?: DraftConsoleDiffItem;
  isEditing: boolean;
  onDeletePanel: (panelId: string) => void;
  onPanelContentChange: (panelId: string, content: Record<string, unknown>) => void;
  onPanelEditingChange: (panelId: string, editing: boolean) => void;
  onSelectDiffItem: (item: DraftConsoleDiffItem) => void;
}) {
  const showDiffBadge = diffItem && (!isEditing || diffItem.changeType === "removed");

  return (
    <div key={panel.id} className="console-grid-item group/console-panel-diff relative rounded-lg">
      <PanelCardRouter
        panel={panel}
        readOnly={readOnly}
        onDelete={() => onDeletePanel(panel.id)}
        onChange={(content) => onPanelContentChange(panel.id, content)}
        onEditingChange={(editing) => onPanelEditingChange(panel.id, editing)}
      />
      {diffItem ? <ConsolePanelDiffBorder status={diffItem.changeType} /> : null}
      {showDiffBadge ? (
        <ConsolePanelDiffBadge
          status={diffItem.changeType}
          panelTitle={diffItem.title || panel.id}
          onShowDiff={() => onSelectDiffItem(diffItem)}
        />
      ) : null}
    </div>
  );
}

function RemovedPanelGhostLayer({
  ghosts,
  gridWidth,
  onSelectDiffItem,
}: {
  ghosts: RemovedPanelGhost[];
  gridWidth: number;
  onSelectDiffItem: (item: DraftConsoleDiffItem) => void;
}) {
  if (gridWidth <= 0 || ghosts.length === 0) return null;

  return (
    <div className="pointer-events-none absolute inset-0 z-0" data-testid="console-removed-panel-ghost-layer">
      {ghosts.map(({ panel, layout, diffItem }) => (
        <div
          key={panel.id}
          className="console-grid-removed-ghost group/console-panel-diff pointer-events-auto absolute flex min-h-0 overflow-hidden rounded-lg opacity-50"
          data-testid="console-removed-panel-ghost"
          style={gridItemStyle(layout, gridWidth)}
        >
          <PanelCardRouter panel={panel} readOnly onDelete={() => {}} onChange={() => {}} />
          <ConsolePanelDiffBorder status={diffItem.changeType} />
          <ConsolePanelDiffBadge
            status={diffItem.changeType}
            panelTitle={diffItem.title || panel.id}
            onShowDiff={() => onSelectDiffItem(diffItem)}
          />
        </div>
      ))}
    </div>
  );
}

function ConsolePanelDiffBorder({ status }: { status: DraftConsoleDiffItem["changeType"] }) {
  return (
    <div
      className={cn(
        "pointer-events-none absolute inset-0 z-[1] rounded-lg border-2",
        consolePanelDiffBorderClassName(status),
      )}
      data-testid="console-panel-diff-border"
    />
  );
}

function buildRGLLayout(panels: ConsolePanel[], layout: ConsoleLayoutItem[]): Layout[] {
  const byId = new Map<string, ConsoleLayoutItem>();
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

type RemovedPanelGhost = {
  panel: ConsolePanel;
  layout: ConsoleLayoutItem;
  diffItem: DraftConsoleDiffItem;
};

function buildRemovedGhosts(
  removedPanels: ConsolePanel[],
  removedLayout: ConsoleLayoutItem[],
  itemsById: Map<string, DraftConsoleDiffItem>,
): RemovedPanelGhost[] {
  const layoutById = new Map(removedLayout.map((item) => [item.i, item]));
  return removedPanels.flatMap((panel) => {
    const layout = layoutById.get(panel.id);
    const diffItem = itemsById.get(panel.id);
    if (!layout) return [];
    if (!diffItem) return [];
    return [{ panel, layout, diffItem }];
  });
}

function gridItemStyle(item: ConsoleLayoutItem, gridWidth: number): CSSProperties {
  const [marginX, marginY] = DASHBOARD_GRID_MARGIN;
  const [paddingX, paddingY] = DASHBOARD_CONTAINER_PADDING;
  const colWidth = (gridWidth - paddingX * 2 - marginX * (DASHBOARD_GRID_COLS - 1)) / DASHBOARD_GRID_COLS;

  return {
    left: paddingX + item.x * (colWidth + marginX),
    top: paddingY + item.y * (DASHBOARD_ROW_HEIGHT + marginY),
    width: item.w * colWidth + Math.max(0, item.w - 1) * marginX,
    height: item.h * DASHBOARD_ROW_HEIGHT + Math.max(0, item.h - 1) * marginY,
  };
}

function layoutBottomPx(layout: ConsoleLayoutItem[], gridWidth: number): number | undefined {
  if (gridWidth <= 0 || layout.length === 0) return undefined;
  return Math.max(
    ...layout.map((item) => {
      const style = gridItemStyle(item, gridWidth);
      return Number(style.top ?? 0) + Number(style.height ?? 0);
    }),
  );
}

type ConsoleVisualDiffState = ReturnType<typeof buildConsoleVisualDiffState>;

function filterRemovedDiffState(
  visualDiffState: ConsoleVisualDiffState,
  liveLayout: ConsoleLayoutItem[],
): ConsoleVisualDiffState {
  const replacedPanelIds = new Set(
    visualDiffState.removedLayout
      .filter((removedItem) => liveLayout.some((liveItem) => layoutItemsOverlap(liveItem, removedItem)))
      .map((item) => item.i),
  );

  if (replacedPanelIds.size === 0) return visualDiffState;

  return {
    ...visualDiffState,
    removedPanelIds: new Set([...visualDiffState.removedPanelIds].filter((id) => !replacedPanelIds.has(id))),
    removedPanels: visualDiffState.removedPanels.filter((panel) => !replacedPanelIds.has(panel.id)),
    removedLayout: visualDiffState.removedLayout.filter((item) => !replacedPanelIds.has(item.i)),
  };
}

function layoutItemsOverlap(
  a: Pick<ConsoleLayoutItem, "x" | "y" | "w" | "h">,
  b: Pick<ConsoleLayoutItem, "x" | "y" | "w" | "h">,
): boolean {
  return a.x < b.x + b.w && a.x + a.w > b.x && a.y < b.y + b.h && a.y + a.h > b.y;
}

function buildConsoleVisualDiffState(summary?: DraftConsoleDiffSummary) {
  const items = summary?.items ?? [];
  const removedItems = items.filter((item) => item.changeType === "removed");

  return {
    itemsById: new Map(items.map((item) => [item.id, item])),
    removedPanelIds: new Set(removedItems.map((item) => item.id)),
    removedPanels: removedItems.map((item) => normalizeDiffPanel(item)).filter((panel) => panel !== null),
    removedLayout: removedItems.map((item) => normalizeDiffLayout(item)).filter((item) => item !== null),
  };
}

function normalizeDiffPanel(item: DraftConsoleDiffItem): ConsolePanel | null {
  if (!item.panel) return null;
  return {
    id: item.panel.id || item.id,
    type: item.panel.type || "markdown",
    content: (item.panel.content as Record<string, unknown>) || {},
  };
}

function normalizeDiffLayout(item: DraftConsoleDiffItem): ConsoleLayoutItem | null {
  if (!item.layout) return null;
  return {
    i: item.layout.i || item.id,
    x: item.layout.x || 0,
    y: item.layout.y || 0,
    w: item.layout.w || 12,
    h: item.layout.h || 6,
    ...(item.layout.minW !== undefined ? { minW: item.layout.minW } : {}),
    ...(item.layout.minH !== undefined ? { minH: item.layout.minH } : {}),
  };
}

function toConsoleLayout(next: Layout[]): ConsoleLayoutItem[] {
  return next.map((item) => {
    const result: ConsoleLayoutItem = {
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
