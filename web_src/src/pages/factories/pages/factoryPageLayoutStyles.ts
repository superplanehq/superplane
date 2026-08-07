import { cn } from "@/lib/utils";

/**
 * Shared frame utilities for /factories pages. These aim to match the
 * v3 aesthetic: token-driven surfaces, a centered max-width column, and
 * a tighter typographic scale. Every page renders inside the same
 * `contentColumn` so widths stay visually consistent from Overview to
 * Settings.
 */

const contentColumn = "mx-auto w-full min-w-[720px] max-w-[960px]";

/** Standard top header inside the main content column. */
export const factoryContentHeaderClassName = cn(
  "bg-background px-8 py-6",
  contentColumn,
  "flex flex-wrap items-center justify-between gap-3",
);

/** Padding for the scrollable body area under the content header. */
export const factoryContentBodyClassName = cn("flex-1 overflow-y-auto px-8 py-8 text-foreground", contentColumn);

/** Consistent card surface used across Overview / WorkOrders / Automations / Settings. */
export const factoryCardClassName = "rounded-lg border border-border bg-card";

/** Page title style used at the top of every page. */
export const factoryPageTitleClassName = "text-[22px] font-semibold tracking-[-0.02em] text-foreground";

/** Subtitle rendered directly beneath a page title. */
export const factoryPageSubtitleClassName = "text-[13px] text-muted-foreground";
