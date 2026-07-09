import { useMemo } from "react";
import { ArrowDownRight, ArrowUpRight, Loader2, Minus, Target } from "lucide-react";

import { cn } from "@/lib/utils";

import { WidgetEmptyState } from "../WidgetEmptyState";

import { formatValue } from "./widgetFormat";
import type { WidgetColumnFormat } from "./types";

/**
 * Prototype `scorecard` panel renderer. Elevates a single KPI (the Number
 * panel) into a "how are we doing vs a goal?" card: a hero value colored by a
 * threshold/target status, a direction-aware trend delta, an optional
 * sparkline, and an optional progress-to-target bar.
 *
 * Pure and presentational — the aggregated `value` (and any `comparison` /
 * `sparkline` series) are passed in, so a future data-bound panel could feed
 * this from `useWidgetData` without touching the renderer.
 */
export type ScorecardGoalDirection = "higher" | "lower";
export type ScorecardStatus = "good" | "warn" | "bad" | "neutral";
/** How the change chip renders the delta: percent, an absolute number, or both. */
export type ScorecardTrendDisplay = "percent" | "absolute" | "both";

/** One threshold boundary. `at` is the value where `status` begins to apply. */
export interface ScorecardThreshold {
  at: number;
  status: "good" | "warn" | "bad";
}

/** Baseline the current value is compared against to compute the trend delta. */
export interface ScorecardComparison {
  value: number;
  label?: string;
}

interface WidgetScorecardProps {
  value: number | null;
  label?: string;
  format?: WidgetColumnFormat;
  prefix?: string;
  suffix?: string;
  /** Whether a higher value is better. Drives all status and delta coloring. */
  goalDirection?: ScorecardGoalDirection;
  /** Simple mode: a single goal line. Enables the progress bar. */
  target?: number;
  /** Multi-band mode: good / warn / bad ranges. Takes precedence over `target`. */
  thresholds?: ScorecardThreshold[];
  comparison?: ScorecardComparison;
  /** How the change chip renders the delta. Defaults to `"percent"`. */
  trendDisplay?: ScorecardTrendDisplay;
  sparkline?: number[];
  showProgress?: boolean;
  isLoading?: boolean;
}

const STATUS_VALUE_CLASS: Record<ScorecardStatus, string> = {
  good: "text-emerald-600 dark:text-emerald-400",
  warn: "text-amber-500 dark:text-amber-400",
  bad: "text-red-600 dark:text-red-400",
  neutral: "text-slate-900 dark:text-gray-100",
};

const STATUS_BAR_CLASS: Record<ScorecardStatus, string> = {
  good: "bg-emerald-500",
  warn: "bg-amber-500",
  bad: "bg-red-500",
  neutral: "bg-slate-400 dark:bg-gray-500",
};

const STATUS_SPARK_CLASS: Record<ScorecardStatus, string> = {
  good: "text-emerald-500",
  warn: "text-amber-500",
  bad: "text-red-500",
  neutral: "text-sky-500 dark:text-gray-400",
};

const STATUS_DOT_CLASS: Record<ScorecardStatus, string> = {
  good: "bg-emerald-500",
  warn: "bg-amber-500",
  bad: "bg-red-500",
  neutral: "bg-slate-300 dark:bg-gray-600",
};

/**
 * Resolve the scorecard status from the configured thresholds/target.
 *
 * Thresholds win when present: sorted by boundary, the band whose `at` the
 * value has crossed wins (ascending crossings for higher-is-better, descending
 * for lower-is-better). Otherwise a single `target` yields good/bad based on
 * whether the value meets the goal in the desired direction.
 */
function resolveStatus(
  value: number,
  direction: ScorecardGoalDirection,
  target: number | undefined,
  thresholds: ScorecardThreshold[] | undefined,
): ScorecardStatus {
  if (thresholds && thresholds.length > 0) {
    const ordered = [...thresholds].sort((a, b) => (direction === "higher" ? a.at - b.at : b.at - a.at));
    let current: ScorecardStatus = "neutral";
    for (const band of ordered) {
      const crossed = direction === "higher" ? value >= band.at : value <= band.at;
      if (crossed) current = band.status;
    }
    return current;
  }
  if (target != null) {
    const meets = direction === "higher" ? value >= target : value <= target;
    return meets ? "good" : "bad";
  }
  return "neutral";
}

