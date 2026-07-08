import type { ComponentProps } from "react";
import * as yaml from "js-yaml";

import type { WidgetColumnFormat, WidgetDataSource, WidgetNumberAggregation } from "./widget/types";
import type { ScorecardGoalDirection, ScorecardThreshold, WidgetScorecard } from "./widget/WidgetScorecard";

/**
 * Editor-facing content model for the prototype scorecard panel. It mirrors how
 * real console panels are structured — a `dataSource` (memory / executions /
 * runs) plus render config — so the fields map onto concepts the console
 * already has. A data-bound panel would run `useWidgetData(dataSource)`, reduce
 * the rows via `aggregation`/`field` for the hero value, and plot `seriesField`
 * across the rows for the sparkline and trend.
 */
export type ScorecardStatusMode = "target" | "thresholds";

export interface ScorecardPanelContent {
  title?: string;
  /** Where the rows come from (same shape as table/chart/number panels). */
  dataSource: WidgetDataSource;
  /** How the rows are reduced to the single hero value. */
  aggregation: WidgetNumberAggregation;
  /** Field to aggregate. Required unless `aggregation` is `count`. */
  field?: string;
  format?: WidgetColumnFormat;
  label?: string;
  prefix?: string;
  suffix?: string;
  goalDirection: ScorecardGoalDirection;
  statusMode: ScorecardStatusMode;
  target?: number;
  thresholds?: ScorecardThreshold[];
  /** Numeric field plotted across the rows; powers the sparkline and the trend. */
  seriesField?: string;
  showSparkline?: boolean;
  /** Change vs the start of the range (first vs latest point of the series). */
  showTrend?: boolean;
  trendLabel?: string;
  showProgress?: boolean;
}

/** Calculations offered in the editor (matches the Number panel vocabulary). */
export const SCORECARD_AGGREGATIONS: WidgetNumberAggregation[] = ["last", "first", "count", "sum", "avg", "min", "max"];

/** Formats offered in the editor's value-format Select. */
export const SCORECARD_FORMATS: WidgetColumnFormat[] = ["number", "percent", "duration"];

export const DEFAULT_TREND_LABEL = "vs start of range";

/** A valid, immediately-rendering default so the editor is never empty. */
export const DEFAULT_SCORECARD_CONTENT: ScorecardPanelContent = {
  title: "",
  dataSource: { kind: "memory", namespace: "deploy_metrics" },
  aggregation: "last",
  field: "success_rate",
  format: "percent",
  label: "Success rate",
  prefix: "",
  suffix: "",
  goalDirection: "higher",
  statusMode: "target",
  target: 95,
  thresholds: [
    { at: 85, status: "warn" },
    { at: 95, status: "good" },
  ],
  seriesField: "success_rate",
  showSparkline: true,
  showTrend: true,
  trendLabel: DEFAULT_TREND_LABEL,
  showProgress: false,
};

type ScorecardRenderProps = ComponentProps<typeof WidgetScorecard>;

/** Reduce a numeric series the way `useWidgetData` rows would be aggregated. */
function reduceSeries(series: number[], aggregation: WidgetNumberAggregation): number | null {
  if (series.length === 0) return null;
  switch (aggregation) {
    case "count":
      return series.length;
    case "sum":
      return series.reduce((total, n) => total + n, 0);
    case "avg":
      return series.reduce((total, n) => total + n, 0) / series.length;
    case "min":
      return Math.min(...series);
    case "max":
      return Math.max(...series);
    case "first":
      return series[0];
    case "last":
      return series[series.length - 1];
  }
}

/**
 * Map the editor content to `WidgetScorecard` render props. In the prototype
 * `sampleSeries` stands in for `rows.map(seriesField)` a data-bound panel would
 * produce; the hero value is reduced from it so the preview reflects the chosen
 * calculation, and the trend compares the series start to its latest point.
 */
export function scorecardPropsFromContent(
  content: ScorecardPanelContent,
  sampleSeries: number[],
): ScorecardRenderProps {
  const value = reduceSeries(sampleSeries, content.aggregation);
  const seriesEnabled = Boolean(content.seriesField && sampleSeries.length > 0);
  const sparkline = content.showSparkline && seriesEnabled ? sampleSeries : undefined;
  const comparison =
    content.showTrend && seriesEnabled && sampleSeries.length > 1
      ? { value: sampleSeries[0], label: content.trendLabel || DEFAULT_TREND_LABEL }
      : undefined;
  return {
    value,
    label: content.label || undefined,
    format: content.format ?? "number",
    prefix: content.prefix || undefined,
    suffix: content.suffix || undefined,
    goalDirection: content.goalDirection,
    target: content.statusMode === "target" ? content.target : undefined,
    thresholds: content.statusMode === "thresholds" ? content.thresholds : undefined,
    comparison,
    sparkline,
    showProgress: content.showProgress,
  };
}

/** Whether a goal line exists (drives the "show progress" affordance). */
export function scorecardHasGoalLine(content: ScorecardPanelContent): boolean {
  if (content.statusMode === "target") return content.target != null && Number.isFinite(content.target);
  return (content.thresholds?.length ?? 0) > 0;
}

/**
 * Return a human-readable problem with the content, or `null` when valid.
 * Powers both the inline field hints and the footer summary strip.
 */
export function validateScorecardContent(content: ScorecardPanelContent): string | null {
  const dataSource = content.dataSource;
  if (dataSource.kind === "memory" && !dataSource.namespace.trim()) {
    return "Choose a memory namespace for the data source.";
  }
  if (content.aggregation !== "count" && !content.field?.trim()) {
    return "Choose a field to aggregate, or use the Count calculation.";
  }
  const statusProblem = validateStatus(content);
  if (statusProblem) return statusProblem;
  if ((content.showTrend || content.showSparkline) && !content.seriesField?.trim()) {
    return "Add a series field to power the sparkline and trend.";
  }
  return null;
}

function validateStatus(content: ScorecardPanelContent): string | null {
  if (content.statusMode === "target") {
    if (content.target == null || !Number.isFinite(content.target)) {
      return "Set a target value, or switch to threshold bands.";
    }
    return null;
  }
  const bands = content.thresholds ?? [];
  if (bands.length === 0) {
    return "Add at least one threshold band, or switch to a single target.";
  }
  if (bands.some((band) => !Number.isFinite(band.at))) {
    return "Every threshold band needs a numeric value.";
  }
  return null;
}

/**
 * Serialize the content to YAML for the editor's YAML tab, in the same
 * `dataSource` + `render` shape the real panels persist.
 */
export function scorecardContentToYaml(content: ScorecardPanelContent): string {
  const render: Record<string, unknown> = {
    aggregation: content.aggregation,
    field: content.aggregation === "count" ? undefined : content.field || undefined,
    format: content.format,
    label: content.label || undefined,
    prefix: content.prefix || undefined,
    suffix: content.suffix || undefined,
    goalDirection: content.goalDirection,
  };
  if (content.statusMode === "target") {
    render.target = content.target;
  } else {
    render.thresholds = content.thresholds;
  }
  if (content.seriesField) render.seriesField = content.seriesField;
  if (content.showSparkline) render.sparkline = true;
  if (content.showTrend) render.trend = { compare: "range-start", label: content.trendLabel || undefined };
  if (content.showProgress) render.showProgress = true;

  return yaml.dump(
    { type: "scorecard", dataSource: content.dataSource, render },
    {
      noRefs: true,
      lineWidth: 100,
      sortKeys: false,
    },
  );
}
