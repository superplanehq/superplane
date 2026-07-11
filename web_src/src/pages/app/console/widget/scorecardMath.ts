/**
 * Pure math + resolution helpers for the scorecard panel.
 *
 * The scorecard combines a KPI value with two independent comparisons:
 *  - Change vs the first finite point in the loaded series (`sparklineField`).
 *  - Target (literal or `{{ CEL }}`), used for optional progress and — when
 *    the change is incomputable — a fallback status color.
 *
 * All rendering (color classes, arrows, DOM) stays in `WidgetScorecard`;
 * this module returns discriminated results the widget can render.
 */

import { buildEnv, compileMaybeExpr, evalRowField } from "./celExpr";
import { getValueAtPath } from "./fieldPath";
import type { WidgetScorecardRender, WidgetTrendBetter } from "./types";
import { computeTrend, type TrendResult } from "./widgetTrend";

/** Numeric series extracted from filtered rows using `render.sparklineField`. */
export interface ScorecardSeries {
  values: number[];
  /** First finite value in `values`, `null` when the series is empty. */
  baseline: number | null;
}

/**
 * Extract the series driving both the sparkline and the change baseline.
 * Non-finite values are dropped so the baseline is the first *usable*
 * point — not the first row with a `null`/`""` entry.
 */
export function extractScorecardSeries(rows: unknown[], sparklineField: string | undefined): ScorecardSeries {
  if (!sparklineField) return { values: [], baseline: null };
  const values: number[] = [];
  for (const row of rows) {
    const raw = getValueAtPath(row, sparklineField);
    const n = toFiniteNumber(raw);
    if (n !== null) values.push(n);
  }
  // Baseline is only meaningful when the series has at least two points —
  // a single-point series has no earlier value to compare against, so the
  // change chip should hide entirely rather than render a spurious `flat`.
  return { values, baseline: values.length > 1 ? values[0] : null };
}

/**
 * Resolve `render.target` into a numeric value. Numeric literals (`"50"`,
 * `"100.5"`) are used verbatim; anything else is passed through the shared
 * field resolver so authors can bind to a row field (`goal`, `payload.max`)
 * or a full CEL expression (`{{ base + delta }}`).
 *
 * `contextRow` is the row used as the CEL environment. Scorecards pass the
 * last filtered row (best proxy for "current state") so expressions like
 * `{{ target }}` or `{{ base * 1.1 }}` resolve against the most recent
 * memory / execution entry.
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
 * - `better: "up"` (higher is better): progress = `current / target`, so a
 *   value at or above target reaches 100%.
 * - `better: "down"` (lower is better): meeting or beating the target is
 *   100%; overshoot uses `target / current` so the bar shrinks as the
 *   value drifts further away from the goal.
 *
 * Returns `null` when either input can't be coerced to a finite number, or
 * the target is <= 0 (division-by-zero / meaningless goal).
 */
export interface ScorecardProgress {
  current: number;
  target: number;
  /** Signed percent. May be > 100 when overshooting a `better: up` goal. */
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

  if ((better ?? "up") === "up") {
    const percent = (currentNum / targetNum) * 100;
    return {
      current: currentNum,
      target: targetNum,
      percent,
      barPercent: Math.max(0, Math.min(100, percent)),
      met: currentNum >= targetNum,
    };
  }

  // "down": meeting or beating a low-is-better goal is 100%.
  if (currentNum <= targetNum) {
    return { current: currentNum, target: targetNum, percent: 100, barPercent: 100, met: true };
  }
  const percent = (targetNum / currentNum) * 100;
  return {
    current: currentNum,
    target: targetNum,
    percent,
    barPercent: Math.max(0, Math.min(100, percent)),
    met: false,
  };
}

/**
 * Compute the change chip vs the series baseline (first finite point).
 * Reuses `computeTrend` so the arrow / percent / color semantics stay
 * identical to the table `format: trend` chip.
 *
 * Returns `null` when the baseline is missing (single-point or empty
 * series) — the widget hides the chip and falls back to target-based
 * status coloring.
 */
export function computeScorecardChange(
  current: number | null,
  baseline: number | null,
  better: WidgetTrendBetter | undefined,
): TrendResult | null {
  if (current == null || baseline == null) return null;
  return computeTrend(current, baseline, { better, display: "percent" });
}

/**
 * Priority-ordered status polarity for coloring the value, sparkline, and
 * target line:
 *  1. Change vs baseline (when both present) — `better` / `worse` / `flat`.
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
 * Returns `""` for non-`changed` trend results so the caller only renders
 * the muted `-` icon that `WidgetTableCell` already uses.
 */
export function formatScorecardChangeLabel(result: TrendResult, mode: WidgetScorecardRender["showChange"]): string {
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

function toFiniteNumber(value: unknown): number | null {
  if (typeof value === "number") return Number.isFinite(value) ? value : null;
  if (typeof value === "string") {
    const trimmed = value.trim();
    if (trimmed === "") return null;
    const n = Number(trimmed);
    return Number.isFinite(n) ? n : null;
  }
  return null;
}
