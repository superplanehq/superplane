export const CANVAS_AGENT_SIDEBAR_OPEN_EVENT = "canvas:agent-sidebar-open";
export const CANVAS_AGENT_SIDEBAR_CHANGED_EVENT = "canvas:agent-sidebar-changed";

type CanvasAgentSidebarDetail = {
  canvasId: string;
  open: boolean;
};

const lastAgentSidebarRequestByCanvas = new Map<string, boolean>();

export function requestCanvasAgentSidebarOpen(canvasId: string): void {
  requestCanvasAgentSidebarState(canvasId, true);
}

export function requestCanvasAgentSidebarClose(canvasId: string): void {
  requestCanvasAgentSidebarState(canvasId, false);
}

export function requestCanvasAgentSidebarState(canvasId: string, open: boolean): void {
  if (!canvasId) return;
  lastAgentSidebarRequestByCanvas.set(canvasId, open);
  dispatchCanvasAgentSidebarEvent(CANVAS_AGENT_SIDEBAR_OPEN_EVENT, canvasId, open);
}

export function lastCanvasAgentSidebarRequest(canvasId: string): boolean | undefined {
  return lastAgentSidebarRequestByCanvas.get(canvasId);
}

export function publishCanvasAgentSidebarChanged(canvasId: string, open: boolean): void {
  dispatchCanvasAgentSidebarEvent(CANVAS_AGENT_SIDEBAR_CHANGED_EVENT, canvasId, open);
}

export function subscribeCanvasAgentSidebarOpen(onOpen: (canvasId: string) => void): () => void {
  return subscribeCanvasAgentSidebarEvent(CANVAS_AGENT_SIDEBAR_OPEN_EVENT, (canvasId, open) => {
    if (open) onOpen(canvasId);
  });
}

export function subscribeCanvasAgentSidebarState(onChange: (canvasId: string, open: boolean) => void): () => void {
  return subscribeCanvasAgentSidebarEvent(CANVAS_AGENT_SIDEBAR_OPEN_EVENT, onChange);
}

export function subscribeCanvasAgentSidebarChanged(onChange: (canvasId: string, open: boolean) => void): () => void {
  return subscribeCanvasAgentSidebarEvent(CANVAS_AGENT_SIDEBAR_CHANGED_EVENT, onChange);
}

function dispatchCanvasAgentSidebarEvent(name: string, canvasId: string, open: boolean): void {
  if (!canvasId || typeof window === "undefined") return;
  window.dispatchEvent(new CustomEvent<CanvasAgentSidebarDetail>(name, { detail: { canvasId, open } }));
}

function subscribeCanvasAgentSidebarEvent(
  name: string,
  onChange: (canvasId: string, open: boolean) => void,
): () => void {
  if (typeof window === "undefined") return () => undefined;

  const listener = (event: Event) => {
    const detail = (event as CustomEvent<CanvasAgentSidebarDetail>).detail;
    if (!detail?.canvasId) return;
    onChange(detail.canvasId, detail.open);
  };

  window.addEventListener(name, listener);
  return () => window.removeEventListener(name, listener);
}
