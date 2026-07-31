import { withEventStatusBadgeClasses } from "@/lib/eventStatusBadge";
import { cn } from "@/lib/utils";

/**
 * Shared status/tag chip styling for console panels. Matches the uppercase
 * colored badges used in run details and org settings
 * (`text-[10px] font-semibold uppercase tracking-wide` on a solid fill).
 */
export const CONSOLE_BADGE_BASE_CLASSES =
  "inline-flex shrink-0 items-center justify-center whitespace-nowrap rounded px-1 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-content-inverse";

const CONSOLE_STATUS_BADGE_COLOR: Record<string, string> = {
  passed: "bg-emerald-500",
  deployed: "bg-emerald-500",
  ready: "bg-emerald-500",
  active: "bg-emerald-500",
  "very low": "bg-emerald-500",
  low: "bg-emerald-500",
  failed: "bg-red-500",
  critical: "bg-red-500",
  high: "bg-orange-500",
  running: "bg-blue-500",
  medium: "bg-yellow-500",
  cancelled: "bg-content-muted",
  pending: "bg-content-muted",
  idle: "bg-content-muted",
  unknown: "bg-content-muted",
};

/** Resolve the full Tailwind class list for a console badge label. */
export function consoleBadgeClassName(value: string, colorClass?: string): string {
  const key = value.trim().toLowerCase();
  const color = colorClass ?? CONSOLE_STATUS_BADGE_COLOR[key] ?? "bg-content-muted";
  return cn(CONSOLE_BADGE_BASE_CLASSES, withEventStatusBadgeClasses(color));
}
