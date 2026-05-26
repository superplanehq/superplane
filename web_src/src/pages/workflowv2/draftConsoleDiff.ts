import type { CanvasesCanvasDashboard, CanvasesDashboardLayoutItem, CanvasesDashboardPanel } from "@/api-client";

function comparablePanels(panels: CanvasesDashboardPanel[] | undefined): unknown[] {
  return (panels ?? [])
    .map((panel) => ({
      id: panel.id ?? "",
      type: panel.type ?? "",
      content: panel.content ?? {},
    }))
    .sort((left, right) => left.id.localeCompare(right.id));
}

function comparableLayout(layout: CanvasesDashboardLayoutItem[] | undefined): unknown[] {
  return (layout ?? [])
    .map((item) => ({
      i: item.i ?? "",
      x: item.x ?? 0,
      y: item.y ?? 0,
      w: item.w ?? 0,
      h: item.h ?? 0,
      ...(item.minW !== undefined ? { minW: item.minW } : {}),
      ...(item.minH !== undefined ? { minH: item.minH } : {}),
    }))
    .sort((left, right) => left.i.localeCompare(right.i));
}

function comparableConsoleSnapshot(dashboard?: CanvasesCanvasDashboard | null): string {
  return JSON.stringify({
    panels: comparablePanels(dashboard?.panels),
    layout: comparableLayout(dashboard?.layout),
  });
}

/** True when draft console differs from live (panels and/or layout). */
export function hasDraftVersusLiveConsoleDiff(
  liveDashboard?: CanvasesCanvasDashboard | null,
  draftDashboard?: CanvasesCanvasDashboard | null,
): boolean {
  return comparableConsoleSnapshot(liveDashboard) !== comparableConsoleSnapshot(draftDashboard);
}
