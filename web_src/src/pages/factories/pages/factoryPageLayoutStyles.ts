import { cn } from "@/lib/utils";

const contentColumn = "mx-auto w-full max-w-[var(--workspace-content-max-width)]";

export const factoryContentHeaderClassName = cn(
  "bg-background px-[var(--workspace-page-gutter)] py-6",
  contentColumn,
  "flex flex-wrap items-center justify-between gap-3",
);

/** The app shell owns vertical scrolling. */
export const factoryContentBodyClassName = cn(
  "px-[var(--workspace-page-gutter)] py-[var(--workspace-page-padding-block)] text-foreground",
  contentColumn,
);

export const factoryCardClassName = "rounded-lg border border-border bg-card";

export const factoryPageTitleClassName = "workspace-page-title";

export const factoryPageSubtitleClassName = "workspace-body-text text-muted-foreground";
