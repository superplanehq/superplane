import { useMemo, useState } from "react";
import { ArrowDownRight, ArrowUpRight } from "lucide-react";
import { Area, AreaChart, Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts";

import {
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";
import { formatCompactTokenValue } from "@/lib/formatTokenCount";
import { cn } from "@/lib/utils";
import { SegmentedNav } from "@/ui/SegmentedNav";

import { WorkspacePageHeader } from "../layout/WorkspacePageHeader";
import { factoryCenteredSectionBodyClassName, factoryCenteredSectionHeaderClassName } from "./factoryPageLayoutStyles";
import { VelocityPeopleTable, type VelocityPerson } from "./VelocityPeopleTable";

type PeriodDays = 14 | 30;
type Breakdown = "origin" | "outcome" | "intake";

/** Workspace setup pins one app repository, so Velocity reports on that repository. */
const WORKSPACE_REPOSITORY = "acme/refunds";

interface VelocityPoint {
  day: string;
  people: number;
  superplane: number;
  merged: number;
  waste: number;
  githubIssues: number;
  sentryExceptions: number;
  manual: number;
  api: number;
  runningHours: number;
  waitingHours: number;
  costUsd: number;
  tokenCostUsd: number;
  computeCostUsd: number;
  wasteCostUsd: number;
  tokens: number;
}

interface BreakdownSeries {
  key: keyof VelocityPoint;
  label: string;
  color: string;
}

const PERIOD_OPTIONS = [
  { value: "14", label: "14d" },
  { value: "30", label: "30d" },
];

const BREAKDOWN_OPTIONS = [
  { value: "origin", label: "Origin" },
  { value: "outcome", label: "Outcome" },
  { value: "intake", label: "Intake source" },
];

const BREAKDOWN_SERIES: Record<Breakdown, BreakdownSeries[]> = {
  origin: [
    { key: "people", label: "People", color: "#64748b" },
    { key: "superplane", label: "SuperPlane", color: "#10b981" },
  ],
  outcome: [
    { key: "merged", label: "Merged", color: "#10b981" },
    { key: "waste", label: "Closed without merge", color: "#ef4444" },
  ],
  intake: [
    { key: "githubIssues", label: "GitHub issue", color: "#3b82f6" },
    { key: "sentryExceptions", label: "Sentry exception", color: "#8b5cf6" },
    { key: "manual", label: "Manually created", color: "#f59e0b" },
    { key: "api", label: "API", color: "#06b6d4" },
  ],
};

const BREAKDOWN_COPY: Record<Breakdown, { title: string; description: string }> = {
  origin: {
    title: "Merged pull requests by origin",
    description: "Team output split between people and SuperPlane.",
  },
  outcome: {
    title: "Pull requests by outcome",
    description: "Merged pull requests and Factory pull requests closed without merge.",
  },
  intake: {
    title: "Factory merges by intake source",
    description: "Merged pull requests grouped by how their tasks entered the Factory.",
  },
};

function githubAvatar(accountId: number): string {
  return `https://avatars.githubusercontent.com/u/${accountId}?v=4&s=64`;
}

/**
 * Per-member share of the period. Avatars use the GitHub account id, which is
 * how the connected provider serves them in production.
 */
const PEOPLE_SHARE: Array<
  Omit<VelocityPerson, "id" | "authoredMerged" | "factoryMerged" | "factoryWaste" | "costUsd"> & {
    id: string;
    authoredShare: number;
    factoryShare: number;
    wasteShare: number;
  }
> = [
  { id: "darkofabijan", name: "Darko Fabijan", email: "darko@superplane.com", avatarUrl: githubAvatar(20469), authoredShare: 0.24, factoryShare: 0.32, wasteShare: 0.18, medianCycleHours: 16 },
  { id: "AleksandarCole", name: "Aleksandar Mitrovic", email: "alex@superplane.com", avatarUrl: githubAvatar(61409859), authoredShare: 0.2, factoryShare: 0.2, wasteShare: 0.14, medianCycleHours: 21 },
  { id: "shiroyasha", name: "Igor Šarčević", email: "igor@superplane.com", avatarUrl: githubAvatar(1779493), authoredShare: 0.16, factoryShare: 0.16, wasteShare: 0.15, medianCycleHours: 19 },
  { id: "andrecalil", name: "André Calil", email: "andre@superplane.com", avatarUrl: githubAvatar(1105923), authoredShare: 0.14, factoryShare: 0.13, wasteShare: 0.2, medianCycleHours: 28 },
  { id: "forestileao", name: "Pedro Leão", email: "pedro@superplane.com", avatarUrl: githubAvatar(60622592), authoredShare: 0.12, factoryShare: 0.1, wasteShare: 0.18, medianCycleHours: 24 },
  { id: "markoa", name: "Marko Anastasov", email: "marko@superplane.com", avatarUrl: githubAvatar(8651), authoredShare: 0.08, factoryShare: 0.06, wasteShare: 0.1, medianCycleHours: 31 },
  { id: "lucaspin", name: "Lucas Pinheiro", email: "lucas@superplane.com", avatarUrl: githubAvatar(12387728), authoredShare: 0.06, factoryShare: 0.03, wasteShare: 0.05, medianCycleHours: 12 },
];

const flowChartConfig = {
  runningHours: { label: "Time running", color: "#60a5fa" },
  waitingHours: { label: "Time waiting", color: "#f59e0b" },
} satisfies ChartConfig;

const costChartConfig = {
  tokenCostUsd: { label: "Model tokens", color: "#64748b" },
  computeCostUsd: { label: "Execution compute", color: "#6366f1" },
} satisfies ChartConfig;

const cardClassName = "rounded-xl border border-border bg-card px-4 py-4 sm:px-5 sm:py-5";
const cardTitleClassName = "text-[14px] font-medium tracking-[-0.01em] text-foreground";
const cardSubtitleClassName = "mt-0.5 text-[12px] text-muted-foreground";

/**
 * `windowOffset` shifts the mock series along the timeline, so the current and
 * previous windows are different slices. Higher offset is more recent.
 *
 * The series drifts on purpose: Factory output ramps up while review time grows.
 * That gives the prototype a mixed result instead of every metric improving.
 */
function buildVelocityPoints(periodDays: PeriodDays, windowOffset = 0): VelocityPoint[] {
  return Array.from({ length: periodDays }, (_, index) => {
    const day = index + windowOffset;
    const people = Math.max(1, Math.round(8 + 2.6 * Math.sin(day * 0.8)));
    const superplane = Math.max(1, Math.round(4 + 1.8 * Math.sin(day * 0.55 + 1) + day * 0.09));
    const merged = people + superplane;
    const waste = day % 6 === 2 ? Math.max(1, Math.round(superplane * 0.35)) : day % 4 === 0 ? 1 : 0;
    const githubIssues = Math.round(superplane * 0.43);
    const sentryExceptions = Math.round(superplane * 0.27);
    const manual = Math.round(superplane * 0.18);
    const api = Math.max(0, superplane - githubIssues - sentryExceptions - manual);
    const runningHours = 8 + 3 * Math.sin(day * 0.48 + 0.5);
    const waitingHours = 13 + 5 * Math.sin(day * 0.35 + 1.8) + day * 0.18;
    const mergedCostUsd = superplane * 2.14;
    const wasteCostUsd = waste * 1.85;
    const costUsd = mergedCostUsd + wasteCostUsd;
    const tokenCostUsd = Math.round((superplane * 1.67 + waste * 1.44) * 100) / 100;
    const computeCostUsd = Math.round((costUsd - tokenCostUsd) * 100) / 100;

    return {
      day: periodDays === 14 ? `${index + 1}` : index % 5 === 0 || index === periodDays - 1 ? `${index + 1}` : "",
      people,
      superplane,
      merged,
      waste,
      githubIssues,
      sentryExceptions,
      manual,
      api,
      runningHours: Math.max(2, Math.round(runningHours * 10) / 10),
      waitingHours: Math.max(3, Math.round(waitingHours * 10) / 10),
      costUsd: Math.round(costUsd * 100) / 100,
      tokenCostUsd,
      computeCostUsd,
      wasteCostUsd: Math.round(wasteCostUsd * 100) / 100,
      tokens: superplane * 18_500 + waste * 12_400,
    };
  });
}

/**
 * Splits a period total across members with the largest-remainder method, so
 * the People table always adds up to the totals shown above it.
 */
function distribute(total: number, shares: number[]): number[] {
  const exact = shares.map((share) => total * share);
  const floors = exact.map((value) => Math.floor(value));
  let remainder = total - floors.reduce((sum, value) => sum + value, 0);

  const order = exact
    .map((value, index) => ({ index, fraction: value - Math.floor(value) }))
    .sort((a, b) => b.fraction - a.fraction);

  const result = [...floors];
  for (const { index } of order) {
    if (remainder <= 0) break;
    result[index] = (result[index] ?? 0) + 1;
    remainder -= 1;
  }
  return result;
}

function buildPeople(totals: {
  peopleMerged: number;
  superplaneMerged: number;
  waste: number;
  costUsd: number;
}): VelocityPerson[] {
  const authored = distribute(
    totals.peopleMerged,
    PEOPLE_SHARE.map((person) => person.authoredShare),
  );
  const factoryMerged = distribute(
    totals.superplaneMerged,
    PEOPLE_SHARE.map((person) => person.factoryShare),
  );
  const factoryWaste = distribute(
    totals.waste,
    PEOPLE_SHARE.map((person) => person.wasteShare),
  );

  return PEOPLE_SHARE.map((person, index) => ({
    id: person.id,
    name: person.name,
    email: person.email,
    avatarUrl: person.avatarUrl,
    authoredMerged: authored[index] ?? 0,
    factoryMerged: factoryMerged[index] ?? 0,
    factoryWaste: factoryWaste[index] ?? 0,
    medianCycleHours: person.medianCycleHours,
    costUsd: totals.costUsd * person.factoryShare,
  }));
}

function sum(points: VelocityPoint[], field: keyof VelocityPoint): number {
  return points.reduce((total, point) => {
    const value = point[field];
    return total + (typeof value === "number" ? value : 0);
  }, 0);
}

function median(values: number[]): number {
  const sorted = [...values].sort((a, b) => a - b);
  const middle = Math.floor(sorted.length / 2);
  if (sorted.length % 2 === 0) return ((sorted[middle - 1] ?? 0) + (sorted[middle] ?? 0)) / 2;
  return sorted[middle] ?? 0;
}

function formatHours(value: number) {
  return `${Math.round(value)}h`;
}

function formatUsd(value: number) {
  return `$${value.toFixed(2)}`;
}

type ChangeBetter = "up" | "down";

function summarizePoints(points: VelocityPoint[]) {
  const merged = sum(points, "merged");
  const superplaneMerged = sum(points, "superplane");
  const waste = sum(points, "waste");
  const cost = sum(points, "costUsd");
  const cycleHours = median(points.map((point) => point.runningHours + point.waitingHours));
  const wasteRate = superplaneMerged + waste > 0 ? Math.round((waste / (superplaneMerged + waste)) * 100) : 0;
  const costPerMerge = superplaneMerged > 0 ? cost / superplaneMerged : 0;

  return {
    merged,
    peopleMerged: sum(points, "people"),
    superplaneMerged,
    waste,
    wasteRate,
    cycleHours,
    runningHours: median(points.map((point) => point.runningHours)),
    waitingHours: median(points.map((point) => point.waitingHours)),
    cost,
    tokenCost: sum(points, "tokenCostUsd"),
    computeCost: sum(points, "computeCostUsd"),
    wasteCost: sum(points, "wasteCostUsd"),
    tokens: sum(points, "tokens"),
    costPerMerge,
  };
}

/** The arrow shows which way the number moved. The color shows whether that is good. */
interface MetricChange {
  text: string;
  tone: "better" | "worse" | "same";
  direction: "up" | "down" | "flat";
}

const CHANGE_TONE_CLASS: Record<MetricChange["tone"], string> = {
  better: "bg-emerald-50 text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-400",
  worse: "bg-rose-50 text-rose-700 dark:bg-rose-950/60 dark:text-rose-400",
  same: "bg-muted text-muted-foreground",
};

function buildChange(delta: number, better: ChangeBetter, format: (magnitude: number) => string): MetricChange {
  if (delta === 0) return { text: "No change", tone: "same", direction: "flat" };

  const improved = better === "up" ? delta > 0 : delta < 0;
  return {
    text: format(Math.abs(delta)),
    tone: improved ? "better" : "worse",
    direction: delta > 0 ? "up" : "down",
  };
}

function ChangeChip({ change }: { change: MetricChange }) {
  const Icon = change.direction === "up" ? ArrowUpRight : ArrowDownRight;

  return (
    <span
      className={cn(
        "inline-flex items-center gap-0.5 rounded-full px-1.5 py-0.5 text-[11px] font-medium tabular-nums",
        CHANGE_TONE_CLASS[change.tone],
      )}
    >
      {change.direction === "flat" ? null : <Icon className="size-3" aria-hidden />}
      {change.text}
    </span>
  );
}

function Metric({
  label,
  value,
  hint,
  change,
}: {
  label: string;
  value: string;
  hint?: string;
  change?: MetricChange;
}) {
  return (
    <div className="min-w-0">
      <p className="text-[12px] text-muted-foreground">{label}</p>
      <div className="mt-2 flex flex-wrap items-baseline gap-x-2 gap-y-1">
        <p className="text-[30px] leading-none font-semibold tracking-[-0.04em] tabular-nums text-foreground">
          {value}
        </p>
        {change ? <ChangeChip change={change} /> : null}
      </div>
      {hint ? <p className="mt-2 text-[12px] text-muted-foreground">{hint}</p> : null}
    </div>
  );
}

function DeliveryChart({ points, breakdown }: { points: VelocityPoint[]; breakdown: Breakdown }) {
  const series = BREAKDOWN_SERIES[breakdown];
  const config = Object.fromEntries(
    series.map((item) => [item.key, { label: item.label, color: item.color }]),
  ) satisfies ChartConfig;

  return (
    <ChartContainer
      config={config}
      className="aspect-auto h-[260px] w-full"
      initialDimension={{ width: 760, height: 260 }}
    >
      <BarChart data={points} margin={{ top: 8, right: 4, left: 0, bottom: 0 }}>
        <CartesianGrid vertical={false} strokeDasharray="3 3" className="stroke-border" />
        <XAxis dataKey="day" tickLine={false} axisLine={false} tickMargin={8} interval={0} className="text-[11px]" />
        <YAxis allowDecimals={false} tickLine={false} axisLine={false} width={28} className="text-[11px]" />
        <ChartTooltip content={<ChartTooltipContent />} />
        <ChartLegend content={<ChartLegendContent />} verticalAlign="bottom" />
        {series.map((item, index) => (
          <Bar
            key={item.key}
            dataKey={item.key}
            name={item.label}
            stackId="day"
            fill={item.color}
            radius={index === series.length - 1 ? [3, 3, 0, 0] : undefined}
            maxBarSize={24}
          />
        ))}
      </BarChart>
    </ChartContainer>
  );
}

function FlowChart({ points }: { points: VelocityPoint[] }) {
  return (
    <ChartContainer
      config={flowChartConfig}
      className="aspect-auto h-[180px] w-full"
      initialDimension={{ width: 500, height: 180 }}
    >
      <AreaChart data={points} margin={{ top: 8, right: 4, left: 0, bottom: 0 }}>
        <CartesianGrid vertical={false} strokeDasharray="3 3" className="stroke-border" />
        <XAxis dataKey="day" tickLine={false} axisLine={false} tickMargin={8} interval={0} className="text-[11px]" />
        <YAxis
          tickLine={false}
          axisLine={false}
          width={36}
          className="text-[11px]"
          tickFormatter={(value: number) => `${value}h`}
        />
        <ChartTooltip content={<ChartTooltipContent />} />
        <Area
          type="monotone"
          dataKey="runningHours"
          stackId="time"
          stroke="#60a5fa"
          fill="#60a5fa"
          fillOpacity={0.3}
        />
        <Area
          type="monotone"
          dataKey="waitingHours"
          stackId="time"
          stroke="#f59e0b"
          fill="#f59e0b"
          fillOpacity={0.3}
        />
      </AreaChart>
    </ChartContainer>
  );
}

function CostChart({ points }: { points: VelocityPoint[] }) {
  return (
    <ChartContainer
      config={costChartConfig}
      className="aspect-auto h-[180px] w-full"
      initialDimension={{ width: 500, height: 180 }}
    >
      <AreaChart data={points} margin={{ top: 8, right: 4, left: 0, bottom: 0 }}>
        <CartesianGrid vertical={false} strokeDasharray="3 3" className="stroke-border" />
        <XAxis dataKey="day" tickLine={false} axisLine={false} tickMargin={8} interval={0} className="text-[11px]" />
        <YAxis
          tickLine={false}
          axisLine={false}
          width={34}
          className="text-[11px]"
          tickFormatter={(value: number) => `$${value}`}
        />
        <ChartTooltip content={<ChartTooltipContent />} />
        <ChartLegend content={<ChartLegendContent />} verticalAlign="bottom" />
        <Area
          type="monotone"
          dataKey="tokenCostUsd"
          stackId="cost"
          stroke="#64748b"
          fill="#64748b"
          fillOpacity={0.28}
          strokeWidth={1.5}
        />
        <Area
          type="monotone"
          dataKey="computeCostUsd"
          stackId="cost"
          stroke="#6366f1"
          fill="#6366f1"
          fillOpacity={0.28}
          strokeWidth={1.5}
        />
      </AreaChart>
    </ChartContainer>
  );
}

export function VelocityPrototypePage() {
  const [periodDays, setPeriodDays] = useState<PeriodDays>(14);
  const [breakdown, setBreakdown] = useState<Breakdown>("origin");
  const points = useMemo(() => buildVelocityPoints(periodDays, periodDays), [periodDays]);
  const previousPoints = useMemo(() => buildVelocityPoints(periodDays, 0), [periodDays]);

  const current = summarizePoints(points);
  const previous = summarizePoints(previousPoints);
  const mergedDelta = current.merged - previous.merged;
  const wasteDelta = current.wasteRate - previous.wasteRate;
  const cycleDelta = Math.round(current.cycleHours - previous.cycleHours);
  const costDelta = Math.round((current.costPerMerge - previous.costPerMerge) * 100) / 100;
  const periodLabel = `Last ${periodDays} days`;
  const breakdownCopy = BREAKDOWN_COPY[breakdown];
  const people = useMemo(
    () =>
      buildPeople({
        peopleMerged: current.peopleMerged,
        superplaneMerged: current.superplaneMerged,
        waste: current.waste,
        costUsd: current.cost,
      }),
    [current.peopleMerged, current.superplaneMerged, current.waste, current.cost],
  );

  return (
    /* Gray page, white cards. Dark mode keeps its darker page, because the
       factories theme paints the sidebar and cards the same color. */
    <div className="min-h-full bg-sidebar dark:bg-background">
      <WorkspacePageHeader
        className={cn(factoryCenteredSectionHeaderClassName, "bg-transparent")}
        title="Velocity"
        subtitle={`What ${WORKSPACE_REPOSITORY} ships, how long the work takes, and what it costs.`}
        actions={
          <SegmentedNav
            ariaLabel="Velocity period"
            size="xs"
            value={String(periodDays)}
            onValueChange={(value) => setPeriodDays(value === "30" ? 30 : 14)}
            options={PERIOD_OPTIONS}
          />
        }
      />

      <div className={cn(factoryCenteredSectionBodyClassName, "space-y-5 pb-10")} data-testid="factory-velocity-page">
        <section className={cardClassName}>
          <p className="text-[12px] text-muted-foreground">
            {periodLabel}. Compared with the previous {periodDays} days.
          </p>
          <div className="mt-4 grid grid-cols-2 gap-x-8 gap-y-6 lg:grid-cols-4">
            <Metric
              label="Merged PRs"
              value={String(current.merged)}
              hint="People and SuperPlane"
              change={buildChange(mergedDelta, "up", (magnitude) => String(magnitude))}
            />
            <Metric
              label="Factory waste"
              value={`${current.wasteRate}%`}
              hint={`${current.waste} PRs closed without merge`}
              change={buildChange(wasteDelta, "down", (magnitude) => `${magnitude} pp`)}
            />
            <Metric
              label="Median cycle time"
              value={formatHours(current.cycleHours)}
              hint="From start to close"
              change={buildChange(cycleDelta, "down", (magnitude) => `${magnitude}h`)}
            />
            <Metric
              label="Cost per Factory merge"
              value={formatUsd(current.costPerMerge)}
              hint="Tokens and execution compute"
              change={buildChange(costDelta, "down", (magnitude) => `$${magnitude.toFixed(2)}`)}
            />
          </div>
        </section>

        <section className={cardClassName}>
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h2 className={cardTitleClassName}>{breakdownCopy.title}</h2>
              <p className={cardSubtitleClassName}>{breakdownCopy.description}</p>
            </div>
            <SegmentedNav
              ariaLabel="Group merged pull requests by"
              size="xs"
              value={breakdown}
              onValueChange={(value) => setBreakdown(value as Breakdown)}
              options={BREAKDOWN_OPTIONS}
            />
          </div>
          <div className="mt-5">
            <DeliveryChart points={points} breakdown={breakdown} />
          </div>
        </section>

        <VelocityPeopleTable people={people} periodLabel={periodLabel} />

        <div className="grid gap-5 lg:grid-cols-2">
          <section className={cardClassName}>
            <h2 className={cardTitleClassName}>Task time</h2>
            <p className={cardSubtitleClassName}>Median time for Factory tasks that closed in this period.</p>
            <div className="mt-5 grid grid-cols-3 gap-5">
              <Metric label="Cycle time" value={formatHours(current.cycleHours)} />
              <Metric label="Running" value={formatHours(current.runningHours)} />
              <Metric label="Waiting" value={formatHours(current.waitingHours)} />
            </div>
            <div className="mt-5 border-t border-border pt-4">
              <FlowChart points={points} />
            </div>
          </section>

          <section className={cardClassName}>
            <h2 className={cardTitleClassName}>Tracked Factory cost</h2>
            <p className={cardSubtitleClassName}>Model tokens and execution compute. Third-party charges are excluded.</p>
            <div className="mt-5 grid grid-cols-2 gap-5">
              <Metric label="Total cost" value={formatUsd(current.cost)} />
              <Metric
                label="Spend on waste"
                value={formatUsd(current.wasteCost)}
                hint="Factory work that closed without a merge"
              />
              <Metric
                label="Model tokens"
                value={formatUsd(current.tokenCost)}
                hint={`${formatCompactTokenValue(current.tokens)} tokens`}
              />
              <Metric label="Execution compute" value={formatUsd(current.computeCost)} />
            </div>
            <div className="mt-5 border-t border-border pt-4">
              <CostChart points={points} />
            </div>
          </section>
        </div>
      </div>
    </div>
  );
}
