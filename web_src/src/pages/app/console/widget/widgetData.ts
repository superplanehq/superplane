import type { CanvasMemoryEntry } from "@/hooks/useCanvasData";

import { buildEnv } from "./celExpr";
import { getValueAtPath } from "./fieldPath";
import { flattenMemoryEntries } from "./memoryRow";
import { compileFieldResolver } from "./resolveCellValue";
import { evaluateShow } from "./showExpression";
import type { MemoryNumberSource, WidgetNumberCombine } from "../panelTypes";
import type { WidgetNumberAggregation, WidgetSort } from "./types";

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
 * Sort `rows` by a widget-level `sort` spec. Returns the input unchanged when
 * no sort is configured or the field is blank. The field is compiled once and
 * reused across rows, supporting either a literal dot path or a full `{{ cel }}`
 * expression.
 *
 * Comparison rules — picked in order, per-pair:
 *  1. Both finite numeric values → numeric compare.
 *  2. Both parse as valid `Date.parse` timestamps → millisecond compare.
 *  3. Otherwise → `String(a).localeCompare(String(b))`.
 *
 * `null` / `undefined` values always sort to the end regardless of `order` so
 * empty rows don't poison the visible head of the dataset (a common
 * expectation in dashboards displaying time series).
 */
export function applySort<T>(rows: T[], sort: WidgetSort | undefined): T[] {
  if (!sort || !sort.field.trim()) return rows;
  const resolver = compileFieldResolver(sort.field);
  const directionMultiplier = sort.order === "desc" ? -1 : 1;
  return [...rows].sort((a, b) => {
    const valueA = resolver.resolve(a);
    const valueB = resolver.resolve(b);
    const aMissing = valueA == null;
    const bMissing = valueB == null;
    if (aMissing && bMissing) return 0;
    if (aMissing) return 1;
    if (bMissing) return -1;
    return compareSortValues(valueA, valueB) * directionMultiplier;
  });
}

function compareSortValues(a: unknown, b: unknown): number {
  const numericA = toFiniteNumber(a);
  const numericB = toFiniteNumber(b);
  if (numericA !== null && numericB !== null) {
    return numericA === numericB ? 0 : numericA < numericB ? -1 : 1;
  }
  const dateA = toEpochMillis(a);
  const dateB = toEpochMillis(b);
  if (dateA !== null && dateB !== null) {
    return dateA === dateB ? 0 : dateA < dateB ? -1 : 1;
  }
  return String(a).localeCompare(String(b));
}

/**
 * Coerce a cell value to a finite number, or `null` when it isn't numeric.
 *
 * Blank strings, `null`, `undefined`, and non-numeric strings return `null`
 * — they are *not* coerced to `0` (unlike bare `Number(raw)`). Booleans map
 * to `1` / `0`. Shared by aggregations, chart series, and sparkline extraction
 * so headline KPIs and sparklines agree on the same filtered rows.
 */
export function toFiniteNumber(value: unknown): number | null {
  if (typeof value === "number") return Number.isFinite(value) ? value : null;
  if (typeof value === "boolean") return value ? 1 : 0;
  if (typeof value === "string" && value.trim() !== "") {
    const n = Number(value);
    return Number.isFinite(n) ? n : null;
  }
  return null;
}

/**
 * Extract a densely-packed numeric series from `rows` at `field`.
 * Non-finite / blank / null entries are skipped so they never appear as
 * zero points on a sparkline or as change-chip anchors.
 */
export function extractNumericSeries(rows: unknown[], field: string | undefined): number[] {
  if (!field) return [];
  const values: number[] = [];
  for (const row of rows) {
    const n = toFiniteNumber(getValueAtPath(row, field));
    if (n !== null) values.push(n);
  }
  return values;
}

