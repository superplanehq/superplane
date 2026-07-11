import { useMemo, type ReactNode } from "react";
import { ArrowDownRight, ArrowUpRight, Hash, Loader2 } from "lucide-react";

import { cn } from "@/lib/utils";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Timestamp, type TimestampDisplay } from "@/components/Timestamp";

import { WidgetEmptyState } from "../WidgetEmptyState";
import { CONSOLE_WIDGET_LABEL_CLASSES } from "../consoleTableStyles";

import { Sparkline } from "./Sparkline";
import { aggregateNumber, applyFilters } from "./widgetData";
import { coerceWidgetTimestamp, formatValue } from "./widgetFormat";
import type { WidgetColumnFormat, WidgetScorecardRender } from "./types";
import {
  computeScorecardChange,
  computeScorecardProgress,
  extractScorecardSeries,
  formatScorecardChangeLabel,
  pickChangeAnchors,
  resolveScorecardStatus,
  resolveScorecardTarget,
  type ScorecardProgress,
  type ScorecardStatusPolarity,
} from "./scorecardMath";
import { formatTrendTooltip, type TrendResult } from "./widgetTrend";

const TIMESTAMP_FORMATS: Record<"date" | "datetime" | "relative", TimestampDisplay> = {
  date: "date",
  datetime: "datetime",
  relative: "relative",
};

function isTimestampFormat(format: WidgetColumnFormat | undefined): format is "date" | "datetime" | "relative" {
  return format === "date" || format === "datetime" || format === "relative";
}

interface WidgetScorecardProps {
  render: WidgetScorecardRender;
  rows: unknown[];
  isLoading: boolean;
  totalCount?: number;
}

const VALUE_CLASS = "text-4xl font-medium text-slate-900 dark:text-gray-100";
const SUFFIX_CLASS = "text-xl font-semibold text-slate-900 dark:text-gray-100";

const STATUS_TEXT: Record<ScorecardStatusPolarity, string> = {
  better: "text-emerald-600 dark:text-emerald-400",
  worse: "text-red-600 dark:text-red-400",
  flat: "text-slate-500 dark:text-gray-400",
  none: "text-slate-500 dark:text-gray-400",
};

const STATUS_DOT: Record<ScorecardStatusPolarity, string> = {
  better: "bg-emerald-500 dark:bg-emerald-400",
  worse: "bg-red-500 dark:bg-red-400",
  flat: "bg-slate-400 dark:bg-gray-500",
  none: "bg-slate-300 dark:bg-gray-600",
};

const STATUS_SPARKLINE: Record<ScorecardStatusPolarity, string> = {
  better: "text-emerald-500 dark:text-emerald-400",
  worse: "text-red-500 dark:text-red-400",
  flat: "text-slate-400 dark:text-gray-500",
  none: "text-sky-500 dark:text-indigo-400",
};

const STATUS_PROGRESS_BAR: Record<ScorecardStatusPolarity, string> = {
  better: "bg-emerald-500 dark:bg-emerald-400",
  worse: "bg-red-500 dark:bg-red-400",
  flat: "bg-slate-400 dark:bg-gray-500",
  none: "bg-sky-500 dark:bg-indigo-400",
};

export function WidgetScorecard({ render, rows, isLoading, totalCount }: WidgetScorecardProps) {
  const filtered = useMemo(() => applyFilters(rows, render.filters), [rows, render.filters]);

  const value = useMemo(() => {
    if (render.aggregation === "count" && !render.filters?.length && totalCount !== undefined) {
      return totalCount;
    }
    return aggregateNumber(filtered, render.aggregation, render.field);
  }, [filtered, render.aggregation, render.field, render.filters, totalCount]);

  // Sparkline reads `sparklineField` only. The change chip falls back to
  // the primary aggregation `field` so authors get a "vs previous" chip
  // out of the box even when they haven't opted into a sparkline.
  const sparklineSeries = useMemo(
    () => extractScorecardSeries(filtered, render.sparklineField),
    [filtered, render.sparklineField],
  );
  const changeSeriesField = render.sparklineField ?? render.field;
  const changeSeries = useMemo(
    () => extractScorecardSeries(filtered, changeSeriesField),
    [filtered, changeSeriesField],
  );

  // The target expression evaluates against the last filtered row so authors
  // can reference the most recent memory / execution snapshot (e.g. bind to
  // `{{ goal }}` on the latest entry). Falling back to `{}` keeps the CEL
  // env stable when the dataset is empty — the resolver simply returns null.
  const targetContextRow = filtered.length > 0 ? filtered[filtered.length - 1] : null;
  const target = useMemo(
    () => resolveScorecardTarget(render.target, targetContextRow),
    [render.target, targetContextRow],
  );

  const progress = useMemo(() => {
    if (!render.showProgress || target == null || value == null) return null;
    return computeScorecardProgress(value, target, render.better);
  }, [render.showProgress, render.better, target, value]);

  const change = useMemo(
    () => computeScorecardChange(pickChangeAnchors(changeSeries, render.aggregation), render.better),
    [changeSeries, render.aggregation, render.better],
  );

  // Progress used purely for status coloring — computed even when the bar
  // is hidden so target-based coloring still kicks in as a fallback.
  const statusProgress = useMemo(() => {
    if (progress) return progress;
    if (target == null || value == null) return null;
    return computeScorecardProgress(value, target, render.better);
  }, [progress, target, value, render.better]);

  const status = useMemo(() => resolveScorecardStatus(change, statusProgress), [change, statusProgress]);

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center p-4">
        <Loader2 className="size-4 animate-spin text-slate-400 dark:text-gray-500" />
      </div>
    );
  }

  if (value == null) {
    return <WidgetEmptyState icon={Hash} message="No data to display." testId="widget-scorecard-empty" />;
  }

  return (
    <ScorecardDisplay
      render={render}
      value={value}
      change={change}
      progress={progress}
      status={status}
      series={sparklineSeries}
    />
  );
}

