import { useMemo } from "react";
import {
  Bar,
  BarChart,
  CartesianGrid,
  Layer,
  Rectangle,
  Sankey,
  Tooltip,
  XAxis,
  YAxis,
  type NodeProps,
  type LinkProps,
} from "recharts";
import {
  ChartContainer,
  ChartLegend,
  ChartLegendContent,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";
import { cn } from "@/lib/utils";

/** Storybook mock for the current 7-day window. */
export const WEEK_VELOCITY = {
  periodLabel: "Last 7 days",
  entered: 42,
  succeeded: 28,
  failed: 5,
} as const;

type WeekVelocity = typeof WEEK_VELOCITY;

const DEFAULT_STAGES = ["Plan", "Implement", "Review", "Ship"] as const;

const OUTCOME_SUCCEEDED = "Succeeded";
const OUTCOME_FAILED = "Failed";
const OUTCOME_OPEN = "Still in queue or running";

const OUTCOME_FILL: Record<string, string> = {
  [OUTCOME_SUCCEEDED]: "#10b981",
  [OUTCOME_FAILED]: "#ef4444",
  [OUTCOME_OPEN]: "#60a5fa",
};

const STAGE_FILL = "#64748b";
const PROGRESS_STROKE = "#64748b";

const sankeyChartConfig = {
  succeeded: { label: OUTCOME_SUCCEEDED, color: OUTCOME_FILL[OUTCOME_SUCCEEDED] },
  failed: { label: OUTCOME_FAILED, color: OUTCOME_FILL[OUTCOME_FAILED] },
  open: { label: OUTCOME_OPEN, color: OUTCOME_FILL[OUTCOME_OPEN] },
  stage: { label: "Stage", color: STAGE_FILL },
} satisfies ChartConfig;

function isOutcomeName(name: string) {
  return name === OUTCOME_SUCCEEDED || name === OUTCOME_FAILED || name === OUTCOME_OPEN;
}

function nodeFill(name: string) {
  if (isOutcomeName(name)) return OUTCOME_FILL[name];
  return STAGE_FILL;
}

function linkStroke(_sourceName: string, targetName: string) {
  if (isOutcomeName(targetName)) return nodeFill(targetName) ?? PROGRESS_STROKE;
  return PROGRESS_STROKE;
}

function resolveStages(stageNames: string[]) {
  return stageNames.length > 0 ? stageNames : [...DEFAULT_STAGES];
}

type SankeyGraph = {
  nodes: { name: string }[];
  links: { source: number; target: number; value: number }[];
};

type StageExitSplit = {
  entered: number[];
  midFailed: number[];
  midOpen: number[];
  terminalFailed: number;
  terminalOpen: number;
};

/** Mock split: some Failed / Still-open exit early; the rest resolve at the last stage. */
function splitStageExits(data: WeekVelocity, stageCount: number): StageExitSplit {
  const open = Math.max(0, data.entered - data.succeeded - data.failed);
  const gaps = Math.max(0, stageCount - 1);

  if (stageCount <= 1 || gaps === 0) {
    return {
      entered: [data.entered],
      midFailed: [],
      midOpen: [],
      terminalFailed: data.failed,
      terminalOpen: open,
    };
  }

  let midFailedBudget = Math.floor(data.failed * 0.4);
  let midOpenBudget = Math.floor(open * 0.55);
  const maxMid = Math.max(0, data.entered - data.succeeded - 1);
  if (midFailedBudget + midOpenBudget > maxMid) {
    const scale = maxMid / (midFailedBudget + midOpenBudget || 1);
    midFailedBudget = Math.floor(midFailedBudget * scale);
    midOpenBudget = Math.max(0, maxMid - midFailedBudget);
  }

  const terminalFailed = data.failed - midFailedBudget;
  const terminalOpen = open - midOpenBudget;
  const lastEntered = data.succeeded + terminalFailed + terminalOpen;

  const midFailed: number[] = [];
  const midOpen: number[] = [];
  let failedLeft = midFailedBudget;
  let openLeft = midOpenBudget;
  for (let i = 0; i < gaps; i += 1) {
    const isLastGap = i === gaps - 1;
    const failedHere = isLastGap ? failedLeft : Math.round(midFailedBudget / gaps);
    const openHere = isLastGap ? openLeft : Math.round(midOpenBudget / gaps);
    midFailed.push(Math.min(failedLeft, failedHere));
    midOpen.push(Math.min(openLeft, openHere));
    failedLeft -= midFailed[i]!;
    openLeft -= midOpen[i]!;
  }

  const entered: number[] = [data.entered];
  for (let i = 0; i < gaps; i += 1) {
    entered.push((entered[i] ?? 0) - (midFailed[i] ?? 0) - (midOpen[i] ?? 0));
  }
  entered[stageCount - 1] = lastEntered;
  if (stageCount >= 2) {
    const prev = entered[stageCount - 2] ?? lastEntered;
    const drop = Math.max(0, prev - lastEntered);
    const failedHere = Math.min(midFailed[gaps - 1] ?? 0, drop);
    midFailed[gaps - 1] = failedHere;
    midOpen[gaps - 1] = Math.max(0, drop - failedHere);
  }

  return { entered, midFailed, midOpen, terminalFailed, terminalOpen };
}

/**
 * Stage columns left → right. Failed / Still-in-queue from every stage merge into the
 * same three far-right outcomes. Succeeded only from the last stage.
 */
function buildMergedOutcomePipeline(data: WeekVelocity, stageNames: string[]): SankeyGraph {
  const stages = resolveStages(stageNames);
  const stageCount = stages.length;
  const open = Math.max(0, data.entered - data.succeeded - data.failed);

  if (stageCount === 1) {
    const nodes = [
      { name: `Entered ${stages[0]}` },
      { name: OUTCOME_SUCCEEDED },
      { name: OUTCOME_FAILED },
      { name: OUTCOME_OPEN },
    ];
    const links: SankeyGraph["links"] = [];
    if (data.succeeded > 0) links.push({ source: 0, target: 1, value: data.succeeded });
    if (data.failed > 0) links.push({ source: 0, target: 2, value: data.failed });
    if (open > 0) links.push({ source: 0, target: 3, value: open });
    return { nodes, links };
  }

  const { entered, midFailed, midOpen, terminalFailed, terminalOpen } = splitStageExits(data, stageCount);
  const gaps = stageCount - 1;

  const nodes: { name: string }[] = stages.map((name, index) => ({
    name: index === 0 ? `Entered ${name}` : name,
  }));
  const succIdx = nodes.length;
  nodes.push({ name: OUTCOME_SUCCEEDED });
  const failIdx = nodes.length;
  nodes.push({ name: OUTCOME_FAILED });
  const openIdx = nodes.length;
  nodes.push({ name: OUTCOME_OPEN });

  const links: SankeyGraph["links"] = [];
  for (let i = 0; i < gaps; i += 1) {
    const progress = entered[i + 1] ?? 0;
    if (progress > 0) links.push({ source: i, target: i + 1, value: progress });
    const failedHere = midFailed[i] ?? 0;
    const openHere = midOpen[i] ?? 0;
    if (failedHere > 0) links.push({ source: i, target: failIdx, value: failedHere });
    if (openHere > 0) links.push({ source: i, target: openIdx, value: openHere });
  }

  const last = stageCount - 1;
  if (data.succeeded > 0) links.push({ source: last, target: succIdx, value: data.succeeded });
  if (terminalFailed > 0) links.push({ source: last, target: failIdx, value: terminalFailed });
  if (terminalOpen > 0) links.push({ source: last, target: openIdx, value: terminalOpen });

  return { nodes, links };
}

type StageSuccessMetrics = {
  name: string;
  /** % of work that successfully entered the next stage. */
  successRate: number;
  avgDurationLabel: string;
  avgDurationCostUsd: number;
  avgTokens: number;
  avgTokenCostUsd: number;
};

type OutcomeSpendMetrics = {
  wellSpentPct: number;
  poorlySpentPct: number;
  wellSpentUsd: number;
  poorlySpentUsd: number;
  totalUsd: number;
};

function formatUsd(value: number) {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    maximumFractionDigits: value >= 100 ? 0 : 2,
  }).format(value);
}

