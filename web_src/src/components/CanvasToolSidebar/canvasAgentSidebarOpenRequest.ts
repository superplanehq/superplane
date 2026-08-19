export const CANVAS_AGENT_SIDEBAR_OPEN_EVENT = "canvas:agent-sidebar-open";

type CanvasAgentSidebarOpenDetail = {
  canvasId: string;
};

export function requestCanvasAgentSidebarOpen(canvasId: string): void {
  if (!canvasId || typeof window === "undefined") return;
  window.dispatchEvent(
    new CustomEvent<CanvasAgentSidebarOpenDetail>(CANVAS_AGENT_SIDEBAR_OPEN_EVENT, { detail: { canvasId } }),
  );
}

export function subscribeCanvasAgentSidebarOpen(onOpen: (canvasId: string) => void): () => void {
  if (typeof window === "undefined") return () => undefined;

  const listener = (event: Event) => {
    const canvasId = (event as CustomEvent<CanvasAgentSidebarOpenDetail>).detail?.canvasId;
    if (canvasId) onOpen(canvasId);
  };

  window.addEventListener(CANVAS_AGENT_SIDEBAR_OPEN_EVENT, listener);
  return () => window.removeEventListener(CANVAS_AGENT_SIDEBAR_OPEN_EVENT, listener);
}
