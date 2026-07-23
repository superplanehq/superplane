import type { WidgetBoardLaneColor } from "./types";

/**
 * Compact palette for board lane headers. Kept as a decoupled tone-to-
 * class map so YAML using `color: green` keeps working across future
 * Tailwind refactors. Missing / unknown colors render as `neutral`.
 *
 * `header` styles a lane's title strip; `strip` colors the left border
 * so lanes read at a glance even when scrolled sideways.
 */
export const BOARD_LANE_STYLE: Record<WidgetBoardLaneColor, { header: string; strip: string; badge: string }> = {
  neutral: {
    header: "bg-slate-50 text-slate-700 dark:bg-gray-800/60 dark:text-gray-200",
    strip: "border-l-slate-300 dark:border-l-gray-600",
    badge: "bg-slate-200 text-slate-700 dark:bg-gray-700 dark:text-gray-200",
  },
  gray: {
    header: "bg-slate-100 text-slate-800 dark:bg-gray-800 dark:text-gray-100",
    strip: "border-l-slate-400 dark:border-l-gray-500",
    badge: "bg-slate-300 text-slate-800 dark:bg-gray-600 dark:text-gray-100",
  },
  blue: {
    header: "bg-sky-50 text-sky-800 dark:bg-sky-950/40 dark:text-sky-100",
    strip: "border-l-sky-400 dark:border-l-sky-400",
    badge: "bg-sky-200 text-sky-800 dark:bg-sky-900 dark:text-sky-100",
  },
  green: {
    header: "bg-emerald-50 text-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-100",
    strip: "border-l-emerald-400 dark:border-l-emerald-400",
    badge: "bg-emerald-200 text-emerald-800 dark:bg-emerald-900 dark:text-emerald-100",
  },
  yellow: {
    header: "bg-yellow-50 text-yellow-800 dark:bg-yellow-950/40 dark:text-yellow-100",
    strip: "border-l-yellow-400 dark:border-l-yellow-400",
    badge: "bg-yellow-200 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-100",
  },
  orange: {
    header: "bg-orange-50 text-orange-800 dark:bg-orange-950/40 dark:text-orange-100",
    strip: "border-l-orange-400 dark:border-l-orange-400",
    badge: "bg-orange-200 text-orange-800 dark:bg-orange-900 dark:text-orange-100",
  },
  red: {
    header: "bg-red-50 text-red-800 dark:bg-red-950/40 dark:text-red-100",
    strip: "border-l-red-400 dark:border-l-red-400",
    badge: "bg-red-200 text-red-800 dark:bg-red-900 dark:text-red-100",
  },
  purple: {
    header: "bg-purple-50 text-purple-800 dark:bg-purple-950/40 dark:text-purple-100",
    strip: "border-l-purple-400 dark:border-l-purple-400",
    badge: "bg-purple-200 text-purple-800 dark:bg-purple-900 dark:text-purple-100",
  },
};

export function laneStyleFor(color: WidgetBoardLaneColor | undefined) {
  return color ? BOARD_LANE_STYLE[color] : BOARD_LANE_STYLE.neutral;
}
