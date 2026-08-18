import type { FactoriesFactoryLine } from "@/api-client";
import { cn } from "@/lib/utils";
import { Workflow } from "lucide-react";
import type { KeyboardEvent, ReactNode } from "react";
import { useNavigate } from "react-router";
import { humanizeLineName } from "../lib/humanizeLineName";
import { factoryCardClassName } from "./factoryPageLayoutStyles";
import {
  formatCostDelta,
  formatCostPerSuccess,
  formatReworkDelta,
  formatReworkRate,
  formatSuccessDelta,
  formatSuccessRate,
  formatThroughput,
} from "./lineListMetricsFormat";
import type { LineListMetrics } from "./lineListMetrics";

export function LineListCard({
  line,
  href,
  metrics,
  description,
}: {
  line: FactoriesFactoryLine;
  href: string;
  metrics: LineListMetrics | null;
  description?: string;
}) {
  return (
    <CardShell href={href} testId={`lines-card-${line.id}`} className="px-4 py-4">
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-2">
          <Workflow className="size-4 shrink-0 text-muted-foreground" strokeWidth={1.75} aria-hidden />
          <span className="text-[15px] font-medium tracking-[-0.01em] text-foreground">
            {humanizeLineName(line.name)}
          </span>
        </div>
        {description ? <p className="mt-1 text-[12px] leading-snug text-muted-foreground">{description}</p> : null}
      </div>
      <LineListHeroSplit metrics={metrics} />
    </CardShell>
  );
}

/** Success sparkline on the left. Completion bars on the right. */
export function LineListHeroSplit({ metrics }: { metrics: LineListMetrics | null }) {
  return (
    <div className="mt-4 flex items-end justify-between gap-6" data-testid="lines-card-metrics">
      <div className="grid min-w-0 flex-1 grid-cols-2 gap-6">
        <div className="min-w-0">
          <p className="text-[11px] text-muted-foreground">Success rate</p>
          <div className="mt-1 flex items-end gap-3">
            <p className="text-[28px] leading-none font-semibold tracking-[-0.04em] tabular-nums text-foreground">
              {formatSuccessRate(metrics)}
            </p>
            <p className="pb-0.5 text-[12px] tabular-nums text-muted-foreground">{formatSuccessDelta(metrics)}</p>
          </div>
          <Sparkline values={metrics?.successTrendPct} className="mt-3 h-10 w-full text-foreground" />
        </div>
        <div className="min-w-0">
          <p className="text-[11px] text-muted-foreground">Completions</p>
          <p className="mt-1 text-[28px] leading-none font-semibold tracking-[-0.04em] tabular-nums text-foreground">
            {formatThroughput(metrics)}
          </p>
          <ThroughputBars values={metrics?.throughputTrend} className="mt-3 h-10 w-full text-muted-foreground" />
        </div>
      </div>
      <div className="flex shrink-0 flex-col gap-3 border-l border-border pl-5">
        <MiniStat label="Rework" value={formatReworkRate(metrics)} hint={formatReworkDelta(metrics)} />
        <MiniStat label="Cost" value={formatCostPerSuccess(metrics)} hint={formatCostDelta(metrics)} />
      </div>
    </div>
  );
}

function MiniStat({ label, value, hint }: { label: string; value: string; hint: string }) {
  return (
    <div>
      <p className="text-[11px] text-muted-foreground">{label}</p>
      <p className="mt-0.5 text-[15px] font-medium tracking-[-0.02em] tabular-nums text-foreground">{value}</p>
      <p className="text-[11px] tabular-nums text-muted-foreground">{hint}</p>
    </div>
  );
}

function Sparkline({ values, className }: { values: number[] | undefined; className?: string }) {
  if (!values || values.length < 2) {
    return <span className={cn("block", className)} aria-hidden />;
  }

  const width = 120;
  const height = 32;
  const min = Math.min(...values);
  const max = Math.max(...values);
  const range = max - min || 1;
  const points = values
    .map((value, index) => {
      const x = (index / (values.length - 1)) * width;
      const y = height - ((value - min) / range) * (height - 2) - 1;
      return `${x.toFixed(1)},${y.toFixed(1)}`;
    })
    .join(" ");

  return (
    <svg viewBox={`0 0 ${width} ${height}`} className={className} aria-hidden preserveAspectRatio="none">
      <polyline fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" points={points} />
    </svg>
  );
}

function ThroughputBars({ values, className }: { values: number[] | undefined; className?: string }) {
  if (!values || values.length === 0) {
    return <span className={cn("block", className)} aria-hidden />;
  }

  const width = 120;
  const height = 32;
  const max = Math.max(...values, 1);
  const gap = 1.2;
  const barWidth = Math.max(2, width / values.length - gap);

  return (
    <svg viewBox={`0 0 ${width} ${height}`} className={className} aria-hidden preserveAspectRatio="none">
      {values.map((value, index) => {
        const barHeight = (value / max) * (height - 2);
        const x = (index / values.length) * width;
        return (
          <rect
            key={index}
            x={x}
            y={height - barHeight}
            width={barWidth}
            height={barHeight}
            className="fill-current"
            opacity={0.45}
          />
        );
      })}
    </svg>
  );
}

function CardShell({
  href,
  testId,
  className,
  children,
}: {
  href: string;
  testId: string;
  className?: string;
  children: ReactNode;
}) {
  const navigate = useNavigate();

  return (
    <div
      role="button"
      tabIndex={0}
      className={cn(
        factoryCardClassName,
        "group/line w-full cursor-pointer text-left transition-colors",
        "hover:border-foreground/25 hover:bg-accent/30",
        className,
      )}
      onClick={() => navigate(href)}
      onKeyDown={(event: KeyboardEvent<HTMLDivElement>) => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          navigate(href);
        }
      }}
      data-testid={testId}
    >
      {children}
    </div>
  );
}
