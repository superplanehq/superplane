import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn } from "@/lib/utils";

/**
 * gray-600/gray-300 rather than the gray-500/gray-400 pairing used elsewhere:
 * both of those fail WCAG 4.5:1 — 500 on white (4.2:1) and 400 on the raised
 * dark surface (4.38:1).
 */
export const mutedTextClassName = "text-gray-600 dark:text-gray-300";

/** PRD: responsive width with a restrained cap near 1,500–1,600px. */
export const factoryPageWidthClassName = "mx-auto w-full max-w-[1600px] px-8";

/** PRD: the Work Order chronology may use a narrower reading column. */
export const workOrderPageWidthClassName = "mx-auto w-full max-w-3xl px-8";

export const factoryPanelClassName = cn(
  "rounded-lg bg-white p-5 outline outline-slate-950/10",
  appDarkModeClasses.surfaceRaised,
  "dark:outline-gray-700/70",
);

export const factoryCardClassName = cn(
  "rounded-md bg-white outline outline-slate-950/10",
  appDarkModeClasses.surfaceRaised,
  "dark:outline-gray-700/70",
);

export const sectionTitleClassName = "text-sm font-semibold text-slate-900 dark:text-gray-100";

export const countPillClassName =
  "inline-flex min-w-5 items-center justify-center rounded-full bg-slate-100 px-1.5 text-xs font-medium text-gray-700 dark:bg-gray-800 dark:text-gray-300";
