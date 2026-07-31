import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn } from "@/lib/utils";

/** Neutral page canvas — warm gray in light mode, not cool slate. */
export const factoryPageBackgroundClassName = cn(
  "min-h-full w-full min-w-0 bg-gray-50",
  appDarkModeClasses.surface,
);

/** Full-width page content with edge padding only — no max-width column. */
export const factoryPageContentClassName = "w-full min-w-0 px-6 py-8 sm:px-8";

/** Primary content panel wrapping work orders + sidebar. */
export const factoryDetailPanelClassName = cn(
  "w-full min-w-0 overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-gray-700/70 dark:bg-gray-900",
);

export const factoryDetailMainClassName = "min-w-0 px-6 py-6 sm:px-8";

export const factoryDetailSidebarClassName = cn(
  "border-t border-gray-200 px-6 py-6 dark:border-gray-700/70 lg:border-t-0 lg:border-l lg:px-6",
);

/** Work order list row — flat divider, no floating card. */
export const factoryWorkOrderRowClassName = cn(
  "border-b border-gray-200 py-5 last:border-b-0 dark:border-gray-700/70",
);

export const factoryFilterPillActiveClassName =
  "bg-gray-200/80 text-gray-900 dark:bg-gray-800 dark:text-gray-100";

export const factoryFilterPillInactiveClassName =
  "text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-100";

export const factoryCountBadgeClassName =
  "rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-600 dark:bg-gray-800 dark:text-gray-300";

export const factorySidebarActionClassName =
  "inline-flex items-center gap-1.5 text-sm font-medium text-gray-600 hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-100";

export const factoryFormCardClassName = cn(
  "rounded-lg border border-gray-200 bg-white px-6 py-8 dark:border-gray-700/70 dark:bg-gray-900 sm:px-8",
);

/** @deprecated Use factoryWorkOrderRowClassName — kept for detail pages until migrated. */
export const factoryCardClassName = cn(
  "rounded-lg border border-gray-200 bg-white dark:border-gray-700/70 dark:bg-gray-900",
);
