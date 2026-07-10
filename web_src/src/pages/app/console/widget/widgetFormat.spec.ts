import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { computeProgress, formatPercentageDisplay, formatValue } from "./widgetFormat";

/**
 * `format: "duration"` always interprets its input as milliseconds. The old
 * magnitude-based heuristic mis-classified small ms values (4527 ms used to
 * print as `1h 15m` because anything <= 10_000 was assumed to be seconds).
 */
describe("formatValue relative", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-03-29T12:00:00.000Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("uses compact relative labels without an ago suffix", () => {
    expect(formatValue("2026-03-29T11:55:00.000Z", "relative")).toBe("5m");
    expect(formatValue("2026-03-29T10:00:00.000Z", "relative")).toBe("2h");
  });
});

describe("formatValue duration", () => {
  it("formats sub-second values in milliseconds", () => {
    expect(formatValue(547, "duration")).toBe("547ms");
    expect(formatValue(1, "duration")).toBe("1ms");
  });

  it("formats the bug-report value (4527 ms) as ~4.5s, not 1h 15m", () => {
    expect(formatValue(4527, "duration")).toBe("4.5s");
  });

  it("formats values just above 1s with one decimal under 10 seconds", () => {
    expect(formatValue(1234, "duration")).toBe("1.2s");
    expect(formatValue(9999, "duration")).toBe("10.0s");
  });

  it("formats values >= 10s as integer seconds", () => {
    expect(formatValue(45_000, "duration")).toBe("45s");
  });

  it("formats minute-scale values as minutes and seconds", () => {
    expect(formatValue(75_000, "duration")).toBe("1m 15s");
    expect(formatValue(60_000 * 30, "duration")).toBe("30m 0s");
  });

  it("formats hour-scale values as hours and minutes", () => {
    expect(formatValue(60_000 * 75, "duration")).toBe("1h 15m");
    expect(formatValue(60_000 * 60 * 5, "duration")).toBe("5h 0m");
  });

  it("formats day-scale values as days and hours", () => {
    expect(formatValue(60_000 * 60 * 25, "duration")).toBe("1d 1h");
    expect(formatValue(60_000 * 60 * 24 * 3, "duration")).toBe("3d 0h");
  });

  it("accepts numeric strings", () => {
    expect(formatValue("4527", "duration")).toBe("4.5s");
    expect(formatValue("547", "duration")).toBe("547ms");
  });

  it("handles zero and negative durations", () => {
    expect(formatValue(0, "duration")).toBe("0ms");
    expect(formatValue(-4527, "duration")).toBe("-4.5s");
  });

  it("falls back to the raw string for non-numeric input", () => {
    expect(formatValue("not-a-number", "duration")).toBe("not-a-number");
  });

  it("returns an empty string for null/undefined", () => {
    expect(formatValue(null, "duration")).toBe("");
    expect(formatValue(undefined, "duration")).toBe("");
  });
});

describe("computeProgress", () => {
  it("computes a straightforward fraction below the target", () => {
    const result = computeProgress(5, 10);
    expect(result).not.toBeNull();
    expect(result!.percent).toBe(50);
    expect(result!.barPercent).toBe(50);
    expect(result!.current).toBe(5);
    expect(result!.target).toBe(10);
  });

  it("clamps the bar at 100% when the current value overshoots the target", () => {
    const result = computeProgress(12, 10);
    expect(result).not.toBeNull();
    expect(result!.percent).toBe(120);
    expect(result!.barPercent).toBe(100);
  });

  it("clamps the bar at 0% when the current value is negative", () => {
    const result = computeProgress(-3, 10);
    expect(result).not.toBeNull();
    expect(result!.percent).toBe(-30);
    expect(result!.barPercent).toBe(0);
  });

  it("accepts numeric strings for both current and target", () => {
    const result = computeProgress("7", "20");
    expect(result).not.toBeNull();
    expect(result!.percent).toBe(35);
  });

  it("returns null when either value is not a finite number", () => {
    expect(computeProgress("nope", 10)).toBeNull();
    expect(computeProgress(5, "")).toBeNull();
    expect(computeProgress(null, 10)).toBeNull();
    expect(computeProgress(5, undefined)).toBeNull();
    expect(computeProgress(Number.NaN, 10)).toBeNull();
  });

  it("returns null when the target is zero or negative (division would be undefined)", () => {
    expect(computeProgress(5, 0)).toBeNull();
    expect(computeProgress(5, -1)).toBeNull();
  });

  it("does not silently promote fractional current values like `percent` formatting", () => {
    // A `current` of 0.5 against a `target` of 10 is 5%, not 50%.
    const result = computeProgress(0.5, 10);
    expect(result!.percent).toBe(5);
    expect(result!.barPercent).toBe(5);
  });
});

describe("formatPercentageDisplay", () => {
  it("rounds whole numbers without a decimal", () => {
    expect(formatPercentageDisplay(50)).toBe("50%");
    expect(formatPercentageDisplay(100)).toBe("100%");
    expect(formatPercentageDisplay(0)).toBe("0%");
  });

  it("shows one decimal place for fractional percentages", () => {
    expect(formatPercentageDisplay(33.3333)).toBe("33.3%");
    expect(formatPercentageDisplay(66.6)).toBe("66.6%");
  });

  it("preserves overshoot values as-is", () => {
    expect(formatPercentageDisplay(120)).toBe("120%");
    expect(formatPercentageDisplay(-15)).toBe("-15%");
  });
});
