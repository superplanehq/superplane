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

/** The app shell owns vertical scrolling. */
export const factoryContentBodyClassName = cn(
  "px-[var(--workspace-page-gutter)] py-[var(--workspace-page-padding-block)] text-foreground",
  contentColumn,
);

/** Full-pane shell for a GitHub-Projects-style kanban (no page scroll). */
export const factoryKanbanPageClassName = "flex h-full min-h-0 flex-col overflow-hidden";

/** Gutter + remaining height, no max-width, so lanes can grow to the right. */
export const factoryKanbanBodyClassName = cn(
  "px-[var(--workspace-page-gutter)] py-[var(--workspace-page-padding-block)] text-foreground",
  "flex min-h-0 flex-1 flex-col",
);

export const factoryCardClassName = "rounded-lg border border-border bg-card";

export const factoryPageTitleClassName = "workspace-page-title";

export const factoryPageSubtitleClassName = "workspace-body-text text-muted-foreground";
