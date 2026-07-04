/**
 * Validators for `WidgetChartRender` content. Extracted from `panelTypes.ts`
 * to keep that file readable and within the lint budget.
 *
 * Keep the backend Go validator (`pkg/models/console_yml.go`,
 * `validateChartPanelContent`) in lockstep with the shapes enforced here.
 */

import { asObject, validateSort } from "./panelTypes";
import { WIDGET_CHART_LEGEND_MODES } from "./widget/types";
import type { WidgetChartLegendMode } from "./widget/types";

const ALLOWED_CHART_TYPES = ["bar", "stacked-bar", "line", "area", "donut"];

const CHART_OPTIONAL_STRING_FIELDS = ["seriesField", "xFormat", "yLabel", "yFormat"] as const;

const CHART_SERIES_STRING_FIELDS = ["field", "label", "color", "format", "prefix", "suffix"] as const;

export function validateChartRender(render: Record<string, unknown>): string | null {
  if (render.kind !== "chart") return 'render.kind must be "chart".';
  if (typeof render.type !== "string" || !ALLOWED_CHART_TYPES.includes(render.type)) {
    return `render.type must be one of ${ALLOWED_CHART_TYPES.join(", ")}.`;
  }
  if (typeof render.xField !== "string" || render.xField.trim() === "") {
    return "render.xField must be a non-empty string.";
  }
  for (const key of CHART_OPTIONAL_STRING_FIELDS) {
    const error = validateOptionalString(`render.${key}`, render[key]);
    if (error) return error;
  }
  const seriesError = validateChartSeriesArray(render.series);
  if (seriesError) return seriesError;
  const legendError = validateChartLegend(render.legend);
  if (legendError) return legendError;
  return validateSort(render.sort);
}

function validateChartSeriesArray(series: unknown): string | null {
  if (!Array.isArray(series) || series.length === 0) {
    return "render.series must be a non-empty array.";
  }
  for (let i = 0; i < series.length; i += 1) {
    const error = validateChartSeries(series[i], i);
    if (error) return error;
  }
  return null;
}

function validateChartSeries(raw: unknown, index: number): string | null {
  const series = asObject(raw);
  if (!series) return `render.series[${index}] must be an object.`;
  for (const key of CHART_SERIES_STRING_FIELDS) {
    const error = validateOptionalString(`render.series[${index}].${key}`, series[key]);
    if (error) return error;
  }
  return null;
}

function validateChartLegend(legend: unknown): string | null {
  if (legend === undefined) return null;
  if (typeof legend !== "string" || !WIDGET_CHART_LEGEND_MODES.includes(legend as WidgetChartLegendMode)) {
    return `render.legend must be one of ${WIDGET_CHART_LEGEND_MODES.join(", ")}.`;
  }
  return null;
}

function validateOptionalString(field: string, value: unknown): string | null {
  if (value === undefined || value === null) return null;
  if (typeof value !== "string") return `${field} must be a string.`;
  return null;
}
