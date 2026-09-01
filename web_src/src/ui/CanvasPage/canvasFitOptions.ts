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
