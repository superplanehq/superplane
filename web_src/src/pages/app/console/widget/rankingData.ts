import { buildEnv } from "./celExpr";
import { compileFieldResolver } from "./resolveCellValue";
import type { WidgetColumnFormat, WidgetNumberAggregation } from "./types";

/**
 * Render config for the (prototype) ranking / leaderboard panel. Groups the
 * source rows by `groupField`, aggregates a metric per group, sorts the groups
 * into a ranking, and — when `trend` is set — compares each group's current
 * metric against the previous rolling window to surface a direction + delta.
 *
 * Mirrors the shape of the other widget render configs in `types.ts` and
 * reuses their `WidgetNumberAggregation` / `WidgetColumnFormat` types so the
 * same authoring vocabulary carries over. Not registered in `PANEL_TYPES` yet;
 * this powers the Storybook prototype only.
 */
export interface WidgetRankingRender {
  kind: "ranking";
  /** Group key: literal dot-path (e.g. `nodeName`) or `{{ CEL }}` expression. */
  groupField: string;
  /** Header for the group column. Defaults to a generic label in the renderer. */
  groupLabel?: string;
  /** Metric aggregation. `count` counts rows; others reduce `valueField`. */
  aggregation: WidgetNumberAggregation;
  /** Numeric field to aggregate. Required unless `aggregation === "count"`. */
  valueField?: string;
  /** Top-N groups to keep after sorting. Defaults to 10. */
  limit?: number;
  /** Header for the metric column. */
  label?: string;
  /** Display format for the metric value (reused table/number formatting). */
  format?: WidgetColumnFormat;
  /**
   * Trend comparison. When set, rows are split by `timestampField` into the
   * current window `[now-window, now]` and the previous window
   * `[now-2*window, now-window]`, and the metric is compared across them.
   * `window` accepts `s`/`m`/`h`/`d`/`w` suffixes (e.g. `24h`, `7d`, `4w`).
   */
  trend?: { timestampField: string; window: string };
}

/** Direction of a group's metric relative to the previous window. */
export type RankingTrendDirection = "up" | "down" | "flat" | "new";

/** One ranked group row produced by `buildRankingData`. */
export interface RankingRow {
  rank: number;
  group: string;
  value: number;
  /** Previous-window metric. `null` when there is no prior baseline. */
  previousValue: number | null;
  /** Fractional change vs `previousValue` (0.4 = +40%). `null` when undefined. */
  deltaPct: number | null;
  direction: RankingTrendDirection;
}

const DEFAULT_LIMIT = 10;

const WINDOW_UNIT_SECONDS: Record<string, number> = {
  s: 1,
  m: 60,
  h: 60 * 60,
  d: 60 * 60 * 24,
  w: 60 * 60 * 24 * 7,
};

/**
 * Parse a rolling-window string (`24h`, `7d`, `4w`, …) into seconds. Returns
 * `null` for blank or malformed input so callers can fall back to a no-trend
 * ranking instead of throwing on an authoring mistake.
 */
export function parseWindowSeconds(window: string | undefined): number | null {
  if (!window) return null;
  const match = window.trim().match(/^(\d+(?:\.\d+)?)\s*([smhdw])$/i);
  if (!match) return null;
  const amount = Number(match[1]);
  const unit = WINDOW_UNIT_SECONDS[match[2].toLowerCase()];
  if (!Number.isFinite(amount) || amount <= 0 || !unit) return null;
  return amount * unit;
}

/**
 * Build the ranked rows consumed by the ranking renderer. Pure: pass
 * `nowSeconds` to make trend windows deterministic (tests, SSR); it defaults
 * to the wall clock in seconds to match `buildEnv`'s `now`.
 *
 * Steps: resolve group/value fields (dot-path or CEL), optionally split rows
 * into current/previous trend windows, aggregate per group per window, sort by
 * current value desc, assign ranks, slice to `limit`, and compute the trend
 * delta + direction.
 */
export function buildRankingData(
  rows: unknown[],
  render: WidgetRankingRender,
  nowSeconds: number = Math.floor(Date.now() / 1000),
): RankingRow[] {
  const env = buildEnv();
  const groupResolver = compileFieldResolver(render.groupField, env);
  const valueResolver =
    render.aggregation !== "count" && render.valueField ? compileFieldResolver(render.valueField, env) : null;

  const windowSeconds = render.trend ? parseWindowSeconds(render.trend.window) : null;
  const timestampResolver =
    render.trend && windowSeconds ? compileFieldResolver(render.trend.timestampField, env) : null;

  const { current, previous } = splitByWindow(rows, {
    resolveGroup: (row) => String(groupResolver.resolve(row) ?? ""),
    resolveMetric: (row) => (valueResolver ? toFiniteNumber(valueResolver.resolve(row)) : 1),
    resolveTimestamp: timestampResolver ? (row) => toEpochMs(timestampResolver.resolve(row)) : null,
    nowMs: nowSeconds * 1000,
    windowMs: windowSeconds ? windowSeconds * 1000 : 0,
  });

  const groups = Array.from(current.entries()).map(([group, metrics]) => ({
    group,
    value: aggregate(metrics, render.aggregation) ?? 0,
    previousValue: timestampResolver ? aggregateOrNull(previous.get(group), render.aggregation) : null,
  }));

  groups.sort((a, b) => b.value - a.value);

  const limit = render.limit && render.limit > 0 ? render.limit : DEFAULT_LIMIT;
  return groups.slice(0, limit).map((entry, index) => {
    const { deltaPct, direction } = computeTrend(entry.value, entry.previousValue, Boolean(timestampResolver));
    return {
      rank: index + 1,
      group: entry.group,
      value: entry.value,
      previousValue: entry.previousValue,
      deltaPct,
      direction,
    };
  });
}

