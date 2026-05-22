import type { CanvasMemoryEntry } from "@/hooks/useCanvasData";

import { getValueAtPath } from "./fieldPath";
import { flattenMemoryEntries } from "./memoryRow";
import { evaluateShow } from "./showExpression";
import type { MemoryNumberSource, WidgetNumberCombine } from "../panelTypes";
import type { WidgetNumberAggregation } from "./types";

/**
 * Apply a list of filter expressions to a row collection. Each expression is
 * evaluated against the row; only rows for which ALL expressions are truthy
 * are kept. Expressions that fail to parse are treated as `false` so that
 * authoring mistakes don't accidentally hide data — they keep the row in the
 * dataset for visibility.
 */
export function applyFilters<T>(rows: T[], filters: string[] | undefined): T[] {
  if (!filters || filters.length === 0) return rows;
  return rows.filter((row) => filters.every((expr) => evaluateShow(expr, row, false)));
}

/**
 * Compute the numeric aggregation for a number widget. Returns the aggregate
 * value (or `null` when no data is present).
 */
export function aggregateNumber(
  rows: unknown[],
  aggregation: WidgetNumberAggregation,
  field: string | undefined,
): number | null {
  if (rows.length === 0) return aggregation === "count" ? 0 : null;
  if (aggregation === "count") return rows.length;

  const numeric: number[] = [];
  for (const row of rows) {
    const raw = field ? getValueAtPath(row, field) : row;
    const value = typeof raw === "number" ? raw : Number(raw);
    if (Number.isFinite(value)) numeric.push(value);
  }
  if (numeric.length === 0) return null;
  switch (aggregation) {
    case "sum":
      return numeric.reduce((a, b) => a + b, 0);
    case "avg":
      return numeric.reduce((a, b) => a + b, 0) / numeric.length;
    case "min":
      return Math.min(...numeric);
    case "max":
      return Math.max(...numeric);
    case "first":
      return numeric[0];
    case "last":
      return numeric[numeric.length - 1];
    default:
      return null;
  }
}

/**
 * Aggregate a single memory namespace contribution for a composite number
 * widget. Flattens the relevant entries (optionally walking `fieldPath`),
 * filters them with the widget's shared `render.filters`, and runs the
 * source's own aggregation/field configuration. Returns `null` when the
 * source has no rows or no numeric values to aggregate (except `count`,
 * which returns `0` for an empty namespace).
 */
export function aggregateNumberPerSource(
  entries: CanvasMemoryEntry[],
  source: MemoryNumberSource,
  filters: string[] | undefined,
): number | null {
  const rows = flattenMemoryEntries(entries, source.namespace, source.fieldPath);
  const filtered = applyFilters(rows, filters);
  return aggregateNumber(filtered, source.aggregation, source.field);
}

/**
 * Merge per-source partial aggregates into a single value. Null partials are
 * skipped (a namespace that produced no numeric value does not poison the
 * combine); if every partial is `null`, the result is `null` so the widget
 * renders its em-dash placeholder.
 *
 * `avg` is the unweighted mean of the available partials — not a row-weighted
 * average across namespaces. Document this in the number panel UI so users
 * can pick `sum` when they want row-level math.
 */
export function combinePartials(partials: Array<number | null>, combine: WidgetNumberCombine): number | null {
  const present = partials.filter((value): value is number => value != null);
  if (present.length === 0) return null;
  switch (combine) {
    case "sum":
      return present.reduce((a, b) => a + b, 0);
    case "min":
      return Math.min(...present);
    case "max":
      return Math.max(...present);
    case "avg":
      return present.reduce((a, b) => a + b, 0) / present.length;
    default:
      return null;
  }
}

/**
 * Build the dataset consumed by chart widgets. Given the parsed rows, the
 * xField, and a list of series field paths, produces an array of `{ x, …series }`
 * objects ready for charting libraries like Recharts.
 */
export function buildChartData(
  rows: unknown[],
  xField: string,
  seriesFields: Array<{ key: string; field?: string }>,
): Array<Record<string, unknown>> {
  return rows.map((row) => {
    const entry: Record<string, unknown> = { x: getValueAtPath(row, xField) ?? "" };
    for (const series of seriesFields) {
      entry[series.key] = series.field ? getValueAtPath(row, series.field) : 1;
    }
    return entry;
  });
}
