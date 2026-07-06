import { cn } from "@/lib/utils";

export const createActionCardClassName = cn(
  "relative flex w-full flex-row items-center gap-4 rounded-md border border-dashed border-green-500 bg-green-50 px-4 py-3 transition-colors",
  "dark:border-green-500 dark:bg-green-950/30 hover:bg-green-100 dark:hover:bg-green-950/50",
);

export const createActionCardDisabledClassName = cn(
  "relative flex w-full flex-row items-center gap-4 rounded-md border border-dashed border-slate-300 bg-slate-200/70 px-4 py-3 text-slate-500 transition-colors cursor-not-allowed",
  "dark:border-slate-600 dark:bg-slate-800/50",
);

export const createActionIconClassName =
  "flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-green-500 text-white dark:bg-green-300 dark:text-green-950";

export const createActionIconDisabledClassName =
  "flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-slate-400 text-white dark:bg-slate-500";
