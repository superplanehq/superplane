import { ArrowDownRight, ArrowUpRight } from "lucide-react";

import { Avatar } from "@/components/Avatar/avatar";
import { Timestamp, type TimestampDisplay } from "@/components/Timestamp";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

import { ConsoleBadge } from "../ConsoleBadge";
import { resolveConsoleAvatar } from "../consoleAvatar";
import { CONSOLE_CODE_BADGE_CLASSES } from "../consoleCodeStyles";
import { CONSOLE_LINK_CLASSES } from "../consoleLinkStyles";
import { evaluateRowShow } from "./rowVisibility";
import { resolveCellValue } from "./resolveCellValue";
import { resolveHref } from "./resolveHref";
import type { WidgetTableRender } from "./types";
import { coerceWidgetTimestamp, computeProgress, formatPercentageDisplay, formatValue } from "./widgetFormat";
import { computeTrend, formatTrendLabel, formatTrendTooltip, type TrendResult } from "./widgetTrend";

type WidgetTableColumn = WidgetTableRender["columns"][number];

export interface WidgetTableCellProps {
  col: WidgetTableColumn;
  row: Record<string, unknown>;
  /**
   * The row rendered immediately below the current one, after filter+sort.
   * Only consumed by `format: "trend"` columns. `undefined` for the last
   * visible row.
   */
  nextRow?: Record<string, unknown>;
  /**
   * Whether more rows can still be loaded below the current one. Only
   * meaningful for the last visible row of a paginated table when the
   * previous entry has not been fetched yet; enables the `...` pending
   * state on `format: "trend"` cells. Must not be set merely because more
   * rows are loaded but still hidden behind the progressive display window
   * — pass those via `nextRow` instead.
   */
  hasMoreBelow?: boolean;
}

export function WidgetTableCell({ col, row, nextRow, hasMoreBelow }: WidgetTableCellProps) {
  const visible = evaluateRowShow(col.show, row);
  if (!visible) return <EmptyCell />;

  if (col.format === "trend") {
    return <TrendCell col={col} row={row} nextRow={nextRow} hasMoreBelow={hasMoreBelow} />;
  }

  const value = resolveCellValue(col.field, row);
  const formatted = formatValue(value, col.format);

  switch (col.format) {
    case "badge":
    case "status":
      return <BadgeCell label={formatted} />;
    case "date":
    case "datetime":
    case "relative":
      return <TimestampCell format={col.format} value={value} label={formatted} />;
    case "avatar":
      return <AvatarCell col={col} row={row} value={value} />;
    case "link":
      return <LinkCell col={col} row={row} value={value} label={formatted} />;
    case "code":
      return <CodeCell label={formatted} />;
    case "progress":
      return <ProgressCell col={col} row={row} value={value} />;
    default:
      if (col.href) return <LinkCell col={col} row={row} value={value} label={formatted} />;
      return <TextCell label={formatted} />;
  }
}

function EmptyCell() {
  return <td className="px-3 py-1.5 text-slate-300 dark:text-gray-600">—</td>;
}

function BadgeCell({ label }: { label: string }) {
  return (
    <td className="px-3 py-1.5">
      <ConsoleBadge label={label} />
    </td>
  );
}

const TIMESTAMP_DISPLAY_BY_FORMAT: Record<"date" | "datetime" | "relative", TimestampDisplay> = {
  date: "date",
  datetime: "datetime",
  relative: "relative",
};

function TimestampCell({
  format,
  value,
  label,
}: {
  format: "date" | "datetime" | "relative";
  value: unknown;
  label: string;
}) {
  const date = coerceWidgetTimestamp(value);
  if (!date) {
    // Preserve the raw fallback text (e.g. an unparseable string) rather than
    // rendering an empty cell — matches the pre-Timestamp behavior.
    return <TextCell label={label} />;
  }
  return (
    <td className="px-3 py-1.5 text-slate-700 dark:text-gray-300">
      <Timestamp
        date={date}
        display={TIMESTAMP_DISPLAY_BY_FORMAT[format]}
        relativeStyle="abbreviated"
        includeAgo={false}
      />
    </td>
  );
}

