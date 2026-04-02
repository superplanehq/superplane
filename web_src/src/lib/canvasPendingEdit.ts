const storageKeyForCanvas = (canvasId: string) => `sp_canvas_pending_edit:${canvasId}`;
const storageKeyForComponentsSidebar = (canvasId: string) => `sp_canvas_pending_components_sidebar:${canvasId}`;

/**
 * Call before navigating to a newly created canvas so WorkflowPageV2 can open edit mode
 * and the building-blocks (components) sidebar once data is ready.
 */
export function markCanvasPendingOpenInEditMode(canvasId: string): void {
  if (typeof window === "undefined") {
    return;
  }
  sessionStorage.setItem(storageKeyForCanvas(canvasId), "1");
  sessionStorage.setItem(storageKeyForComponentsSidebar(canvasId), "1");
}

/** Returns true once per pending flag (sessionStorage), then clears it. */
export function consumeCanvasPendingOpenInEditMode(canvasId: string): boolean {
  if (typeof window === "undefined") {
    return false;
  }
  const key = storageKeyForCanvas(canvasId);
  if (sessionStorage.getItem(key) !== "1") {
    return false;
  }
  sessionStorage.removeItem(key);
  return true;
}

/** Consumed synchronously on first paint with canvas data so CanvasPage initializes with the sidebar open. */
export function consumeCanvasPendingOpenComponentsSidebar(canvasId: string): boolean {
  if (typeof window === "undefined") {
    return false;
  }
  const key = storageKeyForComponentsSidebar(canvasId);
  if (sessionStorage.getItem(key) !== "1") {
    return false;
  }
  sessionStorage.removeItem(key);
  return true;
}
