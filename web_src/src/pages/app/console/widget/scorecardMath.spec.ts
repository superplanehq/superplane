import { describe, expect, it } from "vitest";

import {
  computeScorecardChange,
  computeScorecardProgress,
  extractScorecardSeries,
  formatScorecardChangeLabel,
  resolveScorecardStatus,
  resolveScorecardTarget,
} from "./scorecardMath";
import { computeTrend } from "./widgetTrend";

describe("extractScorecardSeries", () => {
  it("returns an empty series when no field is configured", () => {
    expect(extractScorecardSeries([{ x: 1 }, { x: 2 }], undefined)).toEqual({ values: [], baseline: null });
  });

  it("extracts finite numbers in row order and picks the first as baseline", () => {
    const rows = [{ x: 10 }, { x: 20 }, { x: 30 }];
    expect(extractScorecardSeries(rows, "x")).toEqual({ values: [10, 20, 30], baseline: 10 });
  });

  it("skips non-finite values so the baseline is the first usable point", () => {
    const rows = [{ x: null }, { x: "" }, { x: "abc" }, { x: 5 }, { x: 8 }];
    expect(extractScorecardSeries(rows, "x")).toEqual({ values: [5, 8], baseline: 5 });
  });

  it("coerces numeric strings", () => {
    const rows = [{ x: "1" }, { x: "2.5" }, { x: "3" }];
    expect(extractScorecardSeries(rows, "x")).toEqual({ values: [1, 2.5, 3], baseline: 1 });
  });

  it("returns null baseline when nothing is finite", () => {
    expect(extractScorecardSeries([{ x: null }, { x: "abc" }], "x")).toEqual({ values: [], baseline: null });
  });

  it("returns null baseline for a single-point series", () => {
    // A one-point series has nothing meaningful to compare against, so the
    // widget should hide the change chip entirely rather than render `flat`.
    expect(extractScorecardSeries([{ x: 42 }], "x")).toEqual({ values: [42], baseline: null });
  });
});

describe("resolveScorecardTarget", () => {
  it("returns null for missing / empty targets", () => {
    expect(resolveScorecardTarget(undefined, {})).toBeNull();
    expect(resolveScorecardTarget("", {})).toBeNull();
    expect(resolveScorecardTarget("   ", {})).toBeNull();
  });

  it("uses numeric literals verbatim", () => {
    expect(resolveScorecardTarget("50", {})).toBe(50);
    expect(resolveScorecardTarget("100.5", {})).toBe(100.5);
    expect(resolveScorecardTarget("-3", {})).toBe(-3);
  });

  it("resolves dot paths against the context row", () => {
    expect(resolveScorecardTarget("goal", { goal: 42 })).toBe(42);
    expect(resolveScorecardTarget("payload.max", { payload: { max: 12.5 } })).toBe(12.5);
  });

  it("evaluates CEL expressions against the context row", () => {
    expect(resolveScorecardTarget("{{ base + delta }}", { base: 10, delta: 5 })).toBe(15);
  });

  it("returns null when the resolved value is not a finite number", () => {
    expect(resolveScorecardTarget("goal", { goal: "not a number" })).toBeNull();
    expect(resolveScorecardTarget("goal", { goal: null })).toBeNull();
    expect(resolveScorecardTarget("{{ base + delta }}", {})).toBeNull();
  });

  it("coerces numeric strings resolved from row fields", () => {
    expect(resolveScorecardTarget("goal", { goal: "80" })).toBe(80);
  });
});

