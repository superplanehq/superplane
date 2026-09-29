import { CANVAS_NODE_FOCUS_FIT_VIEW_OPTIONS } from "@/ui/CanvasPage/canvasFitOptions";

/** Wait for the popup pane to get a real size, then fit the graph. */
export const COMPACT_CANVAS_FIT_SETTLE_MS = 200;

/** Fit the whole compact graph when no node is selected. */
export const COMPACT_FIT_VIEW_OPTIONS = { padding: 0.2 } as const;

/** Zoom the compact canvas onto one selected node. */
export const COMPACT_CANVAS_NODE_FOCUS_FIT_VIEW_OPTIONS = {
  ...CANVAS_NODE_FOCUS_FIT_VIEW_OPTIONS,
  padding: 0.35,
  duration: 400,
} as const;

/** Stable key so the compact canvas remounts and fits when live nodes arrive. */
export function compactCanvasFitKey(nodeIds: string[]): string {
  if (nodeIds.length === 0) {
    return "empty";
  }
  return [...nodeIds].sort().join("|");
}

export function shouldFitCompactCanvas(contentKey: string): boolean {
  return contentKey !== "empty";
}

export function compactCanvasNodeFocusRequest<T extends { id: string }>(node: T | undefined) {
  if (!node) {
    return null;
  }
  return { nodes: [node], ...COMPACT_CANVAS_NODE_FOCUS_FIT_VIEW_OPTIONS };
}
