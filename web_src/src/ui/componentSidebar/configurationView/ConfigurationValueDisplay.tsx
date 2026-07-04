import { cn, isUrl } from "@/lib/utils";
import type { ConfigurationDisplayRow } from "./types";
import { EMPTY_DISPLAY_VALUE } from "./formatConfigurationValue";

type ConfigurationValueDisplayProps = {
  row: ConfigurationDisplayRow;
  className?: string;
};

const INTEGRATION_STATUS_CLASSES = {
  ready:
    "border border-green-950/15 bg-green-100 text-green-800 dark:border-green-950/15 dark:bg-green-900/30 dark:text-green-400",
  error: "border border-red-950/15 bg-red-100 text-red-800 dark:border-red-950/15 dark:bg-red-900/30 dark:text-red-400",
  pending:
    "border border-orange-950/15 bg-orange-100 text-yellow-800 dark:border-orange-950/15 dark:bg-orange-950/30 dark:text-yellow-400",
} as const;

function IntegrationStatusBadge({ row, className }: { row: ConfigurationDisplayRow; className?: string }) {
  const variant = row.integrationStatusVariant ?? "pending";
  const showSummary =
    row.displayText !== "" && row.displayText !== EMPTY_DISPLAY_VALUE && row.displayText !== row.integrationStatus;

  return (
    <span className={cn("inline-flex flex-wrap items-center gap-2", className)}>
      {showSummary ? <span className="text-[13px] text-gray-800 dark:text-gray-100">{row.displayText}</span> : null}
      <span
        className={cn(
          "inline-flex items-center rounded px-2 py-0.5 text-xs font-medium",
          INTEGRATION_STATUS_CLASSES[variant],
        )}
      >
        {row.integrationStatus}
      </span>
    </span>
  );
}

function ChipList({ chips, className }: { chips: string[]; className?: string }) {
  return (
    <div className={cn("flex flex-wrap gap-1", className)}>
      {chips.map((chip) => (
        <span
          key={chip}
          className="inline-flex max-w-full truncate rounded bg-slate-100 px-1.5 py-0.5 text-[11px] font-medium text-slate-700 dark:bg-slate-800 dark:text-slate-200"
          title={chip}
        >
          {chip}
        </span>
      ))}
    </div>
  );
}

export function ConfigurationValueDisplay({ row, className }: ConfigurationValueDisplayProps) {
  if (row.kind === "empty" || row.displayText === EMPTY_DISPLAY_VALUE) {
    return <span className={cn("text-gray-400 dark:text-gray-500", className)}>{EMPTY_DISPLAY_VALUE}</span>;
  }

  if (row.kind === "integration" && row.integrationStatus) {
    return <IntegrationStatusBadge row={row} className={className} />;
  }

  if (row.chips && row.chips.length > 0) {
    return <ChipList chips={row.chips} className={className} />;
  }

  const candidateHref = row.href ?? (row.kind === "url" ? row.displayText : undefined);
  const href = candidateHref && isUrl(candidateHref) ? candidateHref : undefined;
  if (href) {
    return (
      <a
        href={href}
        target="_blank"
        rel="noopener noreferrer"
        className={cn("min-w-0 break-all text-blue-600 underline underline-offset-2 hover:text-blue-700", className)}
      >
        {row.displayText}
      </a>
    );
  }

  const isMonospace = row.kind === "expression" || row.kind === "code";

  return (
    <span
      className={cn(
        "min-w-0 whitespace-pre-wrap break-words text-gray-800 dark:text-gray-100",
        isMonospace && "font-mono text-[12px]",
        className,
      )}
    >
      {row.displayText}
    </span>
  );
}
