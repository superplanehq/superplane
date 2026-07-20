/**
 * Validators for single-KPI and composite number panel content.
 * Extracted from `panelTypes.ts` to stay within the lint budget.
 */

import {
  WIDGET_NUMBER_AGGREGATIONS,
  asObject,
  hasCompositeMemorySourcesKey,
  isAllowedNumberAggregation,
  validateNumberRenderSymbols,
} from "./panelTypes";
import { validateNumberDataSource } from "./numberDataSourceValidation";
import { validateNumberMetrics } from "./numberMetricsValidation";

export function validateNumberContent(content: unknown): string | null {
  const obj = asObject(content);
  if (!obj) return "content must be an object.";
  // Match the backend: presence of `metrics` selects the multi-number path.
  if ("metrics" in obj) {
    return validateNumberMetrics(obj.metrics);
  }
  const dsError = validateNumberDataSource(obj.dataSource);
  if (dsError) return dsError;
  const render = asObject(obj.render);
  if (!render) return "render must be an object.";
  if (render.kind !== "number") return 'render.kind must be "number".';
  const symbolError = validateNumberRenderSymbols(render);
  if (symbolError) return symbolError;
  const dataSource = asObject(obj.dataSource);
  if (dataSource && hasCompositeMemorySourcesKey(dataSource)) {
    return validateCompositeNumberRenderExclusions(render);
  }
  return validateSimpleNumberRender(render);
}

function validateCompositeNumberRenderExclusions(render: Record<string, unknown>): string | null {
  if (render.aggregation !== undefined) {
    return "render.aggregation must not be set when dataSource.sources is used (each source defines its own aggregation).";
  }
  if (render.field !== undefined) {
    return "render.field must not be set when dataSource.sources is used (each source defines its own field).";
  }
  return null;
}

function validateSimpleNumberRender(render: Record<string, unknown>): string | null {
  if (typeof render.aggregation !== "string" || !isAllowedNumberAggregation(render.aggregation)) {
    return `render.aggregation must be one of ${WIDGET_NUMBER_AGGREGATIONS.join(", ")}.`;
  }
  if (render.aggregation !== "count" && (typeof render.field !== "string" || render.field.trim() === "")) {
    return `render.field is required when aggregation is "${render.aggregation}".`;
  }
  return null;
}
