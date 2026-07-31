/** Shared divider styling for canvas node body sections (metadata, specs, events, etc.). */
export const NODE_CANVAS_DIVIDER = "border-edge-strong";

export const nodeCanvasSectionDividerTop = `border-t ${NODE_CANVAS_DIVIDER}`;

export const nodeCanvasSectionDividerBottom = `border-b ${NODE_CANVAS_DIVIDER}`;

export const nodeCanvasMetadataSectionClassName =
  "px-2 py-1.5 border-b border-edge-strong text-content-secondary flex flex-col gap-1";

export const nodeCanvasSpecsSectionClassName = "px-2 py-1.5 text-content-secondary flex flex-col gap-1.5";

export const nodeCanvasChannelLabelClassName =
  "text-xs font-medium whitespace-nowrap absolute bg-surface-subtle text-content-muted";

/** Muted metadata on event sections (timestamp, event id). */
export const eventSectionMetadataTextClassName = "text-content-muted";
