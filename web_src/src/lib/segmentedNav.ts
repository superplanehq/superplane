import { cn } from "@/lib/utils";

export type SegmentedNavSize = "default" | "xs";

export const SEGMENTED_NAV_CLASSES =
  "inline-flex h-7 min-h-7 items-center justify-center gap-0 rounded-full p-1 bg-slate-100 dark:bg-gray-800";

export const SEGMENTED_NAV_XS_CLASSES =
  "inline-flex h-6 min-h-6 items-center justify-center gap-0 rounded-full p-0.5 bg-slate-100 dark:bg-gray-800";

export const SEGMENTED_NAV_TAB_BASE_CLASSES =
  "inline-flex h-[calc(100%-1px)] flex-1 items-center justify-center gap-1.5 rounded-full border border-transparent font-medium whitespace-nowrap transition-[color,box-shadow] focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] focus-visible:outline-1 focus-visible:outline-ring disabled:pointer-events-none disabled:opacity-50";

const SEGMENTED_NAV_TAB_SIZE_CLASSES: Record<SegmentedNavSize, string> = {
  default: "px-2.5 py-1 text-[13px]",
  xs: "px-2 py-0.5 text-xs",
};

export const SEGMENTED_NAV_TAB_ACTIVE_CLASSES =
  "bg-white text-slate-900 shadow-sm dark:bg-gray-400 dark:text-gray-950 dark:shadow-none";

export const SEGMENTED_NAV_TAB_INACTIVE_CLASSES =
  "text-slate-500 hover:text-foreground dark:text-gray-400 dark:hover:text-gray-100";

export function segmentedNavClassName(size: SegmentedNavSize = "default") {
  return size === "xs" ? SEGMENTED_NAV_XS_CLASSES : SEGMENTED_NAV_CLASSES;
}

export function segmentedNavTabClassName(
  isActive: boolean,
  options?: {
    activeClasses?: string;
    inactiveClasses?: string;
    size?: SegmentedNavSize;
  },
) {
  const size = options?.size ?? "default";

  return cn(
    SEGMENTED_NAV_TAB_BASE_CLASSES,
    SEGMENTED_NAV_TAB_SIZE_CLASSES[size],
    isActive
      ? (options?.activeClasses ?? SEGMENTED_NAV_TAB_ACTIVE_CLASSES)
      : (options?.inactiveClasses ?? SEGMENTED_NAV_TAB_INACTIVE_CLASSES),
  );
}
