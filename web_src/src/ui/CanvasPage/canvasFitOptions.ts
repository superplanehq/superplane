/** Include culled nodes when framing the canvas; viewport culling sets `hidden` on off-screen nodes. */
export const CANVAS_FIT_VIEW_INCLUDE_HIDDEN = {
  includeHiddenNodes: true,
} as const;

/** Fit options for the full live workflow graph (sidebar "Live Canvas"). */
export const LIVE_CANVAS_FIT_VIEW_OPTIONS = {
  ...CANVAS_FIT_VIEW_INCLUDE_HIDDEN,
  maxZoom: 1.0,
  padding: 0.08,
} as const;

/** Frame at 100% zoom. Do not shrink the graph to fit the viewport. */
export const NATIVE_ZOOM_FIT_VIEW_OPTIONS = {
  ...CANVAS_FIT_VIEW_INCLUDE_HIDDEN,
  minZoom: 1.0,
  maxZoom: 1.0,
  padding: 0.08,
} as const;

/** Fit at 100% zoom when Factory Configure opens. Do not shrink the graph. */
export const FACTORY_CONFIGURE_FIT_VIEW_OPTIONS = NATIVE_ZOOM_FIT_VIEW_OPTIONS;

/** First-load fit: lock 100% zoom for display previews, otherwise shrink to fit. */
export function resolveInitialCanvasFitViewOptions(lockNativeZoom: boolean) {
  return lockNativeZoom ? NATIVE_ZOOM_FIT_VIEW_OPTIONS : LIVE_CANVAS_FIT_VIEW_OPTIONS;
}

type NodeBox = {
  position: { x: number; y: number };
  width?: number | null;
  height?: number | null;
  measured?: { width?: number | null; height?: number | null };
};

/** Center every measured node at 100% zoom. Padding does not apply: zoom stays at 1. */
export function nativeZoomViewport(
  nodes: ReadonlyArray<NodeBox>,
  width: number,
  height: number,
): { x: number; y: number; zoom: number } | null {
  let minX = Infinity;
  let minY = Infinity;
  let maxX = -Infinity;
  let maxY = -Infinity;
  for (const node of nodes) {
    const nodeWidth = node.measured?.width ?? node.width ?? 0;
    const nodeHeight = node.measured?.height ?? node.height ?? 0;
    if (!nodeWidth || !nodeHeight) continue;
    minX = Math.min(minX, node.position.x);
    minY = Math.min(minY, node.position.y);
    maxX = Math.max(maxX, node.position.x + nodeWidth);
    maxY = Math.max(maxY, node.position.y + nodeHeight);
  }
  if (!Number.isFinite(minX) || maxX <= minX || maxY <= minY) return null;
  return {
    x: width / 2 - (minX + (maxX - minX) / 2),
    y: height / 2 - (minY + (maxY - minY) / 2),
    zoom: 1,
  };
}

const DEFAULT_FIT_VIEW_DURATION_MS = 500;

/** Factory display and Configure enter skip the fit animation so nodes do not slide. */
export function resolveInitialFitViewDuration(factoryDisplayLayout: boolean, factoryConfigure: boolean): number {
  if (factoryDisplayLayout) {
    return 0;
  }
  if (factoryConfigure) {
    return 0;
  }
  return DEFAULT_FIT_VIEW_DURATION_MS;
}

/** Fit options when framing run participant nodes during run inspection. */
export const RUN_CANVAS_FIT_VIEW_OPTIONS = {
  ...CANVAS_FIT_VIEW_INCLUDE_HIDDEN,
  maxZoom: 1.2,
  padding: 0.1,
} as const;

/** Fit options when centering on a single node (search, focus, agent chip). */
export const CANVAS_NODE_FOCUS_FIT_VIEW_OPTIONS = {
  ...CANVAS_FIT_VIEW_INCLUDE_HIDDEN,
  maxZoom: 1.2,
} as const;
