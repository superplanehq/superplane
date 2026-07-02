export const CANVAS_TOOL_SIDEBAR_SELECT_TAB_EVENT = "canvas-tool-sidebar:select-tab";

export type CanvasToolSidebarTab = "agent";

export function openCanvasToolSidebarTab(tab: CanvasToolSidebarTab) {
  window.dispatchEvent(new CustomEvent(CANVAS_TOOL_SIDEBAR_SELECT_TAB_EVENT, { detail: { tab } }));
}

export function canvasToolSidebarTabFromEvent(event: Event): CanvasToolSidebarTab | null {
  const tab = (event as CustomEvent<{ tab?: CanvasToolSidebarTab }>).detail?.tab;
  if (tab === "agent") return tab;
  return null;
}
