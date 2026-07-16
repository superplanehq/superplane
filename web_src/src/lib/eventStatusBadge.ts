/**
 * Literal class strings so Tailwind detects dark: utilities (dynamic template strings are not scanned).
 * Light classes are preserved; dark: classes are additive only.
 */
import { cn } from "@/lib/utils";

export const EVENT_STATUS_BADGE_BASE_CLASSES =
  "inline-flex shrink-0 items-center justify-center rounded px-[5px] py-[1.5px] text-[10px] font-semibold uppercase tracking-wide leading-normal text-white";

const EVENT_STATUS_BADGE_CLASSES: Record<string, string> = {
  "bg-amber-500": "bg-amber-500 dark:bg-amber-400 dark:text-amber-950",
  "bg-blue-500": "bg-blue-500 dark:bg-blue-400 dark:text-blue-950",
  "bg-emerald-500": "bg-emerald-500 dark:bg-emerald-400 dark:text-emerald-950",
  "bg-gray-400": "bg-gray-400 dark:bg-gray-400 dark:text-gray-950",
  "bg-gray-500": "bg-gray-500 dark:bg-gray-400 dark:text-gray-950",
  "bg-green-500": "bg-green-500 dark:bg-green-400 dark:text-green-950",
  "bg-indigo-500": "bg-indigo-500 dark:bg-indigo-400 dark:text-indigo-950",
  "bg-orange-500": "bg-orange-500 dark:bg-orange-400 dark:text-orange-950",
  "bg-red-400": "bg-red-400 dark:bg-red-400 dark:text-red-950",
  "bg-red-500": "bg-red-500 dark:bg-red-400 dark:text-red-950",
  "bg-violet-400": "bg-violet-400 dark:bg-violet-400 dark:text-violet-950",
  "bg-yellow-500": "bg-yellow-500 dark:bg-yellow-400 dark:text-yellow-950",
  "bg-yellow-600": "bg-yellow-600 dark:bg-yellow-400 dark:text-yellow-950",
};

const DEFAULT_EVENT_STATUS_BADGE_CLASSES = "dark:bg-gray-400 dark:text-gray-950";

export function withEventStatusBadgeClasses(badgeColor: string): string {
  const trimmed = badgeColor.trim();
  if (/\bdark:/.test(trimmed)) {
    return trimmed;
  }

  const mapped = EVENT_STATUS_BADGE_CLASSES[trimmed];
  if (mapped) {
    return mapped;
  }

  return `${trimmed} ${DEFAULT_EVENT_STATUS_BADGE_CLASSES}`;
}

export function eventStatusBadgeClassName(badgeColor: string): string {
  return cn(EVENT_STATUS_BADGE_BASE_CLASSES, withEventStatusBadgeClasses(badgeColor));
}
