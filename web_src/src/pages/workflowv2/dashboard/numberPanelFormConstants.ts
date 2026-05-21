import type { WidgetColumnFormat, WidgetNumberAggregation } from "./widget/types";

export const NUMBER_PANEL_AGGREGATIONS: WidgetNumberAggregation[] = [
  "count",
  "sum",
  "avg",
  "min",
  "max",
  "first",
  "last",
];

export const NUMBER_PANEL_FORMATS: WidgetColumnFormat[] = ["text", "number", "percent", "duration"];
