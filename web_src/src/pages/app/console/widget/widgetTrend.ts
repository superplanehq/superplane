/**
 * Pure comparison math for `format: "trend"` table columns.
 *
 * A trend cell compares the current row's numeric value against the "row
 * below" in the filtered+sorted table (the previous entry in whatever
 * ordering the author configured). Rendering is intentionally split out —
 * this module returns a discriminated result the cell renderer can
 * translate into arrow / color / label / tooltip.
 */

import type { WidgetTrendBetter, WidgetTrendDisplay } from "./types";

/** Absolute-value cap applied to percent output, matching UI expectations. */
export const TREND_PERCENT_CAP = 999;

/** Direction the actual value moved between current and previous. */
export type TrendDirection = "up" | "down" | "flat";

/** Whether the direction is "better", "worse", "flat", or has no baseline. */
export type TrendPolarity = "better" | "worse" | "flat" | "none";

/**
 * Result of a trend comparison. `kind` drives what the cell renders:
 *
 * - `pending` → last visible row while `hasMore` is true (`...`)
 * - `no-baseline` → last row with no more data below (`- 0` in gray)
 * - `incomparable` → current or previous isn't a finite number, or
 *   percent-mode was requested against a zero previous (`-`)
 * - `flat` → both sides finite, delta is 0 (`- 0` in gray)
 * - `changed` → the interesting case: arrow + optional signed label
 */
export type TrendResult =
  | { kind: "pending" }
  | { kind: "no-baseline" }
  | { kind: "incomparable" }
  | { kind: "flat"; current: number; previous: number }
  | {
      kind: "changed";
      direction: Exclude<TrendDirection, "flat">;
      polarity: Exclude<TrendPolarity, "flat" | "none">;
      /** Signed delta `current - previous`. */
      delta: number;
      /**
       * Signed percent change `(current - previous) / |previous| * 100`,
       * rounded to one decimal, capped at ±TREND_PERCENT_CAP. `null` when
       * the previous value is 0 (only reachable in `value`/`none` display
       * modes — percent mode returns `incomparable` instead). Always
       * populated for the tooltip when non-null, regardless of `display` mode.
       */
      percent: number | null;
      /** Whether `percent` hit the ±TREND_PERCENT_CAP clamp. */
      percentCapped: boolean;
      current: number;
      previous: number;
    };

export interface ComputeTrendOptions {
  /** Direction that signals "better". Defaults to `up`. */
  better?: WidgetTrendBetter;
  /**
   * When `true`, the row below hasn't been loaded yet (last visible row of
   * a paginated table with more pages available). Renderer shows `...`.
   */
  hasMoreBelow?: boolean;
  /**
   * How the cell will display magnitude. Defaults to `percent`. When the
   * mode is `percent` and the previous value is `0`, percent is undefined
   * so this returns `incomparable` (muted `-`, no arrow) per the PRD.
   * `value` / `none` still render a directional change against a zero baseline.
   */
  display?: WidgetTrendDisplay;
}

/**
 * Compute the trend between two row values. `current` comes from the row
 * being rendered, `previous` from the row below (next entry in display
 * order). Both are the raw output of the field/CEL resolver — this helper
 * coerces numeric strings for convenience.
 */
export function computeTrend(current: unknown, previous: unknown, opts: ComputeTrendOptions = {}): TrendResult {
  const currentNum = toFiniteNumber(current);

  if (previous === undefined) {
    return opts.hasMoreBelow ? { kind: "pending" } : { kind: "no-baseline" };
  }

  const previousNum = toFiniteNumber(previous);
  if (currentNum == null || previousNum == null) return { kind: "incomparable" };

  const delta = currentNum - previousNum;
  if (delta === 0) return { kind: "flat", current: currentNum, previous: previousNum };

  const display = opts.display ?? "percent";
  // Percent change is undefined when previous is 0 — treat as incomparable so
  // the cell renders muted `-` with no directional arrow (PRD).
  if (display === "percent" && previousNum === 0) {
    return { kind: "incomparable" };
  }

  const direction: Exclude<TrendDirection, "flat"> = delta > 0 ? "up" : "down";
  const better: WidgetTrendBetter = opts.better ?? "up";
  const polarity: Exclude<TrendPolarity, "flat" | "none"> = direction === better ? "better" : "worse";

  const { percent, capped } = computePercent(delta, previousNum);

  return {
    kind: "changed",
    direction,
    polarity,
    delta,
    percent,
    percentCapped: capped,
    current: currentNum,
    previous: previousNum,
  };
}

/**
 * Format a trend result for the cell body. The renderer supplies the arrow
 * and color; this returns the accompanying text (may be empty).
 */
export function formatTrendLabel(result: TrendResult, display: WidgetTrendDisplay | undefined): string {
  switch (result.kind) {
    case "pending":
      return "...";
    case "no-baseline":
    case "flat":
      return "0";
    case "incomparable":
      // Icon alone supplies the muted `-`; avoid a second dash in the label.
      return "";
    case "changed": {
      const mode = display ?? "percent";
      if (mode === "none") return "";
      if (mode === "percent") {
        if (result.percent == null) return "-";
        return formatSignedPercent(result.percent, result.percentCapped);
      }
      return formatSignedNumber(result.delta);
    }
  }
}

/**
 * Build the tooltip string. Always shows both percent and absolute delta
 * when both are meaningful; otherwise mirrors the cell label.
 */
export function formatTrendTooltip(result: TrendResult): string | null {
  switch (result.kind) {
    case "pending":
      return "Waiting for more data";
    case "no-baseline":
      return "No previous entry to compare";
    case "incomparable":
      return "Values cannot be compared";
    case "flat":
      return "No change";
    case "changed": {
      const parts: string[] = [];
      if (result.percent != null) parts.push(formatSignedPercent(result.percent, result.percentCapped));
      parts.push(formatSignedNumber(result.delta));
      return parts.join(" · ");
    }
  }
}

function computePercent(delta: number, previous: number): { percent: number | null; capped: boolean } {
  if (previous === 0) return { percent: null, capped: false };
  const raw = (delta / Math.abs(previous)) * 100;
  const rounded = Math.round(raw * 10) / 10;
  if (rounded > TREND_PERCENT_CAP) return { percent: TREND_PERCENT_CAP, capped: true };
  if (rounded < -TREND_PERCENT_CAP) return { percent: -TREND_PERCENT_CAP, capped: true };
  return { percent: rounded, capped: false };
}

function toFiniteNumber(value: unknown): number | null {
  if (typeof value === "number") return Number.isFinite(value) ? value : null;
  if (typeof value === "string" && value.trim() !== "") {
    const n = Number(value);
    return Number.isFinite(n) ? n : null;
  }
  return null;
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
