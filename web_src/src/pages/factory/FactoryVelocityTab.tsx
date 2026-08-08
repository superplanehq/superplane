import { Info } from "lucide-react";

import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { cn } from "@/lib/utils";

import { factoryPanelClassName, mutedTextClassName, sectionTitleClassName } from "./factoryStyles";
import type { VelocityCohort, VelocityData } from "./factoryTypes";

/**
 * Two series, assigned in fixed order and never cycled. Blue/amber is the
 * classic colour-vision-deficiency-safe pair; identity is also carried by the
 * legend and the row labels, so colour is never the only channel.
 */
const SERIES = {
  human: { label: "Human-authored", fill: "#2563eb", dark: "#60a5fa" },
  factory: { label: "Factory-authored", fill: "#d97706", dark: "#fbbf24" },
} as const;

interface FactoryVelocityTabProps {
  velocity: VelocityData;
  onSelectRepository: (repository: string) => void;
}

/**
 * PRD: the same delivery indicators for Team total, human-authored and
 * Factory-authored work, "without framing the comparison as a competition".
 *
 * That framing decision drives the layout: a plain indicator table with one row
 * per cohort, no ranking, no winner highlight, no bars racing each other. The
 * only chart compares the two authored cohorts over time, stacked into the team
 * total rather than set against one another.
 */
