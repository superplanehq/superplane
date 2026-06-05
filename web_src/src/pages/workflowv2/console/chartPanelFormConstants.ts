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

export const CHART_LEGEND_MODE_LABELS: Record<WidgetChartLegendMode, string> = {
  auto: "Auto",
  show: "Always show",
  hide: "Hide",
};
