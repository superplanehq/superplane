import { cn } from "@/lib/utils";

/** Standard top header inside the main content column. */
export const factoryContentHeaderClassName = cn(
  "flex flex-wrap items-center justify-between gap-3 border-b border-slate-950/10 bg-white px-6 py-4 dark:border-gray-700/70 dark:bg-gray-900",
);

/** Padding for the scrollable body area under the content header. */
export const factoryContentBodyClassName = "flex-1 overflow-y-auto px-6 py-6 sm:px-8";
