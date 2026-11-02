export const PLACEHOLDER_NODE_CONTEXT_KEY = "add-placeholder-node";

export function clearPendingPlaceholder(canvasId: string) {
  if (typeof window === "undefined") return;
  if (sessionStorage.getItem(PLACEHOLDER_NODE_CONTEXT_KEY) === canvasId) {
    sessionStorage.removeItem(PLACEHOLDER_NODE_CONTEXT_KEY);
  }
}