interface ScorecardDisplayProps {
  render: WidgetScorecardRender;
  value: number;
  change: TrendResult | null;
  progress: ScorecardProgress | null;
  status: ScorecardStatusPolarity;
  series: number[];
}

function ScorecardDisplay({ render, value, change, progress, status, series }: ScorecardDisplayProps) {
  const format = render.format;
  const formatted = formatValue(value, format ?? "number");
  const hasSparkline = series.length > 1;

  return (
    <div
      className="flex h-full flex-col items-start justify-center gap-2 p-4"
      data-testid="widget-scorecard"
      data-scorecard-status={status}
    >
      {render.label ? (
        <div className="flex items-center gap-2">
          <span
            className={cn("inline-block h-2 w-2 rounded-full", STATUS_DOT[status])}
            aria-hidden
            data-testid="widget-scorecard-status-dot"
          />
          <span className={CONSOLE_WIDGET_LABEL_CLASSES} data-testid="widget-scorecard-label">
            {render.label}
          </span>
        </div>
      ) : null}
      <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
        <ValueBlock
          valueNode={renderValueNode(value, format, formatted)}
          prefix={render.prefix}
          suffix={render.suffix}
        />
        <ChangeChip result={change} render={render} status={status} />
      </div>
      {hasSparkline ? <Sparkline values={series} className={STATUS_SPARKLINE[status]} /> : null}
      {progress ? <ProgressBar progress={progress} status={status} /> : null}
    </div>
  );
}

function renderValueNode(value: number, format: WidgetColumnFormat | undefined, formatted: string) {
  if (isTimestampFormat(format)) {
    const date = coerceWidgetTimestamp(value);
    if (date) {
      return (
        <Timestamp
          date={date}
          display={TIMESTAMP_FORMATS[format]}
          relativeStyle="abbreviated"
          includeAgo={false}
          className={VALUE_CLASS}
        />
      );
    }
  }
  return <span className={VALUE_CLASS}>{formatted}</span>;
}

function ValueBlock({
  valueNode,
  prefix,
  suffix,
}: {
  valueNode: ReactNode;
  prefix: string | undefined;
  suffix: string | undefined;
}) {
  if (!prefix && !suffix) return <>{valueNode}</>;

  if (suffix) {
    return (
      <div className="flex items-baseline gap-0.5">
        {prefix ? <span className={VALUE_CLASS}>{prefix}</span> : null}
        {valueNode}
        <span className={SUFFIX_CLASS}>{suffix}</span>
      </div>
    );
  }

  return (
    <div className="flex items-baseline">
      <span className={VALUE_CLASS}>{prefix}</span>
      {valueNode}
    </div>
  );
}

function ChangeChip({
  result,
  render,
  status,
}: {
  result: TrendResult | null;
  render: WidgetScorecardRender;
  status: ScorecardStatusPolarity;
}) {
  if (!result || result.kind === "no-baseline") return null;

  const showChange = render.showChange ?? "both";
  const label = formatScorecardChangeLabel(result, showChange);
  const tooltip = formatTrendTooltip(result);
  const caption = render.changeCaption;

  const arrow = (() => {
    if (result.kind !== "changed") return null;
    const IconComponent = result.direction === "up" ? ArrowUpRight : ArrowDownRight;
    return <IconComponent className="size-4" aria-hidden />;
  })();

  const chip = (
    <span
      className={cn(
        "inline-flex items-center gap-1 whitespace-nowrap tabular-nums text-sm font-medium",
        STATUS_TEXT[status],
      )}
      data-testid="widget-scorecard-change"
      data-scorecard-change-kind={result.kind}
    >
      {arrow}
      {label ? <span>{label}</span> : null}
    </span>
  );

  const wrappedChip = tooltip ? (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className="inline-flex cursor-default">{chip}</span>
      </TooltipTrigger>
      <TooltipContent side="top">{tooltip}</TooltipContent>
    </Tooltip>
  ) : (
    chip
  );

  if (!caption) return wrappedChip;

  return (
    <span className="inline-flex flex-wrap items-end gap-x-2">
      {wrappedChip}
      <span className="text-xs text-slate-500 dark:text-gray-400" data-testid="widget-scorecard-caption">
        {caption}
      </span>
    </span>
  );
}

function ProgressBar({ progress, status }: { progress: ScorecardProgress; status: ScorecardStatusPolarity }) {
  return (
    <div className="flex w-full flex-col gap-1" data-testid="widget-scorecard-progress">
      <div
        className="h-1.5 w-full overflow-hidden rounded-full bg-slate-200 dark:bg-gray-700"
        aria-hidden
        data-testid="widget-scorecard-progress-track"
      >
        <div
          className={cn("h-full rounded-full transition-[width]", STATUS_PROGRESS_BAR[status])}
          style={{ width: `${progress.barPercent}%` }}
        />
      </div>
      <span className="text-xs text-slate-500 dark:text-gray-400" data-testid="widget-scorecard-progress-label">
        {formatProgressLabel(progress)}
      </span>
    </div>
  );
}

function formatProgressLabel(progress: ScorecardProgress): string {
  const percent = Math.round(progress.percent * 10) / 10;
  const rounded = percent % 1 === 0 ? percent.toFixed(0) : percent.toFixed(1);
  const targetLabel = Number.isInteger(progress.target)
    ? progress.target.toLocaleString()
    : progress.target.toLocaleString(undefined, { maximumFractionDigits: 2 });
  return `${rounded}% of ${targetLabel}`;
}
