import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn } from "@/lib/utils";

export const homePageTitleClassName = "text-xl font-medium text-slate-900 dark:text-gray-100";

export const homeModalTitleClassName = "text-lg font-medium text-slate-900 dark:text-gray-100";

export const homePageSubtitleClassName = "mt-2 text-sm font-medium text-gray-500 dark:text-gray-400";

export const homeCardTitleClassName = "text-base font-medium text-slate-900 dark:text-gray-100";

export const homePanelTitleClassName = "text-sm font-semibold text-slate-900 dark:text-gray-100";

export const homeListCardClassName = cn(
  "rounded-md bg-white outline outline-slate-950/10 transition-colors hover:outline-slate-950/20",
  appDarkModeClasses.surfaceRaised,
  "dark:outline-gray-700/70 dark:hover:outline-gray-600/40",
);

export const homeDividerLineClassName = "w-full border-t border-slate-950/10 dark:border-gray-700/70";

export const homeDividerLabelClassName =
  "bg-slate-100 px-3 text-sm font-medium text-gray-500 dark:bg-gray-900 dark:text-gray-400";

export const homeSectionDividerClassName = "border-t border-slate-100 dark:border-gray-700/70";

export const homeInstallPanelClassName = cn(
  "mt-4 rounded-lg bg-white p-5 outline outline-slate-950/10 animate-in slide-in-from-top-2",
  appDarkModeClasses.surfaceRaised,
  "dark:outline-gray-700/70",
);

export const homeModalOverlayClassName = "fixed inset-0 bg-gray-950/20 dark:bg-black/50";

export const homeModalPanelClassName = cn(
  "relative z-10 w-[calc(100vw-2rem)] max-w-3xl rounded-xl bg-white shadow-2xl dark:bg-gray-900",
  "dark:border dark:border-gray-700/70",
);

export const homeModalHeaderEdgeClassName =
  "flex items-center gap-2 border-b border-slate-200 px-5 py-3 dark:border-gray-700/70";

export const homeModalFooterEdgeClassName =
  "flex items-center justify-between border-t border-slate-200 px-6 py-4 dark:border-gray-700/70";

export const homeTagClassName =
  "rounded-full bg-slate-100 px-2 py-0.5 text-[10px] font-medium text-slate-500 dark:bg-gray-800 dark:text-gray-400";

export const homeTagLargeClassName =
  "rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-medium text-slate-600 dark:bg-gray-800 dark:text-gray-300";
