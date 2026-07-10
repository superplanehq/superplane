import { describe, expect, it } from "vitest";

import { computeTrend, formatTrendLabel, formatTrendTooltip, TREND_PERCENT_CAP } from "./widgetTrend";

describe("computeTrend", () => {
  describe("no baseline", () => {
    it("returns pending when the row below is not yet loaded", () => {
      expect(computeTrend(10, undefined, { hasMoreBelow: true })).toEqual({ kind: "pending" });
    });

    it("returns no-baseline for the last row when no more data is expected", () => {
      expect(computeTrend(10, undefined)).toEqual({ kind: "no-baseline" });
      expect(computeTrend(10, undefined, { hasMoreBelow: false })).toEqual({ kind: "no-baseline" });
    });
  });

  describe("incomparable values", () => {
    it("returns incomparable when either side is not a finite number", () => {
      expect(computeTrend("hello", 5)).toEqual({ kind: "incomparable" });
      expect(computeTrend(5, "world")).toEqual({ kind: "incomparable" });
      expect(computeTrend(null, 5)).toEqual({ kind: "incomparable" });
      expect(computeTrend(5, null)).toEqual({ kind: "incomparable" });
      expect(computeTrend(Number.NaN, 5)).toEqual({ kind: "incomparable" });
    });

    it("coerces numeric strings", () => {
      const result = computeTrend("15", "10");
      expect(result.kind).toBe("changed");
    });
  });

  describe("flat", () => {
    it("returns flat when delta is 0", () => {
      expect(computeTrend(10, 10)).toEqual({ kind: "flat", current: 10, previous: 10 });
    });
  });

  describe("changed — percent math", () => {
    it("computes signed percent with one decimal", () => {
      const result = computeTrend(11, 10);
      expect(result).toMatchObject({ kind: "changed", direction: "up", delta: 1, percent: 10 });
    });

    it("rounds percent to one decimal", () => {
      const result = computeTrend(103, 100);
      expect(result).toMatchObject({ kind: "changed", percent: 3 });

      const oneDecimal = computeTrend(103.75, 100);
      expect(oneDecimal).toMatchObject({ kind: "changed", percent: 3.8 });
    });

    it("returns percent === null when previous is 0", () => {
      const result = computeTrend(5, 0);
      expect(result).toMatchObject({ kind: "changed", percent: null, percentCapped: false });
    });

    it("uses absolute value of previous so a negative baseline preserves direction sign", () => {
      const result = computeTrend(-8, -10);
      expect(result).toMatchObject({ kind: "changed", direction: "up", delta: 2, percent: 20 });
    });

    it("caps percent at +/-999", () => {
      const up = computeTrend(20_000, 10);
      expect(up).toMatchObject({ kind: "changed", percent: TREND_PERCENT_CAP, percentCapped: true });

      const down = computeTrend(-20_000, 10);
      expect(down).toMatchObject({ kind: "changed", percent: -TREND_PERCENT_CAP, percentCapped: true });
    });
  });

  describe("changed — polarity", () => {
    it("marks an increase as better when better=up (default)", () => {
      expect(computeTrend(11, 10)).toMatchObject({ direction: "up", polarity: "better" });
    });

    it("marks a decrease as worse when better=up", () => {
      expect(computeTrend(9, 10)).toMatchObject({ direction: "down", polarity: "worse" });
    });

    it("inverts polarity when better=down", () => {
      expect(computeTrend(11, 10, { better: "down" })).toMatchObject({ direction: "up", polarity: "worse" });
      expect(computeTrend(9, 10, { better: "down" })).toMatchObject({ direction: "down", polarity: "better" });
    });
  });
});

describe("formatTrendLabel", () => {
  it("prints ... for pending", () => {
    expect(formatTrendLabel({ kind: "pending" }, "percent")).toBe("...");
  });

  it("prints 0 for flat / no-baseline regardless of mode", () => {
    expect(formatTrendLabel({ kind: "flat", current: 5, previous: 5 }, "percent")).toBe("0");
    expect(formatTrendLabel({ kind: "flat", current: 5, previous: 5 }, "value")).toBe("0");
    expect(formatTrendLabel({ kind: "flat", current: 5, previous: 5 }, "none")).toBe("0");
    expect(formatTrendLabel({ kind: "no-baseline" }, "percent")).toBe("0");
  });

  it("prints - for incomparable", () => {
    expect(formatTrendLabel({ kind: "incomparable" }, "percent")).toBe("-");
  });

  it("prints - for percent mode when percent is null (previous === 0)", () => {
    const result = computeTrend(5, 0);
    expect(formatTrendLabel(result, "percent")).toBe("-");
  });

  it("prints signed percent for percent mode", () => {
    expect(formatTrendLabel(computeTrend(11, 10), "percent")).toBe("+10%");
    expect(formatTrendLabel(computeTrend(9, 10), "percent")).toBe("-10%");
  });

  it("prints one decimal only when non-integer", () => {
    expect(formatTrendLabel(computeTrend(103.75, 100), "percent")).toBe("+3.8%");
    expect(formatTrendLabel(computeTrend(105, 100), "percent")).toBe("+5%");
  });

  it("prefixes capped percent with > or <", () => {
    expect(formatTrendLabel(computeTrend(20_000, 10), "percent")).toBe(">+999%");
    expect(formatTrendLabel(computeTrend(-20_000, 10), "percent")).toBe("<-999%");
  });

  it("prints signed absolute delta for value mode", () => {
    expect(formatTrendLabel(computeTrend(15, 10), "value")).toBe("+5");
    expect(formatTrendLabel(computeTrend(5, 10), "value")).toBe("-5");
  });

  it("prints empty string for none mode (arrow only) on changed results", () => {
    expect(formatTrendLabel(computeTrend(15, 10), "none")).toBe("");
  });

  it("defaults to percent mode when display is undefined", () => {
    expect(formatTrendLabel(computeTrend(11, 10), undefined)).toBe("+10%");
  });
});

describe("formatTrendTooltip", () => {
  it("returns a helpful message for edge states", () => {
    expect(formatTrendTooltip({ kind: "pending" })).toBe("Waiting for more data");
    expect(formatTrendTooltip({ kind: "no-baseline" })).toBe("No previous entry to compare");
    expect(formatTrendTooltip({ kind: "incomparable" })).toBe("Values cannot be compared");
    expect(formatTrendTooltip({ kind: "flat", current: 5, previous: 5 })).toBe("No change");
  });

  it("shows both percent and absolute delta when comparable", () => {
    expect(formatTrendTooltip(computeTrend(15, 10))).toBe("+50% · +5");
    expect(formatTrendTooltip(computeTrend(5, 10))).toBe("-50% · -5");
  });

  it("omits percent when previous is 0", () => {
    expect(formatTrendTooltip(computeTrend(5, 0))).toBe("+5");
  });
});
