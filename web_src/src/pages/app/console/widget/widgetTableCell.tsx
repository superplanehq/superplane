import { useState, type ReactElement } from "react";
import { ArrowDownRight, ArrowUpRight, Minus, User } from "lucide-react";

import { formatTimestampInUserTimezone } from "@/lib/timezone";
import { cn } from "@/lib/utils";

import { evaluateRowShow } from "./rowVisibility";
import { resolveCellValue } from "./resolveCellValue";
import { resolveHref } from "./resolveHref";
import type { WidgetColumnFormat, WidgetTableRender, WidgetTrendGoodDirection } from "./types";
import { formatTrend, formatValue } from "./widgetFormat";

const STATUS_PILL_CLASS: Record<string, string> = {
  passed: "bg-emerald-500 text-white",
  ready: "bg-emerald-500 text-white",
  active: "bg-emerald-500 text-white",
  "very low": "bg-emerald-500 text-white",
  low: "bg-emerald-500 text-white",
  failed: "bg-red-500 text-white",
  critical: "bg-red-500 text-white",
  high: "bg-orange-500 text-white",
  running: "bg-blue-500 text-white",
  medium: "bg-yellow-500 text-white",
  cancelled: "bg-gray-500 text-white",
  pending: "bg-gray-500 text-white",
  idle: "bg-gray-500 text-white",
};

const STATUS_PILL_BASE_CLASS = "inline-flex rounded-full border-none px-2 py-0.5 text-[11px] font-medium";

const BADGE_PILL_CLASS =
  "inline-flex rounded-full bg-transparent px-2 py-0.5 text-[11px] font-medium text-slate-700 outline outline-1 -outline-offset-1 outline-slate-950/15 dark:text-gray-300 dark:outline-gray-600";

type CellColumn = WidgetTableRender["columns"][number];

interface CellContext {
  col: CellColumn;
  value: unknown;
  formatted: string;
}

/**
 * Per-format cell renderers. `link` (and any column with an `href`) is handled
 * separately in `Cell` because it applies across formats; everything here is
 * keyed purely by `col.format`. Formats with no entry fall through to the plain
 * text cell.
 */
const FORMAT_CELL_RENDERERS: Partial<Record<WidgetColumnFormat, (ctx: CellContext) => ReactElement>> = {
  // `badge` is for neutral tags (service names, categories) with a light
  // outlined treatment.
  badge: ({ formatted }) => (
    <td className="px-3 py-1.5">
      <span className={BADGE_PILL_CLASS}>{formatted}</span>
    </td>
  ),
  // `status` renders semantic values (passed, failed, risk levels) as colored pills.
  status: ({ formatted }) => {
    const toneClass = STATUS_PILL_CLASS[formatted.toLowerCase()] ?? "bg-gray-500 text-white";
    return (
      <td className="px-3 py-1.5">
        <span className={cn(STATUS_PILL_BASE_CLASS, toneClass)}>{formatted}</span>
      </td>
    );
  },
  relative: ({ value, formatted }) => (
    <td className="px-3 py-1.5 text-slate-700 dark:text-gray-300" title={formatAbsoluteTitle(value)}>
      {formatted}
    </td>
  ),
  code: ({ formatted }) => (
    <td className="px-3 py-1.5">
      <code className="rounded bg-slate-100 px-1 py-0.5 font-mono text-[13px] text-slate-800 dark:bg-gray-800 dark:text-gray-100">
        {formatted}
      </code>
    </td>
  ),
  progress: ({ value, formatted }) => <ProgressCell value={value} label={formatted} />,
  trend: ({ col, value }) => <TrendCell value={value} goodDirection={col.goodDirection ?? "up"} />,
  avatar: ({ col, value }) => <AvatarCell value={value} alt={col.label ?? col.field} />,
};

export function Cell({ col, row }: { col: CellColumn; row: Record<string, unknown> }) {
  const visible = evaluateRowShow(col.show, row);
  if (!visible) {
    return <td className="px-3 py-1.5 text-slate-300 dark:text-gray-600">—</td>;
  }
  const value = resolveCellValue(col.field, row);
  const formatted = formatValue(value, col.format);
  if (col.format === "link" || col.href) {
    const href = col.href ? resolveHref(col.href, row) : String(value ?? "");
    return (
      <td className="px-3 py-1.5">
        <a
          href={href}
          target="_blank"
          rel="noopener noreferrer"
          className="text-sky-600 no-underline hover:!underline underline-offset-2 decoration-current dark:text-gray-300 dark:hover:text-gray-100"
        >
          {formatted || href}
        </a>
      </td>
    );
  }
  const renderer = col.format ? FORMAT_CELL_RENDERERS[col.format] : undefined;
  if (renderer) {
    return renderer({ col, value, formatted });
  }
  return <td className="px-3 py-1.5 text-slate-700 dark:text-gray-300">{formatted}</td>;
}

