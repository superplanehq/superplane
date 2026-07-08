/** Shared divider styling for canvas node body sections (metadata, specs, events, etc.). */
export const NODE_CANVAS_DIVIDER = "border-slate-950/20 dark:border-gray-600/70";

export const nodeCanvasSectionDividerTop = `border-t ${NODE_CANVAS_DIVIDER}`;

export const nodeCanvasSectionDividerBottom = `border-b ${NODE_CANVAS_DIVIDER}`;

export const nodeCanvasMetadataSectionClassName =
  "px-2 py-1.5 border-b border-slate-950/20 dark:border-gray-600/70 text-gray-500 flex flex-col gap-1 dark:text-gray-400";

export const nodeCanvasSpecsSectionClassName = "px-2 py-1.5 text-gray-500 flex flex-col gap-1.5 dark:text-gray-400";

export const nodeCanvasChannelLabelClassName =
  "text-xs font-medium whitespace-nowrap absolute bg-slate-100 text-[#8B9AAC] dark:bg-gray-900 dark:text-gray-400";

/** Muted metadata on event sections (timestamp, event id). */
export const eventSectionMetadataTextClassName = "text-gray-950/50 dark:text-white/50";