interface Delta {
  pct: number;
  raw: number;
  trend: "up" | "down" | "flat";
  improving: boolean;
}

const FLAT_THRESHOLD = 0.005;

function resolveDelta(value: number, direction: ScorecardGoalDirection, comparison: ScorecardComparison): Delta {
  const raw = value - comparison.value;
  const pct = comparison.value === 0 ? 0 : raw / Math.abs(comparison.value);
  if (Math.abs(pct) < FLAT_THRESHOLD) {
    return { pct, raw, trend: "flat", improving: false };
  }
  const trend = raw > 0 ? "up" : "down";
  const improving = direction === "higher" ? raw > 0 : raw < 0;
  return { pct, raw, trend, improving };
}

export function WidgetScorecard({
  value,
  goalDirection = "higher",
  target,
  thresholds,
  comparison,
  isLoading = false,
  ...rest
}: WidgetScorecardProps) {
  const status = useMemo<ScorecardStatus>(
    () => (value == null ? "neutral" : resolveStatus(value, goalDirection, target, thresholds)),
    [value, goalDirection, target, thresholds],
  );
  const delta = useMemo<Delta | null>(
    () => (value == null || !comparison ? null : resolveDelta(value, goalDirection, comparison)),
    [value, goalDirection, comparison],
  );

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <Loader2 className="size-4 animate-spin text-slate-400 dark:text-gray-500" />
      </div>
    );
  }

  if (value == null) {
    return <WidgetEmptyState icon={Target} message="No data to display." testId="widget-scorecard-empty" />;
  }

  return (
    <ScorecardBody {...rest} value={value} target={target} status={status} delta={delta} comparison={comparison} />
  );
}

interface ScorecardBodyProps {
  value: number;
  label?: string;
  format?: WidgetColumnFormat;
  prefix?: string;
  suffix?: string;
  target?: number;
  sparkline?: number[];
  showProgress?: boolean;
  status: ScorecardStatus;
  delta: Delta | null;
  comparison?: ScorecardComparison;
  trendDisplay?: ScorecardTrendDisplay;
}

function ScorecardBody({
  value,
  label,
  format = "number",
  prefix,
  suffix,
  target,
  sparkline,
  showProgress = false,
  status,
  delta,
  comparison,
  trendDisplay = "percent",
}: ScorecardBodyProps) {
  const hasSparkline = sparkline != null && sparkline.length > 1;

  return (
    <div className="flex h-full flex-col justify-center gap-2.5 p-4" data-testid="widget-scorecard">
      <ScorecardHeader label={label} status={status} />

      <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
        <span className={cn("text-4xl font-semibold leading-none", STATUS_VALUE_CLASS[status])}>
          {prefix ?? ""}
          {formatValue(value, format)}
          {suffix ? <span className="text-xl font-medium">{suffix}</span> : null}
        </span>
        {delta ? (
          <DeltaChip
            delta={delta}
            comparison={comparison}
            display={trendDisplay}
            format={format}
            prefix={prefix}
            suffix={suffix}
          />
        ) : null}
      </div>

      {hasSparkline ? <Sparkline values={sparkline} className={STATUS_SPARK_CLASS[status]} /> : null}

      {showProgress ? (
        <ScorecardProgress
          value={value}
          target={target}
          status={status}
          format={format}
          prefix={prefix}
          suffix={suffix}
        />
      ) : null}
    </div>
  );
}

function ScorecardHeader({ label, status }: { label?: string; status: ScorecardStatus }) {
  if (!label) return null;
  return (
    <div className="flex items-center gap-1.5">
      {status !== "neutral" ? <span className={cn("size-2 rounded-full", STATUS_DOT_CLASS[status])} /> : null}
      <span className="truncate text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-gray-400">
        {label}
      </span>
    </div>
  );
}