describe("computeScorecardProgress", () => {
  it("returns null when either input is not finite", () => {
    expect(computeScorecardProgress(null, 10, "up")).toBeNull();
    expect(computeScorecardProgress(5, null, "up")).toBeNull();
    expect(computeScorecardProgress("nope", 10, "up")).toBeNull();
  });

  it("returns null when the target is <= 0", () => {
    expect(computeScorecardProgress(5, 0, "up")).toBeNull();
    expect(computeScorecardProgress(5, -3, "up")).toBeNull();
  });

  it("computes current/target for the default higher-is-better direction", () => {
    const result = computeScorecardProgress(50, 100, "up");
    expect(result).toEqual({ current: 50, target: 100, percent: 50, barPercent: 50, met: false });
  });

  it("caps the bar at 100 when overshooting a higher-is-better goal", () => {
    const result = computeScorecardProgress(150, 100, "up");
    expect(result).toMatchObject({ percent: 150, barPercent: 100, met: true });
  });

  it("reports 100% when meeting or beating a lower-is-better goal", () => {
    expect(computeScorecardProgress(80, 100, "down")).toEqual({
      current: 80,
      target: 100,
      percent: 100,
      barPercent: 100,
      met: true,
    });
    expect(computeScorecardProgress(100, 100, "down")).toEqual({
      current: 100,
      target: 100,
      percent: 100,
      barPercent: 100,
      met: true,
    });
  });

  it("shrinks the bar as the value drifts above a lower-is-better goal", () => {
    const result = computeScorecardProgress(200, 100, "down");
    expect(result).toEqual({ current: 200, target: 100, percent: 50, barPercent: 50, met: false });
  });

  it("defaults to higher-is-better when no direction is given", () => {
    const result = computeScorecardProgress(50, 100, undefined);
    expect(result).toMatchObject({ percent: 50, met: false });
  });
});

describe("computeScorecardChange", () => {
  it("returns null when either value is missing", () => {
    expect(computeScorecardChange(null, 10, "up")).toBeNull();
    expect(computeScorecardChange(10, null, "down")).toBeNull();
  });

  it("delegates to computeTrend with percent display", () => {
    expect(computeScorecardChange(98, 127, "down")).toEqual(
      computeTrend(98, 127, { better: "down", display: "percent" }),
    );
  });
});

describe("resolveScorecardStatus", () => {
  it("prioritizes change polarity when available", () => {
    const change = computeScorecardChange(11, 10, "up");
    expect(resolveScorecardStatus(change, null)).toBe("better");
  });

  it("treats flat change as its own status (not better)", () => {
    const change = computeScorecardChange(10, 10, "up");
    expect(resolveScorecardStatus(change, null)).toBe("flat");
  });

  it("falls back to target-based status when change is absent", () => {
    const progressMet = computeScorecardProgress(80, 100, "down");
    expect(resolveScorecardStatus(null, progressMet)).toBe("better");

    const progressMiss = computeScorecardProgress(120, 100, "down");
    expect(resolveScorecardStatus(null, progressMiss)).toBe("worse");
  });

  it("returns none when neither signal is available", () => {
    expect(resolveScorecardStatus(null, null)).toBe("none");
  });
});

describe("formatScorecardChangeLabel", () => {
  const changed = computeTrend(98, 127, { better: "down", display: "percent" });

  it("prints both magnitudes by default", () => {
    expect(formatScorecardChangeLabel(changed, undefined)).toBe("-29 (-22.8%)");
  });

  it("supports percent-only", () => {
    expect(formatScorecardChangeLabel(changed, "percent")).toBe("-22.8%");
  });

  it("supports number-only", () => {
    expect(formatScorecardChangeLabel(changed, "number")).toBe("-29");
  });

  it("returns empty for none", () => {
    expect(formatScorecardChangeLabel(changed, "none")).toBe("");
  });

  it("returns empty for non-changed trend results", () => {
    expect(formatScorecardChangeLabel({ kind: "flat", current: 10, previous: 10 }, "both")).toBe("");
    expect(formatScorecardChangeLabel({ kind: "no-baseline" }, "both")).toBe("");
    expect(formatScorecardChangeLabel({ kind: "incomparable" }, "both")).toBe("");
  });

  it("falls back to the number when percent is unavailable in percent/both mode", () => {
    // computeTrend against a 0 baseline in value display returns changed with percent=null.
    const zeroBaseline = computeTrend(5, 0, { display: "value" });
    expect(formatScorecardChangeLabel(zeroBaseline, "percent")).toBe("+5");
    expect(formatScorecardChangeLabel(zeroBaseline, "both")).toBe("+5");
  });
});
