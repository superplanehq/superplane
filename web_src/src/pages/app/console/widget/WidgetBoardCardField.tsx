import { Timer } from "lucide-react";

import { Timestamp, type TimestampDisplay } from "@/components/Timestamp";
import { cn } from "@/lib/utils";

import { ConsoleBadge } from "../ConsoleBadge";
import { CONSOLE_CODE_BADGE_CLASSES } from "../consoleCodeStyles";
import { CONSOLE_LINK_CLASSES } from "../consoleLinkStyles";
import { evaluateRowShow } from "./rowVisibility";
import { resolveCellValue } from "./resolveCellValue";
import { resolveHref } from "./resolveHref";
import type { WidgetTableColumn } from "./types";
import { coerceWidgetTimestamp, formatValue } from "./widgetFormat";

export type BoardCardFieldVariant = "default" | "header" | "meta";

/**
 * Render one card field on a kanban board. Reuses the shared
 * {@link WidgetTableColumn} shape and the same `format` vocabulary as the
 * table renderer — but omits progress / trend / avatar cases that don't
 * fit inside a compact card meta line.
 *
 * Fields whose `show` expression evaluates false render as `null` (the row
 * is otherwise unaffected). The label is only shown when the column
 * declares one; unlabelled fields render the value alone.
 *
 * Variants:
 * - `header` — compact gray link above the title (label + value as one link)
 * - `meta` — duration (timer icon, left) / relative timestamp (value only, right)
 * - `default` — stacked label + value row under the card
 */
export function WidgetBoardCardField({
  col,
  row,
  variant = "default",
}: {
  col: WidgetTableColumn;
  row: Record<string, unknown>;
  variant?: BoardCardFieldVariant;
}) {
  if (!evaluateRowShow(col.show, row)) return null;

  const value = resolveCellValue(col.field, row);
  const formatted = formatValue(value, col.format);
  if (!formatted.trim()) return null;

  const displayLabel = col.label?.trim();

  if (variant === "header") {
    return <HeaderLinkField col={col} row={row} value={value} formatted={formatted} displayLabel={displayLabel} />;
  }

  if (variant === "meta") {
    return <MetaField col={col} row={row} value={value} formatted={formatted} />;
  }

  return (
    <div
      className="flex items-center gap-1.5 text-[11px] text-slate-600 dark:text-gray-300"
      data-testid="board-card-field"
    >
      {displayLabel ? <span className="shrink-0 text-gray-600 dark:text-gray-500">{displayLabel}</span> : null}
      <span className="min-w-0 truncate">
        <FieldValue col={col} row={row} value={value} formatted={formatted} />
      </span>
    </div>
  );
}

/** Link fields with an `href` (or `format: link`) belong in the header row above the title. */
export function isBoardCardHeaderField(col: WidgetTableColumn): boolean {
  return col.format === "link" || Boolean(col.href?.trim());
}

/** Duration / relative fields share one meta row under the title. */
export function isBoardCardMetaField(col: WidgetTableColumn): boolean {
  return col.format === "duration" || col.format === "relative";
}

function HeaderLinkField({
  col,
  row,
  value,
  formatted,
  displayLabel,
}: {
  col: WidgetTableColumn;
  row: Record<string, unknown>;
  value: unknown;
  formatted: string;
  displayLabel: string | undefined;
}) {
  const href = col.href ? resolveHref(col.href, row) : String(value ?? "");
  if (!href.trim()) return null;
  const text = displayLabel ? `${displayLabel} ${formatted}` : formatted;
  return (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      className="truncate text-xs text-gray-600 no-underline underline-offset-2 hover:!underline dark:text-gray-400"
      data-testid="board-card-field"
    >
      {text}
    </a>
  );
}

function MetaField({
  col,
  row,
  value,
  formatted,
}: {
  col: WidgetTableColumn;
  row: Record<string, unknown>;
  value: unknown;
  formatted: string;
}) {
  return (
    <div className="flex items-center gap-1 text-xs text-gray-600 dark:text-gray-400" data-testid="board-card-field">
      {col.format === "duration" ? <Timer className="size-3 shrink-0" aria-hidden /> : null}
      <span className="min-w-0 truncate">
        <FieldValue col={col} row={row} value={value} formatted={formatted} />
      </span>
    </div>
  );
}

function FieldValue({
  col,
  row,
  value,
  formatted,
}: {
  col: WidgetTableColumn;
  row: Record<string, unknown>;
  value: unknown;
  formatted: string;
}) {
  switch (col.format) {
    case "badge":
    case "status":
      return <ConsoleBadge label={formatted} />;
    case "date":
    case "datetime":
    case "relative":
      return <TimestampField format={col.format} value={value} fallback={formatted} />;
    case "code":
      return <code className={cn(CONSOLE_CODE_BADGE_CLASSES, "truncate")}>{formatted}</code>;
    case "link":
      return <LinkField col={col} row={row} value={value} label={formatted} />;
    default:
      if (col.href) return <LinkField col={col} row={row} value={value} label={formatted} />;
      return <span className="truncate">{formatted}</span>;
  }
}

const TIMESTAMP_DISPLAY_BY_FORMAT: Record<"date" | "datetime" | "relative", TimestampDisplay> = {
  date: "date",
  datetime: "datetime",
  relative: "relative",
};

function TimestampField({
  format,
  value,
  fallback,
}: {
  format: "date" | "datetime" | "relative";
  value: unknown;
  fallback: string;
}) {
  const date = coerceWidgetTimestamp(value);
  if (!date) return <span className="truncate">{fallback}</span>;
  return (
    <Timestamp
      date={date}
      display={TIMESTAMP_DISPLAY_BY_FORMAT[format]}
      relativeStyle="abbreviated"
      includeAgo={false}
    />
  );
}

function LinkField({
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
    <a href={href} target="_blank" rel="noopener noreferrer" className={cn(CONSOLE_LINK_CLASSES, "truncate")}>
      {label || href}
    </a>
  );
}
