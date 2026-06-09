import { useMemo, useState } from "react";
import GridLayout, { type Layout, WidthProvider } from "react-grid-layout";

import { cn } from "@/lib/utils";
import type { ConsoleLayoutItem, ConsolePanel } from "@/hooks/useCanvasData";
import type { DraftConsoleDiffItem, DraftConsoleDiffSummary } from "../draftConsoleDiff";

import { ChartPanelCard } from "./ChartPanelCard";
import { ConsolePanelDiffBadge, ConsolePanelDiffDialog } from "./ConsolePanelDiff";
import { consolePanelDiffBorderClassName } from "./consolePanelDiffPresentation";
import { MarkdownPanelCard } from "./MarkdownPanelCard";
import { NodePanelCard } from "./NodePanelCard";
import { NodesPanelCard } from "./NodesPanelCard";
import { NumberPanelCard } from "./NumberPanelCard";
import { TablePanelCard } from "./TablePanelCard";
import { useConsoleGridTransitionArming } from "./useConsoleGridTransitionArming";

import "react-grid-layout/css/styles.css";
import "./console-grid.css";

const ResponsiveGridLayout = WidthProvider(GridLayout);

const DASHBOARD_GRID_COLS = 12;
const DASHBOARD_ROW_HEIGHT = 40;
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
  const visualDiffState = useMemo(
    () => buildConsoleVisualDiffState(visualDiff?.enabled ? visualDiff.summary : undefined),
    [visualDiff?.enabled, visualDiff?.summary],
  );
  const visibleRemovedDiffState = useMemo(
    () => filterRemovedDiffState(visualDiffState, layout),
    [layout, visualDiffState],
  );
  const renderedPanels = useMemo(
    () => [...panels, ...visibleRemovedDiffState.removedPanels],
    [panels, visibleRemovedDiffState.removedPanels],
  );
  const renderedLayout = useMemo(
    () => [...layout, ...visibleRemovedDiffState.removedLayout],
    [layout, visibleRemovedDiffState.removedLayout],
  );
  const layoutItems = useMemo(
    () => buildRGLLayout(renderedPanels, renderedLayout, visibleRemovedDiffState.removedPanelIds),
    [renderedPanels, renderedLayout, visibleRemovedDiffState.removedPanelIds],
  );
  const { transitionsArmed, gridWrapperRef } = useConsoleGridTransitionArming(renderedPanels.length > 0);

  return (
    <div className="flex h-full w-full flex-col overflow-auto">
      <div ref={gridWrapperRef} className="px-4 py-3">
        <ResponsiveGridLayout
          className={cn("console-grid", !transitionsArmed && "console-grid--instant")}
          layout={layoutItems}
          cols={DASHBOARD_GRID_COLS}
          rowHeight={DASHBOARD_ROW_HEIGHT}
          margin={[12, 12]}
          containerPadding={[0, 0]}
          isDraggable={!readOnly}
          isResizable={!readOnly}
          draggableHandle=".console-grid-drag-handle"
          draggableCancel=".console-grid-no-drag"
          resizeHandles={["se"]}
          onLayoutChange={(next) => {
            if (readOnly) return;
            onLayoutChange(toConsoleLayout(next).filter((item) => !visualDiffState.removedPanelIds.has(item.i)));
          }}
          useCSSTransforms
          compactType="vertical"
          preventCollision={false}
        >
          {renderedPanels.map((panel) => {
            const diffItem = visualDiffState.itemsById.get(panel.id);
            const isRemoved = diffItem?.changeType === "removed";

            return (
              <div
                key={panel.id}
                className={cn(
                  "console-grid-item group/console-panel-diff relative rounded-lg",
                  isRemoved && "opacity-50",
                )}
              >
                <PanelCardRouter
                  panel={panel}
                  readOnly={readOnly || isRemoved}
                  onDelete={() => onDeletePanel(panel.id)}
                  onChange={(content) => onPanelContentChange(panel.id, content)}
                />
                {diffItem ? (
                  <div
                    className={cn(
                      "pointer-events-none absolute inset-0 z-[1] rounded-lg border-2",
                      consolePanelDiffBorderClassName(diffItem.changeType),
                    )}
                    data-testid="console-panel-diff-border"
                  />
                ) : null}
                {diffItem ? (
                  <ConsolePanelDiffBadge
                    status={diffItem.changeType}
                    panelTitle={diffItem.title || panel.id}
                    onShowDiff={() => setSelectedDiffItem(diffItem)}
                  />
                ) : null}
              </div>
            );
          })}
        </ResponsiveGridLayout>
      </div>
      <ConsolePanelDiffDialog
        item={selectedDiffItem}
        onOpenChange={(open) => {
          if (!open) setSelectedDiffItem(null);
        }}
      />
    </div>
  );
}

function buildRGLLayout(
  panels: ConsolePanel[],
  layout: ConsoleLayoutItem[],
  inactivePanelIds = new Set<string>(),
): Layout[] {
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
        isDraggable: !inactivePanelIds.has(panel.id),
        isResizable: !inactivePanelIds.has(panel.id),
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
      isDraggable: !inactivePanelIds.has(panel.id),
      isResizable: !inactivePanelIds.has(panel.id),
    });
    nextY += 6;
  }
  return result;
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

function PanelCardRouter({
  panel,
  readOnly,
  onDelete,
  onChange,
}: {
  panel: ConsolePanel;
  readOnly: boolean;
  onDelete: () => void;
  onChange: (content: Record<string, unknown>) => void;
}) {
  switch (panel.type) {
    case "node":
      return <NodePanelCard panel={panel} readOnly={readOnly} onDelete={onDelete} onChange={onChange} />;
    case "nodes":
      return <NodesPanelCard panel={panel} readOnly={readOnly} onDelete={onDelete} onChange={onChange} />;
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
