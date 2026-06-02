/**
 * Validators for the multi-number panel mode (`content.metrics[]`).
 *
 * Each metric uses a simple (non-composite) data source plus its own number
 * render so the panel can display several independently-configured KPIs in a
 * wrapping row. Kept in its own file to keep `panelTypes.ts` focused and
 * within the dashboard package's per-file size budget.
 */

import {
  WIDGET_NUMBER_AGGREGATIONS,
  asObject,
  hasCompositeMemorySourcesKey,
  isAllowedNumberAggregation,
  validateDataSource,
  validateNumberRenderSymbols,
} from "./panelTypes";

export function validateNumberMetrics(metrics: unknown): string | null {
  if (!Array.isArray(metrics)) return "metrics must be an array.";
  if (metrics.length === 0) return "metrics must be a non-empty array.";
  for (let i = 0; i < metrics.length; i += 1) {
    const error = validateNumberMetric(metrics[i], i);
    if (error) return error;
  }
  return null;
}

function validateNumberMetric(raw: unknown, index: number): string | null {
  const metric = asObject(raw);
  if (!metric) return `metrics[${index}] must be an object.`;
  // Multi-number metrics use the simple (non-composite) data-source shape.
  const dsObj = asObject(metric.dataSource);
  if (dsObj && hasCompositeMemorySourcesKey(dsObj)) {
    return `metrics[${index}].dataSource must be a single-source memory/executions/runs source.`;
  }
  const dsError = validateDataSource(metric.dataSource);
  if (dsError) return `metrics[${index}].${dsError}`;
  const render = asObject(metric.render);
  if (!render) return `metrics[${index}].render must be an object.`;
  if (render.kind !== "number") return `metrics[${index}].render.kind must be "number".`;
  const symbolError = validateNumberRenderSymbols(render);
  if (symbolError) return `metrics[${index}].${symbolError}`;
  return validateMetricAggregation(render, index);
}

function validateMetricAggregation(render: Record<string, unknown>, index: number): string | null {
  if (!isAllowedNumberAggregation(render.aggregation)) {
    return `metrics[${index}].render.aggregation must be one of ${WIDGET_NUMBER_AGGREGATIONS.join(", ")}.`;
  }
  if (render.aggregation !== "count" && (typeof render.field !== "string" || render.field.trim() === "")) {
    return `metrics[${index}].render.field is required when aggregation is "${render.aggregation}".`;
  }
  return null;
}