function ScorecardProgress({
  value,
  target,
  status,
  format,
  prefix,
  suffix,
}: {
  value: number;
  target?: number;
  status: ScorecardStatus;
  format: WidgetColumnFormat;
  prefix?: string;
  suffix?: string;
}) {
  if (target == null || target === 0) return null;
  const progressPct = Math.max(0, Math.min(1, value / target));
  return (
    <div className="flex flex-col gap-1">
      <div className="h-1.5 w-full overflow-hidden rounded-full bg-slate-100 dark:bg-gray-800">
        <div
          className={cn("h-full rounded-full transition-all", STATUS_BAR_CLASS[status])}
          style={{ width: `${(progressPct * 100).toFixed(1)}%` }}
        />
      </div>
      <span className="text-[11px] text-slate-400 dark:text-gray-500">
        Target {prefix ?? ""}
        {formatValue(target, format)}
        {suffix ?? ""}
      </span>
    </div>
  );
}

function DeltaChip({
  delta,
  comparison,
  display = "percent",
  format = "number",
  prefix,
  suffix,
}: {
  delta: Delta;
  comparison?: ScorecardComparison;
  display?: ScorecardTrendDisplay;
  format?: WidgetColumnFormat;
  prefix?: string;
  suffix?: string;
}) {
  const toneClass =
    delta.trend === "flat"
      ? "text-slate-500 dark:text-gray-400"
      : delta.improving
        ? "text-emerald-600 dark:text-emerald-400"
        : "text-red-600 dark:text-red-400";
  const Icon = delta.trend === "up" ? ArrowUpRight : delta.trend === "down" ? ArrowDownRight : Minus;
  const sign = delta.raw > 0 ? "+" : delta.raw < 0 ? "-" : "";
  const pctLabel = `${sign}${Math.abs(delta.pct * 100).toFixed(1)}%`;
  const absLabel = `${sign}${prefix ?? ""}${formatValue(Math.abs(delta.raw), format)}${suffix ?? ""}`;
  const changeLabel =
    display === "percent" ? pctLabel : display === "absolute" ? absLabel : `${absLabel} (${pctLabel})`;
  return (
    <span className={cn("inline-flex items-center gap-1 text-sm font-medium", toneClass)}>
      <Icon className="size-4" aria-hidden />
      {changeLabel}
      {comparison?.label ? (
        <span className="text-[11px] font-normal text-slate-400 dark:text-gray-500">{comparison.label}</span>
      ) : null}
    </span>
  );
}

/**
 * Compact filled sparkline. Mirrors the SVG approach used by `WidgetNumber`,
 * kept local so the scorecard prototype can tint it by status without
 * modifying the Number widget.
 */
function Sparkline({ values, className }: { values: number[]; className?: string }) {
  const width = 160;
  const height = 32;
  const strokeWidth = 1.5;
  const padY = Math.ceil(strokeWidth / 2) + 1;
  const plotTop = padY;
  const plotBottom = height - padY;
  const plotHeight = plotBottom - plotTop;
  const min = Math.min(...values);
  const max = Math.max(...values);
  const range = max - min || 1;
  const stepX = values.length > 1 ? width / (values.length - 1) : 0;
  const linePoints = values.map((v, i) => {
    const x = i * stepX;
    const y = plotTop + plotHeight - ((v - min) / range) * plotHeight;
    return `${x.toFixed(1)},${y.toFixed(1)}`;
  });
  const firstX = (0).toFixed(1);
  const lastX = ((values.length - 1) * stepX).toFixed(1);
  const baselineY = plotBottom.toFixed(1);
  const areaPath = `M${linePoints[0]} L${linePoints.slice(1).join(" L")} L${lastX},${baselineY} L${firstX},${baselineY} Z`;
  return (
    <svg
      width={width}
      height={height}
      className={cn("block", className)}
      viewBox={`0 0 ${width} ${height}`}
      preserveAspectRatio="none"
      aria-hidden
    >
      <path d={areaPath} fill="currentColor" fillOpacity={0.15} stroke="none" />
      <polyline
        points={linePoints.join(" ")}
        fill="none"
        stroke="currentColor"
        strokeWidth={strokeWidth}
        strokeLinejoin="round"
        strokeLinecap="round"
      />
    </svg>
  );
}