/**
 * `avatar` renders the resolved cell value (an image URL) as a small circular
 * image. When the URL is empty or fails to load, it falls back to a neutral
 * circular placeholder with a `User` icon so the column stays aligned.
 */
function AvatarCell({ value, alt }: { value: unknown; alt: string }) {
  const [errored, setErrored] = useState(false);
  const src = typeof value === "string" ? value.trim() : "";

  if (src === "" || errored) {
    return (
      <td className="px-3 py-1.5">
        <span
          className="inline-flex size-6 items-center justify-center rounded-full bg-slate-100 text-slate-400 dark:bg-gray-800 dark:text-gray-500"
          data-testid="widget-table-avatar"
          aria-label={alt}
        >
          <User className="size-3.5" aria-hidden />
        </span>
      </td>
    );
  }

  return (
    <td className="px-3 py-1.5">
      <img
        src={src}
        alt={alt}
        loading="lazy"
        onError={() => setErrored(true)}
        className="size-6 rounded-full object-cover"
        data-testid="widget-table-avatar"
      />
    </td>
  );
}

function ProgressCell({ value, label }: { value: unknown; label: string }) {
  return (
    <td className="px-3 py-1.5">
      <div className="flex items-center gap-2">
        <div className="h-1.5 w-24 overflow-hidden rounded-full bg-slate-100 dark:bg-gray-800">
          <div
            className="h-full rounded-full bg-sky-500 transition-all dark:bg-sky-400"
            style={{ width: `${(progressFraction(value) * 100).toFixed(1)}%` }}
          />
        </div>
        <span className="text-[11px] tabular-nums text-slate-500 dark:text-gray-400">{label}</span>
      </div>
    </td>
  );
}

/**
 * `trend` renders a signed change value as a colored delta indicator following
 * the scorecard's language. The arrow always follows the real movement (up for
 * a rise, down for a fall, dash for no change); `goodDirection` decides which
 * movement is painted green (good) vs red (bad), so metrics like open issues or
 * latency can treat a drop as an improvement.
 */
function TrendCell({ value, goodDirection }: { value: unknown; goodDirection: WidgetTrendGoodDirection }) {
  const n = typeof value === "number" ? value : Number(value);
  const direction = !Number.isFinite(n) || n === 0 ? "flat" : n > 0 ? "up" : "down";
  const Icon = direction === "up" ? ArrowUpRight : direction === "down" ? ArrowDownRight : Minus;
  const tone =
    direction === "flat"
      ? "text-slate-500 dark:text-gray-400"
      : direction === goodDirection
        ? "text-emerald-600 dark:text-emerald-400"
        : "text-red-600 dark:text-red-400";
  return (
    <td className="px-3 py-1.5">
      <span className={cn("inline-flex items-center gap-1 text-sm font-medium tabular-nums", tone)}>
        <Icon className="size-4" aria-hidden />
        {formatTrend(value)}
      </span>
    </td>
  );
}

/**
 * Clamp a cell value to a 0–1 fill fraction for the `progress` format. Values
 * in `(0, 1]` are treated as fractions; anything larger is read as a percentage
 * (e.g. `92` -> `0.92`), matching the `percent` label convention.
 */
function progressFraction(value: unknown): number {
  const n = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(n)) return 0;
  const fraction = n > 0 && n <= 1 ? n : n / 100;
  return Math.max(0, Math.min(1, fraction));
}

function formatAbsoluteTitle(value: unknown): string | undefined {
  if (value == null) return undefined;
  if (typeof value === "string" && value.trim() !== "") {
    const parsed = Date.parse(value);
    if (Number.isFinite(parsed)) return formatTimestampInUserTimezone(new Date(parsed).toISOString());
  }
  const n = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(n)) return undefined;
  const ms = n > 1e12 ? n : n * 1000;
  return formatTimestampInUserTimezone(new Date(ms).toISOString());
}
