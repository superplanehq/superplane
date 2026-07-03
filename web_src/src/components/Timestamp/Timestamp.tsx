import React, { useEffect, useReducer } from "react";
import { twMerge } from "tailwind-merge";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { CopyButton } from "@/ui/CopyButton";
import { formatAbsolute, formatISO, formatRelative, formatUTC, toDate, type TimestampInput } from "@/lib/datetime";

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
 * Live-updating relative label. Uses `formatRelative` (not `TimeAgo`) so future
 * timestamps render as "in …" instead of being clamped to "0s ago".
 */
function RelativeLabel({ date, iso }: { date: Date; iso: string }) {
  const [, tick] = useReducer((n: number) => n + 1, 0);
  useEffect(() => subscribeRelativeTick(tick), []);
  return <time dateTime={iso}>{formatRelative(date)}</time>;
}

interface TimestampProps {
  /** Accepts a `Date`, ISO string, or epoch milliseconds. */
  date: TimestampInput | null | undefined;
  /**
   * Controls the visible label:
   * - `"absolute"` (default): locale date-time in the user's timezone.
   * - `"relative"`: live-updating "5m ago" style text.
   */
  display?: "absolute" | "relative";
  /** Render the dashed underline affordance that hints at the hover details. Default `true`. */
  withHint?: boolean;
  className?: string;
  /** Alignment of the hover card relative to the trigger. */
  align?: "start" | "center" | "end";
  /** Rendered when the date is missing/invalid. */
  fallback?: React.ReactNode;
}

function DetailRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <>
      <dt className="text-right font-medium text-gray-500 dark:text-gray-400">{label}</dt>
      <dd className="min-w-0 text-gray-800 dark:text-gray-100">{children}</dd>
    </>
  );
}

/**
 * Standardized timestamp display: a locale-aware label with a dashed underline
 * hint that reveals absolute (UTC), relative, and raw ISO values on hover,
 * with a copy button for the ISO value.
 */
export const Timestamp = React.memo(function Timestamp({
  date,
  display = "absolute",
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
          {display === "relative" ? (
            <RelativeLabel date={resolved} iso={iso} />
          ) : (
            <time dateTime={iso}>{formatAbsolute(resolved)}</time>
          )}
        </span>
      </HoverCardTrigger>
      <HoverCardContent align={align} className="w-auto max-w-sm p-3">
        <dl className="grid grid-cols-[auto_1fr] items-center gap-x-4 gap-y-1.5 text-sm">
          <DetailRow label="UTC">{formatUTC(resolved)}</DetailRow>
          <DetailRow label="Relative">{formatRelative(resolved)}</DetailRow>
          <DetailRow label="Timestamp">
            <div className="flex items-center gap-1.5">
              <span className="min-w-0 truncate font-mono text-xs">{iso}</span>
              <CopyButton text={iso} data-testid="timestamp-copy" />
            </div>
          </DetailRow>
        </dl>
      </HoverCardContent>
    </HoverCard>
  );
});
