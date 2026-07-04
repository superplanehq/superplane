import type { WidgetChartKind, WidgetChartLegendMode, WidgetColumnFormat } from "./widget/types";

export const CHART_KINDS: WidgetChartKind[] = ["bar", "stacked-bar", "line", "area", "donut"];

export const CHART_KIND_LABELS: Record<WidgetChartKind, string> = {
  bar: "Bar",
  "stacked-bar": "Stacked bar",
  line: "Line",
  area: "Area",
  donut: "Donut",
};

export const CHART_SERIES_FORMATS: WidgetColumnFormat[] = ["text", "number", "percent", "duration"];

/**
 * Formats appropriate for X-axis ticks. Includes time-oriented formats so
 * authors can point `xField` directly at a timestamp without wrapping it in
 * a CEL expression.
 */
export const CHART_X_AXIS_FORMATS: WidgetColumnFormat[] = [
  "text",
  "number",
  "percent",
  "date",
  "datetime",
  "relative",
  "duration",
];

/**
 * Formats appropriate for Y-axis ticks. Numeric-oriented only — date/badge
 * formats don't apply to a continuous numeric axis.
 */
export const CHART_Y_AXIS_FORMATS: WidgetColumnFormat[] = ["number", "percent", "duration"];

export const CHART_LEGEND_MODE_LABELS: Record<WidgetChartLegendMode, string> = {
  auto: "Auto",
  show: "Always show",
  hide: "Hide",
};
