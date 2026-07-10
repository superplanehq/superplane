import React from "react";

import { CopyButton } from "@/ui/CopyButton";
import { formatAbsolute, formatISO, formatRelative, formatUTC, toDate, type TimestampInput } from "@/lib/datetime";

interface TimestampDetailsProps {
  /** Accepts a `Date`, ISO string, or epoch milliseconds. */
  date: TimestampInput | null | undefined;
  /** Rendered when the date is missing/invalid. */
  fallback?: React.ReactNode;
  /** Optional test id for the copy button (kept stable across surfaces). */
  copyTestId?: string;
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
 * Presentational multi-format timestamp grid: Local, UTC, Relative, and raw
 * ISO with a copy affordance. Extracted so both the shared `Timestamp` hover
 * card and other tooltip surfaces (e.g. chart tooltips) render an identical
 * details block.
 */
export function TimestampDetails({ date, fallback = null, copyTestId = "timestamp-copy" }: TimestampDetailsProps) {
  const resolved = toDate(date);
  if (!resolved) return <>{fallback}</>;
  const iso = formatISO(resolved);
  return (
    <dl className="grid grid-cols-[auto_1fr] items-center gap-x-4 gap-y-1.5 text-sm">
      <DetailRow label="Local">{formatAbsolute(resolved)}</DetailRow>
      <DetailRow label="UTC">{formatUTC(resolved)}</DetailRow>
      <DetailRow label="Relative">{formatRelative(resolved)}</DetailRow>
      <DetailRow label="Timestamp">
        <div className="flex items-center gap-1.5">
          <span className="min-w-0 truncate font-mono text-xs">{iso}</span>
          <CopyButton text={iso} data-testid={copyTestId} />
        </div>
      </DetailRow>
    </dl>
  );
}