function formatTokens(value: number) {
  if (value >= 1000) return `${(value / 1000).toFixed(value >= 10000 ? 0 : 1)}k`;
  return String(value);
}

/** Deterministic mock metrics aligned to the line’s stages + a terminal outcomes column. */
function buildStageSuccessMetrics(stageNames: string[]): {
  stages: StageSuccessMetrics[];
  outcomes: OutcomeSpendMetrics;
} {
  const stages = resolveStages(stageNames);
  const stageMetrics: StageSuccessMetrics[] = stages.map((name, index) => {
    const successRate = Math.max(55, 92 - index * 8 - (index === stages.length - 1 ? 4 : 0));
    const durationMinutes = 18 + index * 27 + (index % 2) * 6;
    const hours = durationMinutes / 60;
    const avgDurationCostUsd = Math.round(hours * 42 * 100) / 100;
    const avgTokens = 4200 + index * 2800 + (index === 0 ? 800 : 0);
    const avgTokenCostUsd = Math.round((avgTokens / 1000) * 0.85 * 100) / 100;
    const hoursPart = Math.floor(durationMinutes / 60);
    const minsPart = durationMinutes % 60;
    const avgDurationLabel = hoursPart > 0 ? `${hoursPart}h ${minsPart}m` : `${minsPart}m`;

    return {
      name,
      successRate,
      avgDurationLabel,
      avgDurationCostUsd,
      avgTokens,
      avgTokenCostUsd,
    };
  });

  const totalUsd =
    stageMetrics.reduce((sum, stage) => sum + stage.avgDurationCostUsd + stage.avgTokenCostUsd, 0) * 12;
  const wellSpentPct = 74;
  const poorlySpentPct = 26;

  return {
    stages: stageMetrics,
    outcomes: {
      wellSpentPct,
      poorlySpentPct,
      wellSpentUsd: Math.round(totalUsd * (wellSpentPct / 100)),
      poorlySpentUsd: Math.round(totalUsd * (poorlySpentPct / 100)),
      totalUsd: Math.round(totalUsd),
    },
  };
}

