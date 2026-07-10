import { Avatar } from "@/components/Avatar/avatar";
import { Timestamp, type TimestampDisplay } from "@/components/Timestamp";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

import { ConsoleBadge } from "../ConsoleBadge";
import { resolveConsoleAvatar } from "../consoleAvatar";
import { CONSOLE_CODE_BADGE_CLASSES } from "../consoleCodeStyles";
import { CONSOLE_LINK_CLASSES } from "../consoleLinkStyles";
import { evaluateRowShow } from "./rowVisibility";
import { resolveCellValue } from "./resolveCellValue";
import { resolveHref } from "./resolveHref";
import type { WidgetTableRender } from "./types";
import { coerceWidgetTimestamp, formatValue } from "./widgetFormat";

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
      <Timestamp date={date} display={TIMESTAMP_DISPLAY_BY_FORMAT[format]} relativeStyle="abbreviated" />
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
