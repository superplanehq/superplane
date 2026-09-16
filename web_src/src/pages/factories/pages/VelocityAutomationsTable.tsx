import { useMemo, useState } from "react";

import { Link } from "@/components/Link/link";
import { cn } from "@/lib/utils";

import { formatDurationHours } from "../lib/factoryVelocityFlow";
import { automationDetailPath } from "../lib/factoryPagePaths";
import type { VelocityAutomation } from "../lib/factoryVelocityReport";
import { velocityCardClassName } from "./velocityCards";
import { VelocitySortableHeader, type VelocitySortDirection } from "./VelocitySortableHeader";

function formatUsd(value: number): string {
  return `$${value.toFixed(2)}`;
}

function failureRatePct(automation: VelocityAutomation): number {
  if (automation.runs <= 0) return 0;
  return Math.round((automation.failed / automation.runs) * 1000) / 10;
}

/** A failure rate this high is worth reading before the cost columns. */
const ALARMING_FAILURE_RATE_PCT = 10;

type SortKey = "runs" | "failed" | "duration" | "averageCost" | "totalCost";

interface Column {
  key: SortKey;
  label: string;
  hint: string;
  format: (automation: VelocityAutomation) => string;
  /** The number the column sorts on, which is the leading number it shows. */
  value: (automation: VelocityAutomation) => number;
  isAlarming?: (automation: VelocityAutomation) => boolean;
}

const COLUMNS: Column[] = [
  {
    key: "runs",
    label: "Runs",
    hint: "Runs started in this period",
    format: (automation) => String(automation.runs),
    value: (automation) => automation.runs,
  },
  {
    key: "failed",
    label: "Failed",
    hint: "Runs that ended with an error",
    format: (automation) => `${automation.failed} (${failureRatePct(automation)}%)`,
    value: (automation) => automation.failed,
    isAlarming: (automation) => failureRatePct(automation) >= ALARMING_FAILURE_RATE_PCT,
  },
  {
    key: "duration",
    label: "Average duration",
    hint: "From start to end of a run",
    format: (automation) => formatDurationHours(automation.averageDurationHours),
    value: (automation) => automation.averageDurationHours,
  },
  {
    key: "averageCost",
    label: "Average cost",
    hint: "Tracked model spend of a run",
    format: (automation) => formatUsd(automation.averageCostUsd),
    value: (automation) => automation.averageCostUsd,
  },
  {
    key: "totalCost",
    label: "Total cost",
    hint: "Tracked model spend of every run in this period",
    format: (automation) => formatUsd(automation.totalCostUsd),
    value: (automation) => automation.totalCostUsd,
  },
];

/** Busiest first, which is the order the report arrives in. */
const DEFAULT_SORT_KEY: SortKey = "runs";
const DEFAULT_SORT_DIRECTION: VelocitySortDirection = "desc";

/**
 * Automation runs of the workspace. The rest of the page counts pull requests,
 * so this is the only place a reader sees the execution layer underneath them.
 *
 * Every automation of the period arrives in one report, so the sort runs here
 * instead of going back to the API the way the People table does.
 */
export function VelocityAutomationsTable({
  automations,
  organizationId,
  factoryKey,
  periodLabel,
}: {
  automations: VelocityAutomation[];
  organizationId: string;
  factoryKey: string;
  periodLabel: string;
}) {
  const [sortKey, setSortKey] = useState<SortKey>(DEFAULT_SORT_KEY);
  const [sortDirection, setSortDirection] = useState<VelocitySortDirection>(DEFAULT_SORT_DIRECTION);

  const sorted = useMemo(
    () => sortAutomations(automations, sortKey, sortDirection),
    [automations, sortKey, sortDirection],
  );

  const onSort = (key: SortKey) => {
    if (key === sortKey) {
      setSortDirection((direction) => (direction === "asc" ? "desc" : "asc"));
      return;
    }
    setSortKey(key);
    setSortDirection(DEFAULT_SORT_DIRECTION);
  };

  return (
    <section className={velocityCardClassName} data-testid="velocity-automations">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className="text-[14px] font-medium tracking-[-0.01em] text-foreground">Automations</h2>
          <p className="mt-0.5 text-[12px] text-muted-foreground">
            {automations.length} {automations.length === 1 ? "automation" : "automations"} with runs in this period
          </p>
        </div>
        <p className="text-[12px] text-muted-foreground">{periodLabel}</p>
      </div>

      <div className="mt-5 overflow-x-auto">
        <table className="w-full min-w-[720px] border-collapse text-[13px]">
          <thead>
            <tr className="border-b border-border">
              <th scope="col" className="pb-2 text-left text-[12px] font-normal text-muted-foreground">
                Automation
              </th>
              {COLUMNS.map((column) => (
                <VelocitySortableHeader
                  key={column.key}
                  label={column.label}
                  hint={column.hint}
                  isActive={sortKey === column.key}
                  direction={sortDirection}
                  onSort={() => onSort(column.key)}
                />
              ))}
            </tr>
          </thead>
          <tbody>
            {sorted.map((automation) => (
              <tr key={automation.id} className="border-b border-border/60 last:border-b-0">
                <td className="py-3 pr-6">
                  {/* `hover:!underline`: unlayered `a { text-decoration: inherit }` in index.css beats Tailwind utilities. */}
                  <Link
                    href={automationDetailPath(organizationId, factoryKey, automation.id)}
                    className="text-foreground underline-offset-4 hover:!underline"
                  >
                    {automation.name}
                  </Link>
                </td>
                {COLUMNS.map((column) => (
                  <td
                    key={column.key}
                    className={cn(
                      "py-3 pl-6 text-right tabular-nums",
                      cellToneClassName(column, automation, sortKey === column.key),
                    )}
                  >
                    {column.format(automation)}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

/** An alarming number keeps its warning tone even when the column is not sorted on. */
function cellToneClassName(column: Column, automation: VelocityAutomation, isSorted: boolean): string {
  if (column.isAlarming?.(automation)) return "text-rose-700 dark:text-rose-400";
  return isSorted ? "text-foreground" : "text-muted-foreground";
}

/** Ties fall back to the automation name, so equal rows hold a stable order. */
function sortAutomations(
  automations: VelocityAutomation[],
  sortKey: SortKey,
  direction: VelocitySortDirection,
): VelocityAutomation[] {
  const column = COLUMNS.find((candidate) => candidate.key === sortKey);
  if (!column) return automations;

  return [...automations].sort((left, right) => {
    const delta = column.value(left) - column.value(right);
    if (delta !== 0) return direction === "asc" ? delta : -delta;
    return left.name.localeCompare(right.name);
  });
}