function StageMetricBlock({ stage }: { stage: StageSuccessMetrics }) {
  return (
    <>
      <div>
        <div className="text-[11px] text-muted-foreground">Success to next</div>
        <div className="mt-1 text-[22px] font-semibold tracking-[-0.03em] tabular-nums text-foreground">
          {stage.successRate}%
        </div>
      </div>
      <div>
        <div className="text-[11px] text-muted-foreground">Avg duration</div>
        <div className="mt-1 text-[15px] font-semibold tracking-[-0.02em] tabular-nums text-foreground">
          {stage.avgDurationLabel}
        </div>
        <div className="mt-0.5 text-[12px] tabular-nums text-muted-foreground">{formatUsd(stage.avgDurationCostUsd)}</div>
      </div>
      <div>
        <div className="text-[11px] text-muted-foreground">Avg tokens</div>
        <div className="mt-1 text-[15px] font-semibold tracking-[-0.02em] tabular-nums text-foreground">
          {formatTokens(stage.avgTokens)}
        </div>
        <div className="mt-0.5 text-[12px] tabular-nums text-muted-foreground">{formatUsd(stage.avgTokenCostUsd)}</div>
      </div>
    </>
  );
}

function OutcomesMetricBlock({ outcomes }: { outcomes: OutcomeSpendMetrics }) {
  return (
    <>
      <div>
        <div className="text-[11px] text-muted-foreground">Money well spent</div>
        <div className="mt-1 flex items-baseline gap-2">
          <span className="text-[22px] font-semibold tracking-[-0.03em] tabular-nums text-emerald-700 dark:text-emerald-300">
            {outcomes.wellSpentPct}%
          </span>
          <span className="text-[12px] tabular-nums text-muted-foreground">{formatUsd(outcomes.wellSpentUsd)}</span>
        </div>
      </div>
      <div>
        <div className="text-[11px] text-muted-foreground">Not well spent</div>
        <div className="mt-1 flex items-baseline gap-2">
          <span className="text-[22px] font-semibold tracking-[-0.03em] tabular-nums text-red-700 dark:text-red-300">
            {outcomes.poorlySpentPct}%
          </span>
          <span className="text-[12px] tabular-nums text-muted-foreground">{formatUsd(outcomes.poorlySpentUsd)}</span>
        </div>
      </div>
      <div>
        <div className="mb-1.5 flex items-center justify-between text-[11px] text-muted-foreground">
          <span>Spend mix</span>
          <span className="tabular-nums">{formatUsd(outcomes.totalUsd)}</span>
        </div>
        <div className="flex h-2 overflow-hidden rounded-full bg-muted">
          <span className="bg-emerald-500/80" style={{ width: `${outcomes.wellSpentPct}%` }} />
          <span className="bg-red-500/70" style={{ width: `${outcomes.poorlySpentPct}%` }} />
        </div>
      </div>
    </>
  );
}

