import React, { useEffect, useReducer } from "react";
import { twMerge } from "tailwind-merge";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { formatTimeAgo } from "@/lib/date";
import { formatAbsolute, formatDate, formatISO, formatRelative, toDate, type TimestampInput } from "@/lib/datetime";
import { TimestampDetails } from "./TimestampDetails";

// A single shared 1s ticker drives every relative label so a page with many
// timestamps doesn't spin up an interval per instance (mirrors `TimeAgo`).
const relativeTickListeners = new Set<() => void>();
let relativeTickInterval: ReturnType<typeof setInterval> | null = null;

function subscribeRelativeTick(listener: () => void): () => void {
  relativeTickListeners.add(listener);
  if (!relativeTickInterval) {
    relativeTickInterval = setInterval(() => {
      relativeTickListeners.forEach((cb) => cb());
    }, 1000);
  }
  return () => {
    relativeTickListeners.delete(listener);
    if (relativeTickInterval && relativeTickListeners.size === 0) {
      clearInterval(relativeTickInterval);
      relativeTickInterval = null;
    }
  };
}

/**
 * Live-updating relative label driven by the shared 1s ticker.
 *
 * - `"full"` uses `formatRelative` so future timestamps render as "in …"
 *   ("in 3 hours", "5 minutes ago").
 * - `"abbreviated"` reuses `formatTimeAgo` (the helper `TimeAgo` uses) so dense
 *   rows keep their compact "5m" / "5m ago" / "in 3h" text.
 */
function RelativeLabel({
  date,
  iso,
  relativeStyle,
  includeAgo,
}: {
  date: Date;
  iso: string;
  relativeStyle: "full" | "abbreviated";
  includeAgo: boolean;
}) {
  const [, tick] = useReducer((n: number) => n + 1, 0);
  useEffect(() => subscribeRelativeTick(tick), []);
  const label = relativeStyle === "abbreviated" ? formatTimeAgo(date, includeAgo) : formatRelative(date);
  return <time dateTime={iso}>{label}</time>;
}

/**
 * Controls the visible label:
 * - `"absolute"` / `"datetime"`: locale date-time in the user's timezone.
 * - `"date"`: locale date-only in the user's timezone (no time-of-day).
 * - `"relative"`: live-updating "5m ago" style text.
 *
 * The hover card content is identical across all display modes.
 */
export type TimestampDisplay = "absolute" | "datetime" | "date" | "relative";

interface TimestampProps {
  /** Accepts a `Date`, ISO string, or epoch milliseconds. */
  date: TimestampInput | null | undefined;
  /** Visible label style. Defaults to `"absolute"`. `"datetime"` is an alias. */
  display?: TimestampDisplay;
  /**
   * Style of the relative label (only applies when `display="relative"`):
   * - `"full"` (default): verbose Intl text, e.g. "5 minutes ago" / "in 3 hours".
   * - `"abbreviated"`: compact text for dense rows, e.g. "5m" / "5m ago".
   */
  relativeStyle?: "full" | "abbreviated";
  /**
   * Whether the abbreviated relative label includes the "ago" suffix, e.g. "5m ago"
   * vs "5m". Only applies when `display="relative"` and `relativeStyle="abbreviated"`.
   * Default `true`.
   */
  includeAgo?: boolean;
  /** Render the dashed underline affordance that hints at the hover details. Default `true`. */
  withHint?: boolean;
  className?: string;
  /** Alignment of the hover card relative to the trigger. */
  align?: "start" | "center" | "end";
  /** Rendered when the date is missing/invalid. */
  fallback?: React.ReactNode;
}

/**
 * Standardized timestamp display: a locale-aware label with a dashed underline
 * hint that reveals local absolute, relative, UTC, and raw ISO values on hover,
 * with a copy button for the ISO value.
 */
export const Timestamp = React.memo(function Timestamp({
  date,
  display = "absolute",
  relativeStyle = "full",
  includeAgo = true,
  withHint = true,
  className,
  align = "start",
  fallback = null,
}: TimestampProps) {
  const resolved = toDate(date);
  if (!resolved) return <>{fallback}</>;

  const iso = formatISO(resolved);
  const hintClasses = withHint
    ? "underline decoration-dashed decoration-gray-300 dark:decoration-gray-600 underline-offset-2 cursor-default"
    : "cursor-default";

  return (
    <HoverCard openDelay={150} closeDelay={100}>
      <HoverCardTrigger asChild>
        <span className={twMerge(hintClasses, className)}>
          <TimestampLabel
            date={resolved}
            iso={iso}
            display={display}
            relativeStyle={relativeStyle}
            includeAgo={includeAgo}
          />
        </span>
      </HoverCardTrigger>
      <HoverCardContent align={align} className="w-auto max-w-sm p-3">
        <TimestampDetails date={resolved} />
      </HoverCardContent>
    </HoverCard>
  );
});

function TimestampLabel({
  date,
  iso,
  display,
  relativeStyle,
  includeAgo,
}: {
  date: Date;
  iso: string;
  display: TimestampDisplay;
  relativeStyle: "full" | "abbreviated";
  includeAgo: boolean;
}) {
  if (display === "relative") {
    return <RelativeLabel date={date} iso={iso} relativeStyle={relativeStyle} includeAgo={includeAgo} />;
  }
  if (display === "date") {
    return <time dateTime={iso}>{formatDate(date)}</time>;
  }
  return <time dateTime={iso}>{formatAbsolute(date)}</time>;
}
