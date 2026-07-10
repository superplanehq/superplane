import { Avatar } from "@/components/Avatar/avatar";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { formatTimestampInUserTimezone } from "@/lib/timezone";

import { ConsoleBadge } from "../ConsoleBadge";
import { resolveConsoleAvatar } from "../consoleAvatar";
import { CONSOLE_CODE_BADGE_CLASSES } from "../consoleCodeStyles";
import { CONSOLE_LINK_CLASSES } from "../consoleLinkStyles";
import { evaluateRowShow } from "./rowVisibility";
import { resolveCellValue } from "./resolveCellValue";
import { resolveHref } from "./resolveHref";
import type { WidgetTableRender } from "./types";
import { computeProgress, formatPercentageDisplay, formatValue } from "./widgetFormat";

type WidgetTableColumn = WidgetTableRender["columns"][number];

export function WidgetTableCell({ col, row }: { col: WidgetTableColumn; row: Record<string, unknown> }) {
  const visible = evaluateRowShow(col.show, row);
  if (!visible) return <EmptyCell />;

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
              aria-valuenow={Math.round(progress.percent)}
              aria-valuemin={0}
              aria-valuemax={100}
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
