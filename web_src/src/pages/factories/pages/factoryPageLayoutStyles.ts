import { cn } from "@/lib/utils";

const contentColumn = "mx-auto w-full max-w-[var(--workspace-content-max-width)]";

/**
 * Shell for the workspace page header: gutter, max width, vertical rhythm.
 * `WorkspacePageHeader` builds on this. Only reach for the class directly
 * when you cannot use the component (very rare — most pages should not).
 */
export const factoryContentHeaderClassName = cn(
  "bg-background px-[var(--workspace-page-gutter)] pt-10 pb-6",
  contentColumn,
  "flex flex-col gap-3",
);

/** Shared left/right inset so the Work Orders title and board share one edge. */
const workOrdersPaneGutter = "px-3";

/**
 * Compact Work Orders list header. Title size matches the sidebar workspace
 * name (13–15px, medium). Top inset matches the workspace switcher (`pt-3`).
 * Full pane width, same gutter as the board — not the centered content column.
 */
export const factoryWorkOrdersHeaderClassName = cn(
  "mx-0 max-w-none pt-3 pb-3",
  workOrdersPaneGutter,
  "[--workspace-page-title-size:var(--workspace-section-title-size)]",
  "[--workspace-page-title-line-height:var(--workspace-section-title-line-height)]",
  "[--workspace-page-title-tracking:var(--workspace-section-title-tracking)]",
  "[&_.workspace-page-title]:font-medium",
);

/** Work Orders body: same gutter as the header, no extra left inset. */
export const factoryWorkOrdersBodyClassName = cn(
  "mx-0 max-w-full pt-0 pb-3 text-foreground",
  workOrdersPaneGutter,
  "flex min-h-0 min-w-0 w-full flex-1 flex-col overflow-hidden",
);

/** The app shell owns vertical scrolling. */
export const factoryContentBodyClassName = cn(
  "px-[var(--workspace-page-gutter)] py-[var(--workspace-page-padding-block)] text-foreground",
  contentColumn,
);

/** Full-pane shell for a GitHub-Projects-style kanban (no page scroll). */
export const factoryKanbanPageClassName = "flex h-full min-h-0 min-w-0 w-full max-w-full flex-col overflow-hidden";

/** Gutter + remaining height, no max-width, so lanes can grow to the right. */
export const factoryKanbanBodyClassName = cn(
  "px-[var(--workspace-page-gutter)] py-[var(--workspace-page-padding-block)] text-foreground",
  "flex min-h-0 flex-1 flex-col",
);

export const factoryCardClassName = "rounded-lg border border-border bg-card";

export const factoryPageTitleClassName = "workspace-page-title";

export const factoryPageSubtitleClassName = "workspace-body-text text-muted-foreground";