interface WindowSplitOptions {
  resolveGroup: (row: unknown) => string;
  resolveMetric: (row: unknown) => number | null;
  /** `null` disables trend windowing — every valid row lands in `current`. */
  resolveTimestamp: ((row: unknown) => number | null) | null;
  nowMs: number;
  windowMs: number;
}

/**
 * Bucket each row's metric into the current window (or, when trend is enabled,
 * the current `[now-window, now]` vs previous `[now-2*window, now-window]`
 * windows) keyed by group. Rows with a non-numeric metric — or, when trend is
 * on, an unparseable/out-of-range timestamp — are dropped.
 */
function splitByWindow(
  rows: unknown[],
  options: WindowSplitOptions,
): { current: Map<string, number[]>; previous: Map<string, number[]> } {
  const current = new Map<string, number[]>();
  const previous = new Map<string, number[]>();
  const currentStartMs = options.nowMs - options.windowMs;
  const previousStartMs = options.nowMs - 2 * options.windowMs;

  for (const row of rows) {
    const metric = options.resolveMetric(row);
    if (metric === null) continue;
    const group = options.resolveGroup(row);

    if (!options.resolveTimestamp) {
      pushMetric(current, group, metric);
      continue;
    }

    const timestampMs = options.resolveTimestamp(row);
    if (timestampMs === null) continue;
    if (timestampMs >= currentStartMs && timestampMs <= options.nowMs) {
      pushMetric(current, group, metric);
    } else if (timestampMs >= previousStartMs && timestampMs < currentStartMs) {
      pushMetric(previous, group, metric);
    }
  }

  return { current, previous };
}

function pushMetric(target: Map<string, number[]>, group: string, metric: number): void {
  const existing = target.get(group);
  if (existing) {
    existing.push(metric);
    return;
  }
  target.set(group, [metric]);
}

function aggregateOrNull(metrics: number[] | undefined, aggregation: WidgetNumberAggregation): number | null {
  return metrics ? aggregate(metrics, aggregation) : null;
}

/**
 * Reduce a group's per-row metrics. `count` sums the placeholder `1`s (the
 * caller pushes `1` per row for count), so it can share this reducer with
 * `sum`. Empty input yields `null` so the group carries no baseline.
 */
function aggregate(metrics: number[], aggregation: WidgetNumberAggregation): number | null {
  if (metrics.length === 0) return null;
  switch (aggregation) {
    case "count":
    case "sum":
      return metrics.reduce((a, b) => a + b, 0);
    case "avg":
      return metrics.reduce((a, b) => a + b, 0) / metrics.length;
    case "min":
      return Math.min(...metrics);
    case "max":
      return Math.max(...metrics);
    case "first":
      return metrics[0];
    case "last":
      return metrics[metrics.length - 1];
    default:
      return null;
  }
}

function computeTrend(
  value: number,
  previousValue: number | null,
  trendEnabled: boolean,
): { deltaPct: number | null; direction: RankingTrendDirection } {
  if (!trendEnabled) return { deltaPct: null, direction: "flat" };
  if (previousValue === null || previousValue === 0) {
    return { deltaPct: null, direction: value > 0 ? "new" : "flat" };
  }
  const deltaPct = (value - previousValue) / previousValue;
  if (value > previousValue) return { deltaPct, direction: "up" };
  if (value < previousValue) return { deltaPct, direction: "down" };
  return { deltaPct, direction: "flat" };
}

function toFiniteNumber(value: unknown): number | null {
  if (typeof value === "number") return Number.isFinite(value) ? value : null;
  if (typeof value === "boolean") return value ? 1 : 0;
  if (typeof value === "string" && value.trim() !== "") {
    const n = Number(value);
    return Number.isFinite(n) ? n : null;
  }
  return null;
}

/**
 * Coerce a date-like value (ISO string, `Date`, epoch seconds, or epoch ms)
 * into milliseconds since epoch. Mirrors the `epochMs` builtin in `celExpr.ts`
 * (numbers `>= 1e12` are treated as ms, smaller as seconds). Returns `null`
 * for unparseable input so those rows are dropped from both trend windows.
 */
function toEpochMs(value: unknown): number | null {
  if (value instanceof Date) {
    const t = value.getTime();
    return Number.isFinite(t) ? t : null;
  }
  if (typeof value === "number") {
    if (!Number.isFinite(value)) return null;
    return value >= 1e12 ? value : value * 1000;
  }
  if (typeof value === "string" && value.trim() !== "") {
    const ms = Date.parse(value);
    return Number.isFinite(ms) ? ms : null;
  }
  return null;
}
