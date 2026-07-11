import { ArrowDownRight, ArrowUpRight } from "lucide-react";

import { Avatar } from "@/components/Avatar/avatar";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { formatTimestampInUserTimezone } from "@/lib/timezone";

import { ConsoleBadge } from "../ConsoleBadge";
import { resolveConsoleAvatar } from "../consoleAvatar";
import { CONSOLE_CODE_BADGE_CLASSES } from "../consoleCodeStyles";
import { CONSOLE_LINK_CLASSES } from "../consoleLinkStyles";
import { evaluateRowShow } from "./rowVisibility";
import { resolveCellValue } from "./resolveCellValue";
import { resolveHref } from "./resolveHref";
import type { WidgetTableRender } from "./types";
import { formatValue } from "./widgetFormat";
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
   * meaningful for the last visible row of a paginated table; enables the
   * `...` pending state on `format: "trend"` cells.
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
    case "relative":
      return <RelativeCell value={value} label={formatted} />;
    case "avatar":
      return <AvatarCell col={col} row={row} value={value} />;
    case "link":
      return <LinkCell col={col} row={row} value={value} label={formatted} />;
    case "code":
      return <CodeCell label={formatted} />;
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

function RelativeCell({ value, label }: { value: unknown; label: string }) {
  return (
    <td className="px-3 py-1.5 text-slate-700 dark:text-gray-300" title={formatAbsoluteTitle(value)}>
      {label}
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