function AvatarCell({ col, row, value }: { col: WidgetTableColumn; row: Record<string, unknown>; value: unknown }) {
  const committer = col.avatarCommitterField ? resolveCellValue(col.avatarCommitterField, row) : undefined;
  const { src, initials, name } = resolveConsoleAvatar(value, committer);
  if (!name && !src && !initials) return <EmptyCell />;

  return (
    <td className="px-3 py-1.5 align-middle">
      <Tooltip>
        <TooltipTrigger asChild>
          <span className="inline-flex cursor-default">
            <Avatar
              src={src ?? null}
              initials={initials}
              className="size-6 bg-slate-200 text-slate-600 dark:bg-gray-700 dark:text-gray-200"
            />
          </span>
        </TooltipTrigger>
        {name ? <TooltipContent side="top">{name}</TooltipContent> : null}
      </Tooltip>
    </td>
  );
}

function LinkCell({
  col,
  row,
  value,
  label,
}: {
  col: WidgetTableColumn;
  row: Record<string, unknown>;
  value: unknown;
  label: string;
}) {
  const href = col.href ? resolveHref(col.href, row) : String(value ?? "");
  return (
    <td className="px-3 py-1.5">
      <a href={href} target="_blank" rel="noopener noreferrer" className={CONSOLE_LINK_CLASSES}>
        {label || href}
      </a>
    </td>
  );
}

function CodeCell({ label }: { label: string }) {
  return (
    <td className="px-3 py-1.5">
      <code className={CONSOLE_CODE_BADGE_CLASSES}>{label}</code>
    </td>
  );
}

function TextCell({ label }: { label: string }) {
  return <td className="px-3 py-1.5 text-slate-700 dark:text-gray-300">{label}</td>;
}

const TREND_MUTED_CLASSES = "text-slate-400 dark:text-gray-500";
const TREND_BETTER_CLASSES = "text-emerald-600 dark:text-emerald-400";
const TREND_WORSE_CLASSES = "text-red-600 dark:text-red-400";

function TrendCell({
  col,
  row,
  nextRow,
  hasMoreBelow,
}: {
  col: WidgetTableColumn;
  row: Record<string, unknown>;
  nextRow: Record<string, unknown> | undefined;
  hasMoreBelow: boolean | undefined;
}) {
  const current = resolveCellValue(col.field, row);
  // `undefined` means "no row below" to computeTrend. A present next row with a
  // missing field must become `null` so it renders as incomparable, not no-baseline.
  const previous = nextRow ? (resolveCellValue(col.field, nextRow) ?? null) : undefined;
  const result = computeTrend(current, previous, {
    better: col.trendBetter,
    display: col.trendDisplay,
    hasMoreBelow: nextRow ? false : Boolean(hasMoreBelow),
  });
  const label = formatTrendLabel(result, col.trendDisplay);
  const tooltip = formatTrendTooltip(result);

  const content = (
    <span
      className={cn("inline-flex items-center gap-1 whitespace-nowrap tabular-nums", trendColorClasses(result))}
      data-testid="widget-trend-cell"
      data-trend-kind={result.kind}
      data-trend-direction={result.kind === "changed" ? result.direction : undefined}
      data-trend-polarity={result.kind === "changed" ? result.polarity : undefined}
    >
      {renderTrendIcon(result)}
      {label ? <span>{label}</span> : null}
    </span>
  );

  return (
    <td className="px-3 py-1.5 align-middle">
      {tooltip ? (
        <Tooltip>
          <TooltipTrigger asChild>
            <span className="inline-flex cursor-default">{content}</span>
          </TooltipTrigger>
          <TooltipContent side="top">{tooltip}</TooltipContent>
        </Tooltip>
      ) : (
        content
      )}
    </td>
  );
}

