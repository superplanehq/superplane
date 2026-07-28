import { Badge } from "@/components/ui/badge";
import {
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { cn } from "@/lib/utils";
import { Blend, Bot, CalendarDays, Coins, Cpu, GitBranch, Info, Users } from "lucide-react";
import { CartesianGrid, Line, LineChart, XAxis, YAxis } from "recharts";
import { useState } from "react";

import type { FactoryRepository, RepositoryVelocity, VelocityCohort, VelocityCohortId, VelocityMetric } from "./types";

interface RepositoryVelocityPanelProps {
  velocity: RepositoryVelocity;
  repositories: FactoryRepository[];
}

const panelClassName =
  "overflow-hidden rounded-lg border border-slate-950/15 bg-white dark:border-gray-700/70 dark:bg-gray-900";

const cohortPresentation: Record<VelocityCohortId, { icon: typeof Users; color: string }> = {
  team: { icon: Blend, color: "#334155" },
  human: { icon: Users, color: "#0284c7" },
  factory: { icon: Bot, color: "#d97706" },
};

const chartConfig = {
  team: { label: "Team total", color: cohortPresentation.team.color },
  human: { label: "Human-authored", color: cohortPresentation.human.color },
  factory: { label: "Factory-authored", color: cohortPresentation.factory.color },
} satisfies ChartConfig;

export function RepositoryVelocityPanel({ velocity, repositories }: RepositoryVelocityPanelProps) {
  const [repositoryId, setRepositoryId] = useState(velocity.defaultRepositoryId);
  const repository = repositories.find((item) => item.id === repositoryId) ?? repositories[0];

  return (
    <div className="space-y-4">
      <section className={panelClassName}>
        <VelocityHeader
          period={velocity.period}
          repositoryId={repository?.id ?? ""}
          repositories={repositories}
          onRepositoryChange={setRepositoryId}
        />
        <DeliveryIndicators cohorts={velocity.cohorts} metrics={velocity.metrics} />
      </section>

      <div className="grid gap-4 lg:grid-cols-[minmax(0,1.45fr)_minmax(19rem,0.55fr)]">
        <ThroughputTrend repository={repository?.name ?? "Selected repository"} velocity={velocity} />
        <CostScope velocity={velocity} />
      </div>
    </div>
  );
}

function VelocityHeader({
  period,
  repositoryId,
  repositories,
  onRepositoryChange,
}: {
  period: string;
  repositoryId: string;
  repositories: FactoryRepository[];
  onRepositoryChange: (repositoryId: string) => void;
}) {
  return (
    <div className="flex min-h-16 flex-col justify-between gap-3 border-b border-slate-200 px-4 py-3 sm:flex-row sm:items-center sm:px-5 dark:border-gray-700/70">
      <div className="min-w-0">
        <h2 className="text-sm font-semibold text-slate-900 dark:text-gray-100">Repository velocity</h2>
        <p className="mt-0.5 text-xs text-slate-500 dark:text-gray-400">
          Pull request delivery, reliability, and tracked execution cost.
        </p>
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <Select value={repositoryId} onValueChange={onRepositoryChange}>
          <SelectTrigger aria-label="Repository" className="h-8 w-[min(18rem,70vw)] bg-white dark:bg-gray-900">
            <GitBranch className="size-3.5" />
            <SelectValue placeholder="Select repository" />
          </SelectTrigger>
          <SelectContent>
            {repositories.map((repository) => (
              <SelectItem key={repository.id} value={repository.id}>
                {repository.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Badge
          variant="outline"
          className="h-8 shrink-0 bg-white px-2.5 text-slate-600 dark:bg-gray-900 dark:text-gray-300"
        >
          <CalendarDays />
          {period}
        </Badge>
      </div>
    </div>
  );
}

function DeliveryIndicators({ cohorts, metrics }: { cohorts: VelocityCohort[]; metrics: VelocityMetric[] }) {
  return (
    <div>
      <div className="px-4 py-3 sm:px-5">
        <h3 className="text-sm font-semibold text-slate-900 dark:text-gray-100">Delivery indicators</h3>
        <p className="mt-0.5 text-xs text-slate-500 dark:text-gray-400">
          The team total combines human-authored and Factory-authored pull requests.
        </p>
      </div>
      <div className="overflow-x-auto border-t border-slate-200 dark:border-gray-700/70">
        <table className="w-full min-w-[820px] border-collapse text-left">
          <thead>
            <tr className="border-b border-slate-200 bg-slate-50/80 dark:border-gray-700/70 dark:bg-gray-800/50">
              <th
                scope="col"
                className="w-[31%] px-4 py-3 text-[11px] font-medium text-slate-500 sm:px-5 dark:text-gray-400"
              >
                Metric
              </th>
              {cohorts.map((cohort) => (
                <CohortHeader key={cohort.id} cohort={cohort} />
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-200 dark:divide-gray-700/70">
            {metrics.map((metric) => (
              <MetricRow key={metric.id} metric={metric} cohorts={cohorts} />
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function CohortHeader({ cohort }: { cohort: VelocityCohort }) {
  const Icon = cohortPresentation[cohort.id].icon;

  return (
    <th scope="col" className="w-[23%] px-4 py-3 font-normal">
      <span className="flex items-center gap-2">
        <span className="flex size-7 shrink-0 items-center justify-center rounded-md bg-white text-slate-600 shadow-sm ring-1 ring-slate-200 dark:bg-gray-900 dark:text-gray-300 dark:ring-gray-700">
          <Icon className="size-3.5" />
        </span>
        <span className="min-w-0">
          <span className="block text-xs font-medium text-slate-800 dark:text-gray-200">{cohort.label}</span>
          <span className="block truncate text-[10px] text-slate-500 dark:text-gray-400">{cohort.description}</span>
        </span>
      </span>
    </th>
  );
}

function MetricRow({ metric, cohorts }: { metric: VelocityMetric; cohorts: VelocityCohort[] }) {
  return (
    <tr>
      <th scope="row" className="px-4 py-3.5 font-normal sm:px-5">
        <span className="block text-xs font-medium text-slate-800 dark:text-gray-200">{metric.label}</span>
        <span className="mt-0.5 block text-[11px] text-slate-500 dark:text-gray-400">{metric.description}</span>
      </th>
      {cohorts.map((cohort) => {
        const value = metric.values[cohort.id];
        return (
          <td key={cohort.id} className="px-4 py-3.5">
            <span
              className={cn(
                "block text-base font-semibold text-slate-900 tabular-nums dark:text-gray-100",
                value.unavailable && "text-slate-400 dark:text-gray-500",
              )}
            >
              {value.value}
            </span>
            <span className="mt-0.5 block text-[11px] text-slate-500 dark:text-gray-400">{value.detail}</span>
            {value.trend ? (
              <span className="mt-0.5 block text-[10px] text-emerald-700 dark:text-emerald-400">{value.trend}</span>
            ) : null}
          </td>
        );
      })}
    </tr>
  );
}

function ThroughputTrend({ repository, velocity }: { repository: string; velocity: RepositoryVelocity }) {
  return (
    <section className={panelClassName}>
      <div className="border-b border-slate-200 px-4 py-3 sm:px-5 dark:border-gray-700/70">
        <h3 className="text-sm font-semibold text-slate-900 dark:text-gray-100">Merged pull requests</h3>
        <p className="mt-0.5 truncate text-xs text-slate-500 dark:text-gray-400">Daily throughput for {repository}</p>
      </div>
      <div className="p-4 sm:p-5">
        <ChartContainer
          config={chartConfig}
          className="h-[240px] w-full"
          initialDimension={{ width: 720, height: 240 }}
        >
          <LineChart data={velocity.trend} margin={{ top: 6, right: 8, left: -24, bottom: 0 }}>
            <CartesianGrid vertical={false} />
            <XAxis dataKey="label" tickLine={false} axisLine={false} fontSize={11} minTickGap={24} />
            <YAxis tickLine={false} axisLine={false} fontSize={11} allowDecimals={false} />
            <ChartTooltip content={<ChartTooltipContent />} />
            <ChartLegend content={<ChartLegendContent />} />
            <Line
              type="monotone"
              dataKey="team"
              stroke="var(--color-team)"
              strokeWidth={2.5}
              dot={false}
              activeDot={{ r: 4 }}
            />
            <Line
              type="monotone"
              dataKey="human"
              stroke="var(--color-human)"
              strokeWidth={1.75}
              strokeDasharray="4 3"
              dot={false}
              activeDot={{ r: 3 }}
            />
            <Line
              type="monotone"
              dataKey="factory"
              stroke="var(--color-factory)"
              strokeWidth={2}
              dot={false}
              activeDot={{ r: 3 }}
            />
          </LineChart>
        </ChartContainer>
      </div>
    </section>
  );
}

function CostScope({ velocity }: { velocity: RepositoryVelocity }) {
  const { costBreakdown } = velocity;

  return (
    <section className={panelClassName}>
      <div className="border-b border-slate-200 px-4 py-3 sm:px-5 dark:border-gray-700/70">
        <h3 className="text-sm font-semibold text-slate-900 dark:text-gray-100">Tracked execution cost</h3>
        <p className="mt-0.5 text-xs text-slate-500 dark:text-gray-400">Factory work during this period.</p>
      </div>
      <dl className="divide-y divide-slate-200 dark:divide-gray-700/70">
        <CostRow icon={Coins} label="Model tokens" value={costBreakdown.tokens} />
        <CostRow icon={Cpu} label="Execution compute" value={costBreakdown.compute} />
        <CostRow icon={Blend} label="Total tracked" value={costBreakdown.total} strong />
      </dl>
      <div className="flex items-start gap-2 border-t border-slate-200 bg-slate-50/80 px-4 py-3 text-[11px] leading-4 text-slate-500 sm:px-5 dark:border-gray-700/70 dark:bg-gray-800/50 dark:text-gray-400">
        <Info className="mt-0.5 size-3.5 shrink-0" />
        <p>{costBreakdown.note}</p>
      </div>
    </section>
  );
}

function CostRow({
  icon: Icon,
  label,
  value,
  strong = false,
}: {
  icon: typeof Coins;
  label: string;
  value: string;
  strong?: boolean;
}) {
  return (
    <div className="flex items-center justify-between gap-4 px-4 py-3 sm:px-5">
      <dt className="flex items-center gap-2 text-xs text-slate-600 dark:text-gray-400">
        <Icon className="size-3.5" />
        {label}
      </dt>
      <dd
        className={cn(
          "text-xs font-medium text-slate-800 tabular-nums dark:text-gray-200",
          strong && "text-base font-semibold text-slate-900 dark:text-gray-100",
        )}
      >
        {value}
      </dd>
    </div>
  );
}
