import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn } from "@/lib/utils";

export const settingsCardClassName = cn("rounded-lg bg-gray-100/5 p-6", appDarkModeClasses.modalEdge);

export const settingsTableCardClassName = cn(
  "overflow-hidden rounded-lg bg-gray-100/5",
  appDarkModeClasses.modalEdge,
  "[&_th]:dark:border-gray-700/70 [&_td]:dark:border-gray-700/50",
);

export const settingsErrorClassName =
  "rounded border border-red-300 bg-white px-4 py-2 text-red-500 dark:border-red-800 dark:bg-red-900/20 dark:text-red-400";

export const settingsInnerMetricCardClassName = cn(
  "rounded-lg border border-slate-950/10 bg-gray-100/5 px-4 py-3",
  "dark:border-gray-700/70",
);

export const settingsPanelClassName = cn("rounded-md bg-gray-100/5", appDarkModeClasses.modalEdge);

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
