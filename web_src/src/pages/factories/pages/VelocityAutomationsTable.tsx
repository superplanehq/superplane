import { Link } from "@/components/Link/link";
import { cn } from "@/lib/utils";

import { formatDurationHours } from "../lib/factoryVelocityFlow";
import { automationDetailPath } from "../lib/factoryPagePaths";
import type { AutomationRunRow } from "./velocityAutomationsMockData";
import { velocityCardClassName } from "./velocityCards";

function formatUsd(value: number): string {
  return `$${value.toFixed(2)}`;
}

function failureRatePct(automation: AutomationRunRow): number {
  if (automation.runs <= 0) return 0;
  return Math.round((automation.failed / automation.runs) * 1000) / 10;
}

/** A failure rate this high is worth reading before the cost columns. */
const ALARMING_FAILURE_RATE_PCT = 10;

interface Column {
  label: string;
  hint: string;
  format: (automation: AutomationRunRow) => string;
  isAlarming?: (automation: AutomationRunRow) => boolean;
}

const COLUMNS: Column[] = [
  {
    label: "Runs",
    hint: "Runs started in this period",
    format: (automation) => String(automation.runs),
  },
  {
    label: "Failed",
    hint: "Runs that ended with an error",
    format: (automation) => `${automation.failed} (${failureRatePct(automation)}%)`,
    isAlarming: (automation) => failureRatePct(automation) >= ALARMING_FAILURE_RATE_PCT,
  },
  {
    label: "Average duration",
    hint: "From start to end of a run",
    format: (automation) => formatDurationHours(automation.averageDurationHours),
  },
  {
    label: "Average cost",
    hint: "Tracked model spend of a run",
    format: (automation) => formatUsd(automation.averageCostUsd),
  },
  {
    label: "Total cost",
    hint: "Tracked model spend of every run in this period",
    format: (automation) => formatUsd(automation.totalCostUsd),
  },
];

/**
 * Automation runs of the workspace. The rest of the page counts pull requests,
 * so this is the only place a reader sees the execution layer underneath them.
 */
export function VelocityAutomationsTable({
  automations,
  organizationId,
  factoryKey,
  periodLabel,
}: {
  automations: AutomationRunRow[];
  organizationId: string;
  factoryKey: string;
  periodLabel: string;
}) {
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
                <th
                  key={column.label}
                  scope="col"
                  title={column.hint}
                  className="pb-2 pl-6 text-right text-[12px] font-normal text-muted-foreground"
                >
                  {column.label}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {automations.map((automation) => (
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
                    key={column.label}
                    className={cn(
                      "py-3 pl-6 text-right tabular-nums",
                      column.isAlarming?.(automation) ? "text-rose-700 dark:text-rose-400" : "text-muted-foreground",
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
