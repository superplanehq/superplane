import { getValueAtPath } from "./fieldPath";
import { evaluateShow } from "./showExpression";
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
