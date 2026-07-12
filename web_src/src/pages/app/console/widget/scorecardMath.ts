/**
 * Pure math + resolution helpers for the scorecard panel.
 *
 * The scorecard combines a KPI value with two independent comparisons:
 *  - Change vs the *immediately previous* value in the series
 *    (`sparklineField` when set, otherwise the primary aggregation `field`).
 *  - Target (literal or `{{ CEL }}`), used for optional progress and — when
 *    the change is incomputable — a fallback status color.
 *
 * All rendering (color classes, arrows, DOM) stays in `WidgetScorecard`;
 * this module returns discriminated results the widget can render.
 */

import { buildEnv, compileMaybeExpr, evalRowField } from "./celExpr";
import { getValueAtPath } from "./fieldPath";
import type { WidgetNumberAggregation, WidgetScorecardRender, WidgetTrendBetter } from "./types";
import { extractNumericSeries, toFiniteNumber } from "./widgetData";
import { computeTrend, type TrendResult } from "./widgetTrend";

/**
 * Extract the numeric series driving the sparkline and change chip.
 *
 * Non-finite entries are dropped so a `null` / `""` / non-numeric row
 * doesn't poison the anchor selection — callers get a densely-packed
 * `number[]` where index 0 is the first usable point in row order.
 */
export function extractScorecardSeries(rows: unknown[], seriesField: string | undefined): number[] {
  return extractNumericSeries(rows, seriesField);
}

/** Pair of values consumed by {@link computeScorecardChange}. */
export interface ScorecardChangeAnchors {
  current: number;
  previous: number;
}

/**
 * Pick the (current, previous) anchor pair used by the change chip.
 *
 * Only aggregations that select a single record from the series have a
 * natural "immediate previous" neighbor:
 *  - `last`  → current = `series[N-1]`, previous = `series[N-2]`.
 *  - `first` → current = `series[0]`,   previous = `series[1]`.
 *
 * Combining aggregations (`sum`, `avg`, `min`, `max`, `count`) do not
 * point at a single row, so there is no coherent "previous" — the widget
 * hides the chip in those cases. Returns `null` when the anchor cannot
 * be resolved (unsupported aggregation, or series with fewer than two
 * finite points).
 */
export function pickChangeAnchors(
  series: number[],
  aggregation: WidgetNumberAggregation,
): ScorecardChangeAnchors | null {
  if (series.length < 2) return null;
  if (aggregation === "last") {
    return { current: series[series.length - 1], previous: series[series.length - 2] };
  }
  if (aggregation === "first") {
    return { current: series[0], previous: series[1] };
  }
  return null;
}

/**
 * Resolve `render.target` into a numeric value. Numeric literals (`"50"`,
 * `"100.5"`) are used verbatim; anything else is passed through the shared
 * field resolver so authors can bind to a row field (`goal`, `payload.max`)
 * or a full CEL expression (`{{ base + delta }}`).
 *
 * `contextRow` is the row used as the CEL environment. Scorecards pass the
 * newest filtered row (index 0 — all widget data sources are newest-first)
 * so expressions like `{{ target }}` or `{{ base * 1.1 }}` resolve against
 * the most recent memory / execution entry.
 *
 * Returns `null` when the target is empty, unparseable, or resolves to
 * something that isn't a finite number.
 */
export function resolveScorecardTarget(target: string | undefined, contextRow: unknown): number | null {
  if (target == null) return null;
  const trimmed = target.trim();
  if (trimmed === "") return null;
  const literal = Number(trimmed);
  if (Number.isFinite(literal)) return literal;
  const record =
    contextRow && typeof contextRow === "object" && !Array.isArray(contextRow)
      ? (contextRow as Record<string, unknown>)
      : {};
  const maybe = compileMaybeExpr(trimmed);
  const resolved = evalRowField(maybe, record, buildEnv(), getValueAtPath);
  return toFiniteNumber(resolved);
}

/**
 * Direction-aware progress values for the scorecard's optional progress bar.
 *
 * In both directions the bar visualizes `current / target` — the fraction
 * of the target the current value covers — clamped to `[0, 100]` for the
 * visible width. This keeps the label honest ("429 covers 85.8% of a
 * ceiling of 500", not a misleading "100% of 500"). The direction only
 * affects `met` and the color the widget picks:
 *
 * - `better: "up"`   (higher is better): `met = current >= target`.
 * - `better: "down"` (lower is better):  `met = current <= target`.
 *
 * Returns `null` when either input can't be coerced to a finite number, or
 * the target is <= 0 (division-by-zero / meaningless goal).
 */
export interface ScorecardProgress {
  current: number;
  target: number;
  /** Raw signed percent. May be > 100 when the value exceeds the target. */
  percent: number;
  /** Clamped `[0, 100]` — drives the bar width. */
  barPercent: number;
  /** Whether the current value is on the "better" side of the target. */
  met: boolean;
}