function renderTrendIcon(result: TrendResult) {
  if (result.kind !== "changed") {
    return <span aria-hidden="true">-</span>;
  }
  if (result.direction === "up") {
    return <ArrowUpRight className="size-3.5" aria-hidden="true" />;
  }
  return <ArrowDownRight className="size-3.5" aria-hidden="true" />;
}

function trendColorClasses(result: TrendResult): string {
  if (result.kind !== "changed") return TREND_MUTED_CLASSES;
  return result.polarity === "better" ? TREND_BETTER_CLASSES : TREND_WORSE_CLASSES;
}

function ProgressCell({ col, row, value }: { col: WidgetTableColumn; row: Record<string, unknown>; value: unknown }) {
  const target = resolveProgressTarget(col.progressTarget, row);
  const progress = computeProgress(value, target);
  const labelKind = col.progressLabel ?? "percent";

  if (!progress) {
    return (
      <td className="px-3 py-1.5">
        <div className="flex min-w-[80px] items-center gap-2">
          <div
            className="h-2 min-w-[32px] flex-1 rounded-full bg-slate-200 dark:bg-slate-700"
            aria-hidden="true"
            data-testid="widget-progress-track"
          />
          {labelKind !== "none" ? (
            <span
              className="shrink-0 whitespace-nowrap tabular-nums text-slate-400 dark:text-gray-500"
              data-testid="widget-progress-label"
            >
              —
            </span>
          ) : null}
        </div>
      </td>
    );
  }

  const tooltipLabel = formatPercentageDisplay(progress.percent);

  return (
    <td className="px-3 py-1.5">
      <div className="flex min-w-[80px] items-center gap-2">
        <Tooltip>
          <TooltipTrigger asChild>
            <div
              className="h-2 min-w-[32px] flex-1 cursor-default overflow-hidden rounded-full bg-slate-200 dark:bg-slate-700"
              role="progressbar"
              aria-valuenow={Math.round(progress.barPercent)}
              aria-valuemin={0}
              aria-valuemax={100}
              aria-valuetext={tooltipLabel}
              data-testid="widget-progress-track"
            >
              <div
                className="h-full rounded-full bg-sky-500 transition-[width] dark:bg-indigo-300"
                style={{ width: `${progress.barPercent}%` }}
                data-testid="widget-progress-fill"
              />
            </div>
          </TooltipTrigger>
          <TooltipContent side="top">{tooltipLabel}</TooltipContent>
        </Tooltip>
        {labelKind !== "none" ? (
          <span
            className="shrink-0 whitespace-nowrap tabular-nums text-slate-700 dark:text-gray-300"
            data-testid="widget-progress-label"
          >
            {formatProgressLabel(progress.current, progress.target, progress.percent, labelKind)}
          </span>
        ) : null}
      </div>
    </td>
  );
}

/**
 * Resolve a column's `progressTarget` against the row. Numeric literals
 * (`"10"`, `"100.5"`) are used verbatim; anything else is passed through the
 * shared field resolver so authors can bind to a row field (`total`,
 * `payload.goal`) or a full CEL expression (`{{ items.size() }}`).
 */
function resolveProgressTarget(target: string | undefined, row: Record<string, unknown>): unknown {
  if (target == null) return undefined;
  const trimmed = target.trim();
  if (trimmed === "") return undefined;
  const literal = Number(trimmed);
  if (Number.isFinite(literal)) return literal;
  return resolveCellValue(trimmed, row);
}

function formatProgressLabel(current: number, target: number, percent: number, kind: "number" | "percent"): string {
  if (kind === "number") {
    return `${formatNumericLabel(current)}/${formatNumericLabel(target)}`;
  }
  return formatPercentageDisplay(percent);
}

function formatNumericLabel(value: number): string {
  // Match `formatValue(_, "number")`: locale-aware thousands separators, no
  // forced decimals. Fractional values keep up to one decimal so `0.5/10`
  // stays readable without trailing 15-digit float noise.
  if (Number.isInteger(value)) return value.toLocaleString();
  return value.toLocaleString(undefined, { maximumFractionDigits: 1 });
}
