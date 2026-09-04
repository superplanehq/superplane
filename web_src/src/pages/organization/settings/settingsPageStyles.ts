import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn } from "@/lib/utils";

export const settingsCardClassName = "rounded-lg border border-border bg-card p-6";

export const settingsTableCardClassName =
  "overflow-hidden rounded-lg border border-border bg-card [&_th]:border-border [&_td]:border-border";

export const settingsErrorClassName =
  "rounded border border-destructive/40 bg-destructive/10 px-4 py-2 text-destructive";

export const settingsInnerMetricCardClassName = "rounded-lg border border-border bg-muted/40 px-4 py-3";

export const settingsPanelClassName = "rounded-md border border-border bg-card";

export const settingsModalClassName = cn(
  "mx-4 w-full rounded-lg bg-white shadow-xl dark:bg-gray-900",
  appDarkModeClasses.modalEdge,
);

export const settingsEmptyStateIconClassName = "text-gray-800 dark:text-gray-100";

export const settingsEmptyStateTitleClassName = "mt-3 text-sm text-gray-800 dark:text-gray-100";

export const settingsEmptyStateSubtitleClassName = "mt-1 text-xs text-gray-500 dark:text-gray-400";

export const settingsTableLinkClassName =
  "cursor-pointer text-sm !font-semibold text-gray-800 !underline underline-offset-2 dark:text-gray-100";

export const settingsRowActionClassName =
  "rounded-sm p-1 text-gray-800 transition-colors hover:bg-gray-100 dark:text-gray-100 dark:hover:bg-gray-700/50";

export const settingsRowMenuClassName = "flex items-center gap-2 text-sm text-gray-800 dark:text-gray-100";
