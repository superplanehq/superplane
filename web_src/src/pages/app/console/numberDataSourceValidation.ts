/**
 * Validators for the number panel's data-source shapes:
 *
 * - The shared single-source shape (memory/executions/runs), reused from the
 *   table/chart validators.
 * - The composite memory shape (`sources[]` + `combine`), where each
 *   namespace declares its own aggregation/field and the partials are merged
 *   via a combine operator.
 *
 * Kept in its own file so the schema-heavy `panelTypes.ts` stays within the
 * dashboard package's per-file size budget.
 */

import {
  WIDGET_NUMBER_AGGREGATIONS,
  WIDGET_NUMBER_COMBINE_OPS,
  asObject,
  hasCompositeMemorySourcesKey,
  isAllowedNumberAggregation,
  validateDataSource,
  type WidgetNumberCombine,
} from "./panelTypes";

/**
 * Number panels accept either the shared data-source shapes (memory with a
 * single namespace, executions, runs) or a composite memory variant where
 * each namespace declares its own aggregation/field and the partials are
 * merged with a configured combine operator.
 */
export function validateNumberDataSource(value: unknown): string | null {
  const obj = asObject(value);
  if (!obj) return "dataSource must be an object.";
  if (hasCompositeMemorySourcesKey(obj)) {
    return validateCompositeMemoryDataSource(obj);
  }
  return validateDataSource(value);
}

function validateCompositeMemoryDataSource(obj: Record<string, unknown>): string | null {
  if (!Array.isArray(obj.sources)) return "dataSource.sources must be an array.";
  if (obj.sources.length === 0) return "dataSource.sources must be a non-empty array.";
  for (let i = 0; i < obj.sources.length; i += 1) {
    const sourceError = validateMemoryNumberSource(obj.sources[i], i);
    if (sourceError) return sourceError;
  }
  if (typeof obj.combine !== "string" || !WIDGET_NUMBER_COMBINE_OPS.includes(obj.combine as WidgetNumberCombine)) {
    return `dataSource.combine must be one of ${WIDGET_NUMBER_COMBINE_OPS.join(", ")}.`;
  }
  return null;
}

function validateMemoryNumberSource(raw: unknown, index: number): string | null {
  const source = asObject(raw);
  if (!source) return `dataSource.sources[${index}] must be an object.`;
  if (typeof source.namespace !== "string" || source.namespace.trim() === "") {
    return `dataSource.sources[${index}].namespace must be a non-empty string.`;
  }
  if (!isAllowedNumberAggregation(source.aggregation)) {
    return `dataSource.sources[${index}].aggregation must be one of ${WIDGET_NUMBER_AGGREGATIONS.join(", ")}.`;
  }
  if (source.aggregation !== "count" && (typeof source.field !== "string" || source.field.trim() === "")) {
    return `dataSource.sources[${index}].field is required when aggregation is "${source.aggregation}".`;
  }
  if (source.fieldPath != null && typeof source.fieldPath !== "string") {
    return `dataSource.sources[${index}].fieldPath must be a string.`;
  }
  return null;
}
