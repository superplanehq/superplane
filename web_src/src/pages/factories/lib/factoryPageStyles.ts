import { cn } from "@/lib/utils";

/** Neutral page canvas — driven by the theme's --background token. */
export const factoryPageBackgroundClassName = "min-h-full w-full min-w-0 bg-background text-foreground";

/** Work order list row — flat divider, no floating card. */
export const factoryWorkOrderRowClassName = "border-b border-border py-5 last:border-b-0";

export const factoryFilterPillActiveClassName = "bg-accent text-foreground";

export const factoryFilterPillInactiveClassName = "text-muted-foreground hover:text-foreground";

export const factoryCountBadgeClassName =
  "rounded-full bg-accent px-2 py-0.5 text-xs font-medium text-muted-foreground";

export const factorySidebarActionClassName =
  "inline-flex items-center gap-1.5 text-sm font-medium text-muted-foreground hover:text-foreground";

export const factoryFormCardClassName = cn("rounded-lg border border-border bg-card px-6 py-8 sm:px-8");
