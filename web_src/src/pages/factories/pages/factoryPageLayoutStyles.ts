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

/** Shared left/right inset so section titles and page bodies share one edge. */
const sectionPaneGutter = "px-3";

/**
 * Compact section header (Overview, Tasks, Lines, Automations, Wiki, Velocity).
 * Title size matches the sidebar workspace name. Top inset matches the
 * workspace switcher (`pt-3`). Full pane width — not the centered column.
 */
export const factorySectionHeaderClassName = cn(
  "mx-0 max-w-none pt-3 pb-3",
  sectionPaneGutter,
  "[--workspace-page-title-size:var(--workspace-section-title-size)]",
  "[--workspace-page-title-line-height:var(--workspace-section-title-line-height)]",
  "[--workspace-page-title-tracking:var(--workspace-section-title-tracking)]",
  "[&_.workspace-page-title]:font-medium",
);

/** Section list/body: same gutter as the header, no extra left inset. */
export const factorySectionBodyClassName = cn("mx-0 max-w-none pt-0 pb-6 text-foreground", sectionPaneGutter);

/**
 * Centered variants for report pages such as Velocity. The section title style
 * stays the same as the other workspace pages, but the header and the body
 * share one centered column so charts keep a readable width on wide screens.
 * The header uses a larger top inset than the full-width page header, because a
 * centered column has no sidebar next to it to set the vertical rhythm.
 */
export const factoryCenteredSectionHeaderClassName = cn(factorySectionHeaderClassName, contentColumn, "pt-14");

export const factoryCenteredSectionBodyClassName = cn(factorySectionBodyClassName, contentColumn);

/** Tasks / Lines kanban body: same gutter, fills leftover height. */
export const factoryWorkOrdersBodyClassName = cn(
  factorySectionBodyClassName,
  "max-w-full pb-3",
  "flex min-h-0 min-w-0 w-full flex-1 flex-col overflow-hidden",
);

/** The app shell owns vertical scrolling. */
export const factoryContentBodyClassName = cn(
  "px-[var(--workspace-page-gutter)] py-[var(--workspace-page-padding-block)] text-foreground",
  contentColumn,
);

/** Full-pane shell for a GitHub-Projects-style kanban (no page scroll). */
export const factoryKanbanPageClassName = "flex h-full min-h-0 min-w-0 w-full max-w-full flex-col overflow-hidden";

export const factoryCardClassName = "rounded-lg border border-border bg-card";

export const factoryPageTitleClassName = "workspace-page-title";

export const factoryPageSubtitleClassName = "workspace-body-text text-muted-foreground";
