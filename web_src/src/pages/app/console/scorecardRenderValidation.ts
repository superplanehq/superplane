/**
 * Validators for `WidgetScorecardRender` content. Extracted from `panelTypes.ts`
 * to keep that file readable and within the lint budget.
 *
 * Keep the backend Go validator (`pkg/models/console_yml.go`) in lockstep with
 * the shapes enforced here.
 */

import {
  isAllowedNumberAggregation,
  validateDataSource,
  validateNumberRenderSymbols,
  WIDGET_NUMBER_AGGREGATIONS,
} from "./panelTypes";
import { asObject, optionalBooleanError, optionalStringError } from "./panelContentValidation";
import { WIDGET_SCORECARD_SHOW_CHANGES, WIDGET_TREND_BETTER } from "./widget/types";
import type { WidgetTrendBetter } from "./widget/types";

export function validateScorecardContent(content: unknown): string | null {
  const obj = asObject(content);
  if (!obj) return "content must be an object.";
  const dsError = validateDataSource(obj.dataSource);
  if (dsError) return dsError;
  const render = asObject(obj.render);
  if (!render) return "render must be an object.";
  if (render.kind !== "scorecard") return 'render.kind must be "scorecard".';
  return validateScorecardRender(render);
}

function validateScorecardRender(render: Record<string, unknown>): string | null {
  if (typeof render.aggregation !== "string" || !isAllowedNumberAggregation(render.aggregation)) {
    return `render.aggregation must be one of ${WIDGET_NUMBER_AGGREGATIONS.join(", ")}.`;
  }
  if (render.aggregation !== "count" && (typeof render.field !== "string" || render.field.trim() === "")) {
    return `render.field is required when aggregation is "${render.aggregation}".`;
  }
  const symbolError = validateNumberRenderSymbols(render);
  if (symbolError) return symbolError;
  const stringError =
    optionalStringError("render.label", render.label) ??
    optionalStringError("render.format", render.format) ??
    optionalStringError("render.sparklineField", render.sparklineField) ??
    optionalStringError("render.target", render.target) ??
    optionalStringError("render.changeCaption", render.changeCaption);
  if (stringError) return stringError;
  const boolError = optionalBooleanError("render.showProgress", render.showProgress);
  if (boolError) return boolError;
  return validateScorecardBetter(render.better) ?? validateScorecardShowChange(render.showChange);
}

function validateScorecardBetter(value: unknown): string | null {
  if (value === undefined || value === null) return null;
  if (typeof value !== "string" || !WIDGET_TREND_BETTER.includes(value as WidgetTrendBetter)) {
    return `render.better must be one of ${WIDGET_TREND_BETTER.join(", ")}.`;
  }
  return null;
}

function validateScorecardShowChange(value: unknown): string | null {
  if (value === undefined || value === null) return null;
  if (
    typeof value !== "string" ||
    !WIDGET_SCORECARD_SHOW_CHANGES.includes(value as (typeof WIDGET_SCORECARD_SHOW_CHANGES)[number])
  ) {
    return `render.showChange must be one of ${WIDGET_SCORECARD_SHOW_CHANGES.join(", ")}.`;
  }
  return null;
}
