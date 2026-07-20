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

/**
 * Render one card field on a kanban board. Reuses the shared
 * {@link WidgetTableColumn} shape and the same `format` vocabulary as the
 * table renderer — but omits progress / trend / avatar cases that don't
 * fit inside a compact card meta line.
 *
 * Fields whose `show` expression evaluates false render as `null` (the row
 * is otherwise unaffected). The label is only shown when the column
 * declares one; unlabelled fields render the value alone.
 */
export function WidgetBoardCardField({ col, row }: { col: WidgetTableColumn; row: Record<string, unknown> }) {
  if (!evaluateRowShow(col.show, row)) return null;

  const value = resolveCellValue(col.field, row);
  const formatted = formatValue(value, col.format);
  const displayLabel = col.label?.trim();

  return (
    <div className="flex items-center gap-1.5 text-[11px] text-slate-600 dark:text-gray-300" data-testid="board-card-field">
      {displayLabel ? <span className="shrink-0 text-slate-400 dark:text-gray-500">{displayLabel}</span> : null}
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