export function computeScorecardProgress(
  current: unknown,
  target: unknown,
  better: WidgetTrendBetter | undefined,
): ScorecardProgress | null {
  const currentNum = toFiniteNumber(current);
  const targetNum = toFiniteNumber(target);
  if (currentNum === null || targetNum === null) return null;
  if (targetNum <= 0) return null;

  const percent = (currentNum / targetNum) * 100;
  const barPercent = Math.max(0, Math.min(100, percent));
  const met = (better ?? "up") === "down" ? currentNum <= targetNum : currentNum >= targetNum;
  return { current: currentNum, target: targetNum, percent, barPercent, met };
}

/**
 * Compute the change chip given the resolved anchor pair. Reuses
 * `computeTrend` so the arrow / percent / color semantics stay identical
 * to the table `format: trend` chip.
 *
 * Returns `null` when the anchor pair is missing (single-point series,
 * empty series, or an aggregation with no natural "previous" — see
 * {@link pickChangeAnchors}). The widget hides the chip in those cases
 * and falls back to target-based status coloring.
 */
export function computeScorecardChange(
  anchors: ScorecardChangeAnchors | null,
  better: WidgetTrendBetter | undefined,
  showChange?: WidgetScorecardRender["showChange"],
): TrendResult | null {
  if (!anchors) return null;
  // Map scorecard `showChange` onto trend `display`. Prefer `value` whenever
  // a numeric delta is useful (`number` / `both` / default) so a zero
  // previous still yields a signed change instead of `incomparable`.
  // Percent-only keeps percent math (and its zero-baseline incomparable).
  const display = showChangeToTrendDisplay(showChange);
  return computeTrend(anchors.current, anchors.previous, { better, display });
}

function showChangeToTrendDisplay(showChange: WidgetScorecardRender["showChange"]): "percent" | "value" | "none" {
  if (showChange === "percent") return "percent";
  if (showChange === "none") return "none";
  return "value";
}

/**
 * Priority-ordered status polarity for coloring the value, sparkline, and
 * target line:
 *  1. Change vs previous (when present) — `better` / `worse` / `flat`.
 *  2. Target comparison (when target resolvable) — `met` → better,
 *     otherwise worse.
 *  3. Neutral.
 *
 * `flat` is separate so the widget can render it in muted slate rather
 * than green (a flat trend is not a win).
 */
export type ScorecardStatusPolarity = "better" | "worse" | "flat" | "none";

export function resolveScorecardStatus(
  change: TrendResult | null,
  progress: ScorecardProgress | null,
): ScorecardStatusPolarity {
  if (change) {
    if (change.kind === "changed") return change.polarity;
    if (change.kind === "flat") return "flat";
  }
  if (progress) return progress.met ? "better" : "worse";
  return "none";
}

/**
 * Format the change chip's magnitude label for the scorecard.
 * Mirrors screenshot conventions:
 *  - `percent` → `-22.8%`
 *  - `number`  → `-29`
 *  - `both`    → `-29 (-22.8%)`
 *  - `none`    → empty string (arrow only)
 *
 * Falls back to just the delta when percent is unavailable (previous = 0).
 * Returns `"0"` for a flat trend (mirrors `formatTrendLabel` / table chips)
 * and `""` for other non-`changed` results so the caller can render icon-only.
 */
export function formatScorecardChangeLabel(result: TrendResult, mode: WidgetScorecardRender["showChange"]): string {
  if (result.kind === "flat") return "0";
  if (result.kind !== "changed") return "";
  const displayMode = mode ?? "both";
  if (displayMode === "none") return "";
  const number = formatSignedNumber(result.delta);
  if (displayMode === "number") return number;
  const percent = result.percent != null ? formatSignedPercent(result.percent, result.percentCapped) : null;
  if (displayMode === "percent") return percent ?? number;
  return percent ? `${number} (${percent})` : number;
}

function formatSignedPercent(percent: number, capped: boolean): string {
  const prefix = capped ? (percent > 0 ? ">" : "<") : "";
  const sign = percent > 0 ? "+" : percent < 0 ? "-" : "";
  const magnitude = Math.abs(percent);
  const withDecimals = magnitude % 1 === 0 ? magnitude.toFixed(0) : magnitude.toFixed(1);
  return `${prefix}${sign}${withDecimals}%`;
}

function formatSignedNumber(value: number): string {
  const sign = value > 0 ? "+" : value < 0 ? "-" : "";
  const magnitude = Math.abs(value);
  const rounded = Math.round(magnitude * 100) / 100;
  const formatted =
    rounded % 1 === 0 ? rounded.toLocaleString() : rounded.toLocaleString(undefined, { maximumFractionDigits: 2 });
  return `${sign}${formatted}`;
}
