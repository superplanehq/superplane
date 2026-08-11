import { cn } from "@/lib/utils";

/**
 * Shared frame utilities for /factories pages. These aim to match the
 * v3 aesthetic: token-driven surfaces, a centered max-width column, and
 * a tighter typographic scale. Every page renders inside the same
 * `contentColumn` so widths stay visually consistent from Overview to
 * Settings.
 */

const contentColumn = "mx-auto w-full max-w-[var(--workspace-content-max-width)]";

/** Standard top header inside the main content column. */
export const factoryContentHeaderClassName = cn(
  "bg-background px-[var(--workspace-page-gutter)] py-6",
  contentColumn,
  "flex flex-wrap items-center justify-between gap-3",
);

/**
 * Padding for the body area under the content header.
 *
 * Note: we intentionally do NOT set `flex-1 overflow-y-auto` here — the app
 * shell (AppRouter's outer wrapper) already owns page-level scrolling, so
 * making the body its own scroll container just produces a second scrollbar
 * (page + main content). Content grows naturally and the outer scroll takes
 * over past the viewport.
 */
export const factoryContentBodyClassName = cn(
  "px-[var(--workspace-page-gutter)] py-[var(--workspace-page-padding-block)] text-foreground",
  contentColumn,
);

/** Consistent card surface used across Overview / WorkOrders / Automations / Settings. */
export const factoryCardClassName = "rounded-lg border border-border bg-card";

/** Page title style used at the top of every page. */
export const factoryPageTitleClassName = "workspace-page-title";

/** Subtitle rendered directly beneath a page title. */
export const factoryPageSubtitleClassName = "workspace-body-text text-muted-foreground";
