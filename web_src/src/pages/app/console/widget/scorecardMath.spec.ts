import { describe, expect, it } from "vitest";

import {
  computeScorecardChange,
  computeScorecardProgress,
  extractScorecardSeries,
  formatScorecardChangeLabel,
  pickChangeAnchors,
  resolveScorecardStatus,
  resolveScorecardTarget,
} from "./scorecardMath";
import { computeTrend } from "./widgetTrend";

describe("extractScorecardSeries", () => {
  it("returns an empty array when no field is configured", () => {
    expect(extractScorecardSeries([{ x: 1 }, { x: 2 }], undefined)).toEqual([]);
  });

  it("extracts finite numbers in row order", () => {
    const rows = [{ x: 10 }, { x: 20 }, { x: 30 }];
    expect(extractScorecardSeries(rows, "x")).toEqual([10, 20, 30]);
  });

  it("skips non-finite values so gaps don't poison the series", () => {
    const rows = [{ x: null }, { x: "" }, { x: "abc" }, { x: 5 }, { x: 8 }];
    expect(extractScorecardSeries(rows, "x")).toEqual([5, 8]);
  });

  it("coerces numeric strings", () => {
    const rows = [{ x: "1" }, { x: "2.5" }, { x: "3" }];
    expect(extractScorecardSeries(rows, "x")).toEqual([1, 2.5, 3]);
  });

  it("returns an empty series when nothing is finite", () => {
    expect(extractScorecardSeries([{ x: null }, { x: "abc" }], "x")).toEqual([]);
  });

  it("returns a single-point series intact (anchor selection handles the length gate)", () => {
    expect(extractScorecardSeries([{ x: 42 }], "x")).toEqual([42]);
  });
});

describe("pickChangeAnchors", () => {
  it("returns null when the series has fewer than two points", () => {
    expect(pickChangeAnchors([], "last")).toBeNull();
    expect(pickChangeAnchors([42], "last")).toBeNull();
  });

  it("uses the last two values for aggregation `last`", () => {
    expect(pickChangeAnchors([10, 20, 30], "last")).toEqual({ current: 30, previous: 20 });
  });

  it("uses the first two values for aggregation `first`", () => {
    // Runs / executions surface newest-first, so `first` means "latest" and
    // its natural neighbor is the next-most-recent row.
    expect(pickChangeAnchors([30, 20, 10], "first")).toEqual({ current: 30, previous: 20 });
  });

  it("returns null for combining aggregations that have no natural previous", () => {
    // sum / avg / min / max / count don't point at a single row, so the
    // scorecard has to hide the change chip.
    expect(pickChangeAnchors([10, 20, 30], "sum")).toBeNull();
    expect(pickChangeAnchors([10, 20, 30], "avg")).toBeNull();
    expect(pickChangeAnchors([10, 20, 30], "min")).toBeNull();
    expect(pickChangeAnchors([10, 20, 30], "max")).toBeNull();
    expect(pickChangeAnchors([10, 20, 30], "count")).toBeNull();
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

  it("uses current/target for lower-is-better direction and marks met when under the ceiling", () => {
    // 80 / 100 = 80% of a lower-is-better budget. Bar fills 80%, met=true
    // (we're under the ceiling), coloring stays green via `met`.
    expect(computeScorecardProgress(80, 100, "down")).toEqual({
      current: 80,
      target: 100,
      percent: 80,
      barPercent: 80,
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

  it("clamps the bar but reports the raw percent when overshooting a lower-is-better goal", () => {
    // 200 / 100 = 200% — we've blown past the ceiling. Label still reports
    // the raw 200%, bar clamps at 100%, met flips to false so status colors red.
    const result = computeScorecardProgress(200, 100, "down");
    expect(result).toEqual({ current: 200, target: 100, percent: 200, barPercent: 100, met: false });
  });

  it("regression: 429 vs 500 with better:down reports 85.8% (not 100%)", () => {
    const result = computeScorecardProgress(429, 500, "down");
    expect(result).toMatchObject({ current: 429, target: 500, percent: 85.8, barPercent: 85.8, met: true });
  });

  it("defaults to higher-is-better when no direction is given", () => {
    const result = computeScorecardProgress(50, 100, undefined);
    expect(result).toMatchObject({ percent: 50, met: false });
  });
});

describe("computeScorecardChange", () => {
  it("returns null when no anchors are provided", () => {
    expect(computeScorecardChange(null, "up")).toBeNull();
  });

  it("delegates to computeTrend with percent display", () => {
    expect(computeScorecardChange({ current: 98, previous: 127 }, "down")).toEqual(
      computeTrend(98, 127, { better: "down", display: "percent" }),
    );
  });

  it("wires anchors picked from the series", () => {
    const anchors = pickChangeAnchors([127, 120, 105, 98], "last");
    expect(computeScorecardChange(anchors, "down")).toEqual(
      computeTrend(98, 105, { better: "down", display: "percent" }),
    );
  });
});

describe("resolveScorecardStatus", () => {
  it("prioritizes change polarity when available", () => {
    const change = computeScorecardChange({ current: 11, previous: 10 }, "up");
    expect(resolveScorecardStatus(change, null)).toBe("better");
  });

  it("treats flat change as its own status (not better)", () => {
    const change = computeScorecardChange({ current: 10, previous: 10 }, "up");
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