export function FactoryVelocityTab({ velocity, onSelectRepository }: FactoryVelocityTabProps) {
  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className={cn("text-sm", mutedTextClassName)}>Last {velocity.periodDays} days</p>
        {/* PRD: repository is the filter — never Automation, since several
            Automations can contribute to one Work Order and repository. */}
        {/* Not a <label>: the Radix trigger is a button, so the accessible name
            rides on aria-label rather than a label/control association. */}
        <div className="flex items-center gap-2 text-sm">
          <span aria-hidden className={mutedTextClassName}>
            Repository
          </span>
          <Select value={velocity.selectedRepository} onValueChange={onSelectRepository}>
            <SelectTrigger className="w-64" aria-label="Filter by repository">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {velocity.repositories.map((repository) => (
                <SelectItem key={repository} value={repository}>
                  {repository}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <CohortTable cohorts={velocity.cohorts} />
      <ThroughputOverTime series={velocity.throughputSeries} />
      <CostBreakdown breakdown={velocity.costBreakdown} />
    </div>
  );
}

const usd = (value: number) => `$${value.toFixed(2)}`;

function CohortTable({ cohorts }: { cohorts: VelocityCohort[] }) {
  return (
    <section className={factoryPanelClassName}>
      <h2 className={cn(sectionTitleClassName, "mb-3")}>Delivery indicators</h2>
      <div className="overflow-x-auto">
        <table className="w-full min-w-[640px] text-sm">
          <thead>
            <tr className={cn("border-b border-slate-200 text-left text-xs dark:border-gray-700", mutedTextClassName)}>
              <th scope="col" className="py-2 pr-4 font-medium">
                Cohort
              </th>
              <th scope="col" className="py-2 pr-4 text-right font-medium">
                Merged pull requests
              </th>
              <th scope="col" className="py-2 pr-4 text-right font-medium">
                Median cycle time
              </th>
              <th scope="col" className="py-2 pr-4 text-right font-medium">
                Success rate
              </th>
              <th scope="col" className="py-2 text-right font-medium">
                Tracked cost
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100 dark:divide-gray-700/70">
            {cohorts.map((cohort) => (
              <tr key={cohort.id}>
                <th scope="row" className="py-2.5 pr-4 text-left font-medium text-slate-900 dark:text-gray-100">
                  <span className="inline-flex items-center gap-2">
                    {cohort.id !== "team" && (
                      <span
                        aria-hidden
                        className="size-2 shrink-0 rounded-full"
                        style={{ backgroundColor: SERIES[cohort.id].fill }}
                      />
                    )}
                    {cohort.label}
                  </span>
                </th>
                <td className="py-2.5 pr-4 text-right tabular-nums text-slate-900 dark:text-gray-100">
                  {cohort.mergedPullRequests}
                </td>
                <td className="py-2.5 pr-4 text-right tabular-nums text-slate-900 dark:text-gray-100">
                  {cohort.cycleTimeHours}h
                </td>
                <td className="py-2.5 pr-4 text-right tabular-nums text-slate-900 dark:text-gray-100">
                  {Math.round(cohort.successRate * 100)}%
                </td>
                <td
                  className={cn(
                    "py-2.5 text-right tabular-nums",
                    cohort.trackedCostUsd === null ? mutedTextClassName : "text-slate-900 dark:text-gray-100",
                  )}
                >
                  {cohort.trackedCostUsd === null ? "Not available" : usd(cohort.trackedCostUsd)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {/* PRD: human work must not read as costing nothing. */}
      <p className={cn("mt-3 flex items-start gap-1.5 text-xs", mutedTextClassName)}>
        <Info className="mt-0.5 size-3.5 shrink-0" aria-hidden />
        Tracked cost covers model tokens and execution compute only. It excludes third-party service charges, and is not
        available for human-authored work — that is a missing attribution model, not an absence of cost.
      </p>
    </section>
  );
}

function ThroughputOverTime({ series }: { series: VelocityData["throughputSeries"] }) {
  const max = Math.max(1, ...series.map((d) => d.human + d.factory));
  const barWidth = 100 / (series.length * 1.6);
  const gap = barWidth * 0.6;

  return (
    <section className={factoryPanelClassName}>
      <div className="mb-1 flex flex-wrap items-center justify-between gap-3">
        <h2 className={sectionTitleClassName}>Merged pull requests over time</h2>
        <ul className="flex items-center gap-4 text-xs">
          {(["human", "factory"] as const).map((key) => (
            <li key={key} className={cn("inline-flex items-center gap-1.5", mutedTextClassName)}>
              <span aria-hidden className="size-2 rounded-full" style={{ backgroundColor: SERIES[key].fill }} />
              {SERIES[key].label}
            </li>
          ))}
        </ul>
      </div>
      <p className={cn("mb-3 text-xs", mutedTextClassName)}>
        Stacked to the team total — the cohorts sum, they do not compete.
      </p>

      <div className="overflow-x-auto">
        <svg
          viewBox="0 0 100 34"
          preserveAspectRatio="none"
          className="h-40 w-full min-w-[560px]"
          role="img"
          aria-label="Merged pull requests per day, human-authored and Factory-authored, stacked"
        >
          {series.map((day, index) => {
            const x = index * (barWidth + gap) + gap / 2;
            const humanHeight = (day.human / max) * 28;
            const factoryHeight = (day.factory / max) * 28;
            return (
              <g key={day.date}>
                <rect x={x} y={30 - humanHeight} width={barWidth} height={humanHeight} fill={SERIES.human.fill} />
                <rect
                  x={x}
                  y={30 - humanHeight - factoryHeight}
                  width={barWidth}
                  height={factoryHeight}
                  fill={SERIES.factory.fill}
                />
              </g>
            );
          })}
          <line
            x1="0"
            y1="30.4"
            x2="100"
            y2="30.4"
            stroke="currentColor"
            strokeWidth="0.15"
            className="text-slate-300 dark:text-gray-600"
          />
        </svg>
      </div>
      <div className={cn("mt-1 flex justify-between text-xs", mutedTextClassName)}>
        <span>{series[0]?.date}</span>
        <span>{series[series.length - 1]?.date}</span>
      </div>
    </section>
  );
}

function CostBreakdown({ breakdown }: { breakdown: VelocityData["costBreakdown"] }) {
  const total = breakdown.tokensUsd + breakdown.computeUsd;
  const rows = [
    { label: "Model tokens", value: breakdown.tokensUsd },
    { label: "Execution compute", value: breakdown.computeUsd },
  ];

  return (
    <section className={factoryPanelClassName}>
      <h2 className={cn(sectionTitleClassName, "mb-3")}>Tracked Factory cost</h2>
      <ul className="flex flex-col gap-2">
        {rows.map((row) => (
          <li key={row.label} className="flex items-center justify-between gap-4 text-sm">
            <span className={mutedTextClassName}>{row.label}</span>
            <span className="tabular-nums text-slate-900 dark:text-gray-100">{usd(row.value)}</span>
          </li>
        ))}
        <li className="mt-1 flex items-center justify-between gap-4 border-t border-slate-200 pt-2 text-sm font-medium dark:border-gray-700">
          <span className="text-slate-900 dark:text-gray-100">Total</span>
          <span className="tabular-nums text-slate-900 dark:text-gray-100">{usd(total)}</span>
        </li>
      </ul>
    </section>
  );
}