function toEpochMillis(value: unknown): number | null {
  if (value instanceof Date) {
    const t = value.getTime();
    return Number.isFinite(t) ? t : null;
  }
  if (typeof value !== "string" || value.trim() === "") return null;
  const ms = Date.parse(value);
  return Number.isFinite(ms) ? ms : null;
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
    const value = toFiniteNumber(raw);
    if (value !== null) numeric.push(value);
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
 * xField, and a list of series field references, produces an array of
 * `{ x, …series }` objects ready for charting libraries like Recharts.
 *
 * `xField` and each `series.field` accept either a literal dot path (e.g.
 * `createdAt`) or a full `{{ cel }}` expression (e.g.
 * `{{ formatDate(createdAt, "MM/dd") }}`). Expressions are compiled once per
 * call and share a single `ExprEnv` so all rows observe the same `now()` and
 * builtin context, matching how the table renderer evaluates column fields.
 *
 * Rows that resolve to the same `x` value are merged into a single chart
 * point: series with a `field` are **summed** across rows, series without a
 * field **count** the rows in that bucket. Bucket order follows the first
 * appearance of each `x` value, so callers can pre-sort rows to control the
 * category order on the chart.
 *
 * When `seriesField` is provided, the long-format rows are pivoted: each
 * distinct value of `seriesField` becomes its own series key, the value
 * comes from the first entry in `seriesFields` (numeric `field` → sum;
 * undefined `field` → count), and `seriesFields` entries beyond the first
 * are ignored for shaping (their formatting still applies in tooltips via
 * caller-provided maps). Without a `seriesField` the chart keeps its
 * configured series keys as before.
 */
export function buildChartData(
  rows: unknown[],
  xField: string,
  seriesFields: Array<{ key: string; field?: string }>,
  options?: { seriesField?: string },
): Array<Record<string, unknown>> {
  const env = buildEnv();
  const xResolver = compileFieldResolver(xField, env);
  const seriesFieldKey = options?.seriesField?.trim();
  if (seriesFieldKey) {
    return buildPivotedChartData(rows, xResolver, seriesFieldKey, seriesFields[0], env);
  }
  const seriesResolvers = seriesFields.map((series) => ({
    key: series.key,
    hasField: Boolean(series.field),
    resolver: series.field ? compileFieldResolver(series.field, env) : null,
  }));
  const orderedKeys: string[] = [];
  const buckets = new Map<string, Record<string, unknown>>();
  for (const row of rows) {
    const xKey = String(xResolver.resolve(row) ?? "");
    let entry = buckets.get(xKey);
    if (!entry) {
      entry = { x: xKey };
      for (const s of seriesResolvers) entry[s.key] = 0;
      buckets.set(xKey, entry);
      orderedKeys.push(xKey);
    }
    for (const s of seriesResolvers) {
      if (!s.hasField) {
        entry[s.key] = (entry[s.key] as number) + 1;
        continue;
      }
      const raw = s.resolver!.resolve(row);
      const numeric = toFiniteNumber(raw);
      if (numeric !== null) entry[s.key] = (entry[s.key] as number) + numeric;
    }
  }
  return orderedKeys.map((key) => buckets.get(key)!);
}

/** Object key used in pivoted chart rows when `seriesField` resolves empty. */
export const EMPTY_PIVOTED_SERIES_KEY = "(empty)";

function pivotedSeriesDataKey(raw: string): string {
  return raw === "" ? EMPTY_PIVOTED_SERIES_KEY : raw;
}

function buildPivotedChartData(
  rows: unknown[],
  xResolver: ReturnType<typeof compileFieldResolver>,
  seriesField: string,
  valueSeries: { key: string; field?: string } | undefined,
  env: ReturnType<typeof buildEnv>,
): Array<Record<string, unknown>> {
  const seriesResolver = compileFieldResolver(seriesField, env);
  const valueResolver = valueSeries?.field ? compileFieldResolver(valueSeries.field, env) : null;
  const orderedX: string[] = [];
  const orderedSeries: string[] = [];
  const seenSeries = new Set<string>();
  const buckets = new Map<string, Record<string, unknown>>();
  for (const row of rows) {
    const xKey = String(xResolver.resolve(row) ?? "");
    const seriesKey = pivotedSeriesDataKey(String(seriesResolver.resolve(row) ?? ""));
    let entry = buckets.get(xKey);
    if (!entry) {
      entry = { x: xKey };
      buckets.set(xKey, entry);
      orderedX.push(xKey);
    }
    if (!seenSeries.has(seriesKey)) {
      seenSeries.add(seriesKey);
      orderedSeries.push(seriesKey);
    }
    if (entry[seriesKey] === undefined) entry[seriesKey] = 0;
    if (!valueResolver) {
      entry[seriesKey] = (entry[seriesKey] as number) + 1;
      continue;
    }
    const numeric = toFiniteNumber(valueResolver.resolve(row));
    if (numeric !== null) entry[seriesKey] = (entry[seriesKey] as number) + numeric;
  }
  return orderedX.map((xKey) => {
    const entry = buckets.get(xKey)!;
    for (const seriesKey of orderedSeries) {
      if (entry[seriesKey] === undefined) entry[seriesKey] = 0;
    }
    return entry;
  });
}

/**
 * Distinct series keys discovered when pivoting `rows` by `seriesField`. Used
 * by chart renderers to emit one chart layer per pivoted series. Order
 * matches the first occurrence in the input rows.
 */
export function distinctSeriesKeys(rows: unknown[], seriesField: string): string[] {
  const env = buildEnv();
  const resolver = compileFieldResolver(seriesField, env);
  const ordered: string[] = [];
  const seen = new Set<string>();
  for (const row of rows) {
    const key = pivotedSeriesDataKey(String(resolver.resolve(row) ?? ""));
    if (!seen.has(key)) {
      seen.add(key);
      ordered.push(key);
    }
  }
  return ordered;
}
