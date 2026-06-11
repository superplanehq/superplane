/** Fit options for the full live workflow graph (sidebar "Live Canvas"). */
export const LIVE_CANVAS_FIT_VIEW_OPTIONS = {
  maxZoom: 1.0,
  padding: 0.08,
} as const;

/** Fit options when framing run participant nodes during run inspection. */
export const RUN_CANVAS_FIT_VIEW_OPTIONS = {
  maxZoom: 1.2,
  minZoom: 0.85,
  padding: 0.1,
} as const;