/** Stage columns aligned to the pipeline (+ always an Outcomes column). */
function StageSuccessColumns({
  stages,
  outcomes,
}: {
  stages: StageSuccessMetrics[];
  outcomes: OutcomeSpendMetrics;
}) {
  const columns = stages.length + 1;

  return (
    <div
      className="overflow-hidden rounded-lg border border-border bg-card"
      style={{ display: "grid", gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))` }}
    >
      {stages.map((stage, index) => (
        <div
          key={stage.name}
          className={cn("flex flex-col gap-4 px-3.5 py-3.5", index > 0 && "border-l border-border")}
        >
          <div className="text-[12px] font-medium tracking-[-0.01em] text-foreground">{stage.name}</div>
          <StageMetricBlock stage={stage} />
        </div>
      ))}
      <div className="flex flex-col gap-4 border-l border-border bg-muted/30 px-3.5 py-3.5">
        <div className="text-[12px] font-medium tracking-[-0.01em] text-foreground">Outcomes</div>
        <OutcomesMetricBlock outcomes={outcomes} />
      </div>
    </div>
  );
}

function VelocitySankeyNode({ x, y, width, height, payload }: NodeProps) {
  const depth = payload.depth;
  const fill = nodeFill(payload.name) ?? STAGE_FILL;
  const labelOnLeft = depth === 0;
  const labelX = labelOnLeft ? x - 8 : x + width + 8;
  const textAnchor = labelOnLeft ? "end" : "start";

  return (
    <Layer>
      <Rectangle x={x} y={y} width={width} height={height} fill={fill} radius={2} />
      <text
        x={labelX}
        y={y + height / 2}
        textAnchor={textAnchor}
        dominantBaseline="middle"
        className="fill-foreground text-[11px] font-medium"
      >
        {payload.name}
      </text>
      <text
        x={labelX}
        y={y + height / 2 + 14}
        textAnchor={textAnchor}
        dominantBaseline="middle"
        className="fill-muted-foreground text-[11px] tabular-nums"
      >
        {payload.value}
      </text>
    </Layer>
  );
}

function VelocitySankeyLink(props: LinkProps) {
  const { sourceX, targetX, sourceY, targetY, sourceControlX, targetControlX, linkWidth, payload } = props;
  const stroke = linkStroke(payload.source.name, payload.target.name);

  return (
    <path
      d={`
        M${sourceX},${sourceY}
        C${sourceControlX},${sourceY} ${targetControlX},${targetY} ${targetX},${targetY}
      `}
      fill="none"
      stroke={stroke}
      strokeWidth={linkWidth}
      strokeOpacity={0.38}
    />
  );
}

const dailyVelocityChartConfig = {
  succeeded: { label: "Succeeded", color: "#10b981" },
  failed: { label: "Failed", color: "#ef4444" },
  inProgress: { label: "Still in progress", color: "#60a5fa" },
} satisfies ChartConfig;

type DailyVelocityPoint = {
  day: string;
  entered: number;
  succeeded: number;
  failed: number;
  inProgress: number;
};

/**
 * Seven daily points. Still-in-progress only on Sat/Sun; weekdays are
 * succeeded + failed only.
 */
function buildDailyVelocity(data: WeekVelocity): DailyVelocityPoint[] {
  const open = Math.max(0, data.entered - data.succeeded - data.failed);
  const dayLabels = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"];
  const intakeWeights = [0.12, 0.16, 0.18, 0.17, 0.15, 0.12, 0.1];
  // No in-progress mid-week; park open volume on the weekend.
  const inProgressShare = [0, 0, 0, 0, 0, 0.45, 0.55];

  const distribute = (total: number, weights: number[]) => {
    const values = weights.map((weight) => Math.round(total * weight));
    const drift = total - values.reduce((sum, value) => sum + value, 0);
    values[values.length - 1] = (values[values.length - 1] ?? 0) + drift;
    return values.map((value) => Math.max(0, value));
  };

  const entered = distribute(data.entered, intakeWeights);
  const closedTotal = data.succeeded + data.failed;
  const successShareOfClosed = closedTotal > 0 ? data.succeeded / closedTotal : 1;

  // First pass: weekday closes only; weekend keeps a share open.
  const points: DailyVelocityPoint[] = dayLabels.map((day, index) => {
    const dayEntered = entered[index] ?? 0;
    const isWeekend = index >= 5;
    let dayInProgress = isWeekend
      ? Math.min(dayEntered, Math.round(dayEntered * (inProgressShare[index] ?? 0)))
      : 0;
    let closed = dayEntered - dayInProgress;
    let daySucceeded = Math.round(closed * successShareOfClosed);
    let dayFailed = closed - daySucceeded;
    if (dayFailed < 0) {
      daySucceeded = closed;
      dayFailed = 0;
    }
    const used = daySucceeded + dayFailed + dayInProgress;
    if (used !== dayEntered) {
      if (isWeekend) {
        dayInProgress = Math.max(0, dayInProgress + (dayEntered - used));
      } else {
        daySucceeded = Math.max(0, daySucceeded + (dayEntered - used));
      }
    }
    return {
      day,
      entered: dayEntered,
      succeeded: daySucceeded,
      failed: dayFailed,
      inProgress: dayInProgress,
    };
  });

  const sum = (key: keyof Omit<DailyVelocityPoint, "day">) =>
    points.reduce((total, point) => total + point[key], 0);

  const shift = (
    from: "succeeded" | "failed" | "inProgress",
    to: "succeeded" | "failed" | "inProgress",
    amount: number,
    dayIndexes: number[],
  ) => {
    let remaining = amount;
    for (const index of dayIndexes) {
      if (remaining <= 0) break;
      const point = points[index];
      if (!point) continue;
      const move = Math.min(remaining, point[from]);
      if (move <= 0) continue;
      point[from] -= move;
      point[to] += move;
      remaining -= move;
    }
  };

  const weekend = [5, 6];
  const weekdays = [0, 1, 2, 3, 4];

  // Move open volume onto Sat/Sun only.
  let openDelta = open - sum("inProgress");
  if (openDelta > 0) {
    shift("succeeded", "inProgress", openDelta, weekend);
    openDelta = open - sum("inProgress");
    if (openDelta > 0) shift("failed", "inProgress", openDelta, weekend);
  } else if (openDelta < 0) {
    shift("inProgress", "succeeded", -openDelta, weekend);
  }

  // Clear any residual weekday in-progress from rebalancing.
  for (const index of weekdays) {
    const point = points[index];
    if (!point || point.inProgress === 0) continue;
    point.succeeded += point.inProgress;
    point.inProgress = 0;
  }

  let successDelta = data.succeeded - sum("succeeded");
  if (successDelta > 0) shift("failed", "succeeded", successDelta, weekdays);
  else if (successDelta < 0) shift("succeeded", "failed", -successDelta, weekdays);

  // Both weekend days should show still-in-progress when the week has open work.
  if (open > 0) {
    for (const index of weekend) {
      const point = points[index];
      if (!point || point.inProgress > 0) continue;
      if (point.succeeded > 0) {
        point.succeeded -= 1;
        point.inProgress += 1;
      } else if (point.failed > 0) {
        point.failed -= 1;
        point.inProgress += 1;
      }
    }
  }

  return points;
}

function DailyVelocityChart({ data }: { data: WeekVelocity }) {
  const points = useMemo(() => buildDailyVelocity(data), [data]);

  return (
    <div className="overflow-hidden rounded-lg border border-border bg-card px-3 py-3">
      <ChartContainer
        config={dailyVelocityChartConfig}
        className="aspect-auto h-[280px] w-full"
        initialDimension={{ width: 720, height: 280 }}
      >
        <BarChart data={points} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
          <CartesianGrid vertical={false} strokeDasharray="3 3" className="stroke-border" />
          <XAxis dataKey="day" tickLine={false} axisLine={false} tickMargin={8} className="text-[11px]" />
          <YAxis
            allowDecimals={false}
            tickLine={false}
            axisLine={false}
            width={28}
            tickMargin={4}
            className="text-[11px]"
          />
          <ChartTooltip content={<ChartTooltipContent />} />
          <ChartLegend content={<ChartLegendContent />} verticalAlign="bottom" />
          <Bar dataKey="succeeded" stackId="day" fill="var(--color-succeeded)" maxBarSize={36} />
          <Bar dataKey="failed" stackId="day" fill="var(--color-failed)" maxBarSize={36} />
          <Bar
            dataKey="inProgress"
            stackId="day"
            fill="var(--color-inprogress)"
            radius={[3, 3, 0, 0]}
            maxBarSize={36}
          />
        </BarChart>
      </ChartContainer>
    </div>
  );
}

type ThroughputStep = {
  key: string;
  label: string;
  value: number;
  color: string;
  pct: number;
};

function buildThroughputSteps(data: WeekVelocity): ThroughputStep[] {
  const open = Math.max(0, data.entered - data.succeeded - data.failed);
  const pctOfEntered = (value: number) =>
    data.entered > 0 ? Math.round((value / data.entered) * 1000) / 10 : 0;

  return [
    { key: "entered", label: "Entered", value: data.entered, color: "#64748b", pct: 100 },
    {
      key: "succeeded",
      label: "Succeeded",
      value: data.succeeded,
      color: "#10b981",
      pct: pctOfEntered(data.succeeded),
    },
    {
      key: "failed",
      label: "Failed",
      value: data.failed,
      color: "#ef4444",
      pct: pctOfEntered(data.failed),
    },
    {
      key: "open",
      label: "Still in progress",
      value: open,
      color: "#60a5fa",
      pct: pctOfEntered(open),
    },
  ];
}

/** Dub-like metric columns with proportional panels. */
function ThroughputFunnelColumns({ data }: { data: WeekVelocity }) {
  const steps = buildThroughputSteps(data);

  return (
    <div className="overflow-hidden rounded-lg border border-border bg-card">
      <div className="grid grid-cols-4 divide-x divide-border">
        {steps.map((step) => {
          const panelHeight = Math.max(48, Math.round(140 * (step.pct / 100)));
          return (
            <div key={step.key} className="flex min-w-0 flex-col pt-3.5">
              <div className="flex items-center gap-1.5 px-3.5">
                <span className="size-2 shrink-0 rounded-[2px]" style={{ backgroundColor: step.color }} aria-hidden />
                <span className="truncate text-[12px] font-medium tracking-[-0.01em] text-foreground">{step.label}</span>
              </div>
              <div className="mt-1.5 px-3.5 text-[28px] leading-none font-semibold tracking-[-0.03em] tabular-nums text-foreground">
                {step.value}
              </div>

              <div className="mt-4 flex flex-1 items-end">
                <div
                  className="relative flex w-full items-center justify-center overflow-hidden"
                  style={{
                    height: panelHeight,
                    backgroundColor: step.color,
                    boxShadow: `inset 0 0 0 1px color-mix(in srgb, ${step.color} 40%, transparent), 0 8px 24px color-mix(in srgb, ${step.color} 22%, transparent)`,
                  }}
                >
                  <span className="rounded-full bg-white/90 px-2 py-0.5 text-[11px] font-semibold tabular-nums tracking-[-0.01em] text-neutral-900 shadow-sm dark:bg-black/50 dark:text-white">
                    {step.pct}%
                  </span>
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

export function LineVelocityPanel({
  data = WEEK_VELOCITY,
  stageNames = [...DEFAULT_STAGES],
}: {
  data?: WeekVelocity;
  stageNames?: string[];
}) {
  const sankeyData = useMemo(() => buildMergedOutcomePipeline(data, stageNames), [data, stageNames]);
  const successMetrics = useMemo(() => buildStageSuccessMetrics(stageNames), [stageNames]);
  const stageCount = resolveStages(stageNames).length;
  const height = Math.max(300, 210 + stageCount * 32);

  return (
    <div className="space-y-8" data-testid="line-velocity-panel">
      <div className="flex items-baseline justify-between gap-3">
        <p className="text-[13px] text-muted-foreground">Work-order throughput for this line.</p>
        <span className="shrink-0 rounded-md border border-border bg-accent px-2 py-0.5 text-[11px] font-medium tracking-[-0.01em] text-muted-foreground">
          {data.periodLabel}
        </span>
      </div>

      <div className="space-y-4">
        <div>
          <h2 className="text-[15px] font-semibold tracking-[-0.01em] text-foreground">Daily velocity</h2>
          <p className="mt-1 text-[13px] text-muted-foreground">
            Stacked bars for succeeded, failed, and still in progress (weekend only).
          </p>
        </div>
        <DailyVelocityChart data={data} />
      </div>

      <div className="space-y-4">
        <ThroughputFunnelColumns data={data} />

        <div className="overflow-hidden rounded-lg border border-border bg-card px-2 py-3">
          <div className="mb-1 flex items-center justify-between px-2 text-[11px] text-muted-foreground">
            <span>Entered first stage</span>
            <span>Succeeded · Failed · Still in queue or running</span>
          </div>
          <ChartContainer
            config={sankeyChartConfig}
            className="aspect-auto w-full [&_.recharts-surface]:overflow-visible"
            style={{ height }}
            initialDimension={{ width: 720, height }}
          >
            <Sankey
              data={sankeyData}
              nodeWidth={12}
              nodePadding={24}
              linkCurvature={0.5}
              margin={{ top: 12, bottom: 12, left: 118, right: 150 }}
              node={VelocitySankeyNode}
              link={VelocitySankeyLink}
              sort={false}
              verticalAlign="top"
            >
              <Tooltip
                content={({ payload }) => {
                  const item = payload?.[0]?.payload as
                    | { source?: { name?: string }; target?: { name?: string }; value?: number }
                    | undefined;
                  if (!item?.value) return null;
                  const from = item.source?.name;
                  const to = item.target?.name;
                  if (!from || !to) return null;
                  return (
                    <div className="rounded-md border border-border bg-background px-2.5 py-1.5 text-[12px] shadow-sm">
                      <span className="text-foreground">
                        {from} → {to}
                      </span>
                      <span className="ml-2 font-medium tabular-nums text-foreground">{item.value}</span>
                    </div>
                  );
                }}
              />
            </Sankey>
          </ChartContainer>
        </div>
      </div>

      <div className="space-y-4">
        <div>
          <h2 className="text-[15px] font-semibold tracking-[-0.01em] text-foreground">Stage success</h2>
          <p className="mt-1 text-[13px] text-muted-foreground">
            Per-stage handoff rate, time, and token cost — plus Outcomes for money well spent vs not.
          </p>
        </div>
        <StageSuccessColumns stages={successMetrics.stages} outcomes={successMetrics.outcomes} />
      </div>
    </div>
  );
}
