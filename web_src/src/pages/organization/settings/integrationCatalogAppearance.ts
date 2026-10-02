import { cn } from "@/lib/utils";
import { factoryCardClassName } from "@/pages/factories/pages/factoryPageLayoutStyles";
import {
  settingsEmptyStateIconClassName,
  settingsEmptyStateTitleClassName,
  settingsPanelClassName,
} from "./settingsPageStyles";

export type CatalogAppearance = "legacy" | "factories";

export function catalogAppearance(appearance: CatalogAppearance) {
  if (appearance === "factories") {
    return {
      root: undefined,
      searchIcon: "absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground",
      clearButton: "absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground",
      clearIcon: "size-4",
      emptyIcon: "mx-auto mb-2 size-6 text-muted-foreground",
      emptyTitle: "text-[13px] text-foreground",
      requestWrap: "mt-3 text-[13px] text-muted-foreground",
      requestButton: "font-medium text-foreground underline underline-offset-2",
      requestPrompt: "Cannot find your integration? ",
      list: "space-y-3",
      card: `${factoryCardClassName} p-4`,
      cardHeader: "flex items-start justify-between gap-4",
      icon: "size-8 text-muted-foreground",
      title: "text-[13px] font-medium text-foreground",
      description: "mt-0.5 text-[13px] text-muted-foreground",
      instancesWrap: "mt-3 pl-[44px]",
      instanceCount: "mb-2 text-[12px] text-muted-foreground",
      instanceRow: "flex items-center gap-2 border-t border-border py-1.5",
      instanceName: "truncate text-[13px] font-medium text-foreground",
      plugReady: "text-green-600",
      plugError: "text-destructive",
      plugOther: "text-amber-600",
      statusLabel: "text-[12px] font-medium text-muted-foreground",
      statusLabelWrap: "w-16 shrink-0",
      footerRequest: "mt-6 text-center text-[13px] text-muted-foreground",
    };
  }

  return {
    root: "pt-6",
    searchIcon: "absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-500 dark:text-gray-400",
    clearButton:
      "absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200",
    clearIcon: "h-4 w-4",
    emptyIcon: cn("mx-auto mb-2 h-6 w-6", settingsEmptyStateIconClassName),
    emptyTitle: settingsEmptyStateTitleClassName,
    requestWrap: "mt-3 text-sm text-gray-500 dark:text-gray-400",
    requestButton: "font-medium text-blue-600 hover:underline dark:text-blue-400",
    requestPrompt: "Can't find your integration? ",
    list: "space-y-4",
    card: settingsPanelClassName,
    cardHeader: "flex items-start justify-between gap-4 p-4",
    icon: "h-8 w-8 text-gray-500 dark:text-gray-400",
    title: "text-sm font-semibold text-gray-800 dark:text-gray-100",
    description: "mt-0.5 text-sm text-gray-800 dark:text-gray-400",
    instancesWrap: "pr-4 pb-4 pl-[60px]",
    instanceCount: "mb-2 text-xs text-gray-500 dark:text-gray-400",
    instanceRow: "flex items-center gap-2 border-t border-gray-200 py-1.5 dark:border-gray-700/70",
    instanceName: "truncate text-sm font-medium text-gray-800 dark:text-gray-100",
    plugReady: "text-green-500",
    plugError: "text-red-500",
    plugOther: "text-amber-600",
    statusLabel: "inline-flex w-fit items-center rounded px-1.5 py-0.5 text-xs font-medium",
    statusLabelWrap: "min-w-16 shrink-0",
    footerRequest: "mt-6 text-center text-sm text-gray-500 dark:text-gray-400",
  };
}

export function instancePlugClass(state: string | undefined, styles: ReturnType<typeof catalogAppearance>) {
  if (state === "ready") {
    return styles.plugReady;
  }
  if (state === "error") {
    return styles.plugError;
  }
  return styles.plugOther;
}

export function instanceStatusLabelClass(
  appearance: CatalogAppearance,
  state: string | undefined,
  styles: ReturnType<typeof catalogAppearance>,
) {
  if (appearance === "factories") {
    return styles.statusLabel;
  }
  if (state === "ready") {
    return cn(styles.statusLabel, "bg-white text-green-500 dark:bg-green-300 dark:text-green-950");
  }
  if (state === "error") {
    return cn(styles.statusLabel, "bg-white text-red-500 dark:bg-red-300 dark:text-red-950");
  }
  return cn(styles.statusLabel, "bg-white text-amber-600 dark:bg-amber-300 dark:text-amber-950");
}
