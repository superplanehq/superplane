import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { coerceWidgetTimestamp, computeProgress, formatPercentageDisplay, formatValue } from "./widgetFormat";

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

  it("formats future timestamps with an in prefix", () => {
    expect(formatValue("2026-03-29T12:30:00.000Z", "relative")).toBe("in 30m");
    expect(formatValue("2026-03-29T14:00:00.000Z", "relative")).toBe("in 2h");
  });

  it("accepts epoch seconds and milliseconds", () => {
    const now = new Date("2026-03-29T12:00:00.000Z").getTime();
    // Epoch seconds (< 1e11) are treated as seconds; ms values (>= 1e11) pass through.
    expect(formatValue((now - 5 * 60 * 1000) / 1000, "relative")).toBe("5m");
    expect(formatValue(now - 2 * 60 * 60 * 1000, "relative")).toBe("2h");
  });

  it("accepts numeric epoch strings", () => {
    const now = new Date("2026-03-29T12:00:00.000Z").getTime();
    expect(formatValue(String((now - 5 * 60 * 1000) / 1000), "relative")).toBe("5m");
    expect(formatValue(String(now - 2 * 60 * 60 * 1000), "relative")).toBe("2h");
  });
});

describe("formatValue date / datetime", () => {
  const iso = "2026-06-02T10:01:10.561Z";

  it("date renders a calendar day and drops the time-of-day segment", () => {
    const formatted = formatValue(iso, "date");
    expect(formatted).toMatch(/Jun/);
    expect(formatted).toMatch(/2026/);
    expect(formatted).not.toMatch(/\d{2}:\d{2}/);
  });

  it("datetime renders a locale timestamp with time-of-day", () => {
    const formatted = formatValue(iso, "datetime");
    expect(formatted).toMatch(/Jun/);
    expect(formatted).toMatch(/2026/);
    expect(formatted).toMatch(/\d{2}:\d{2}/);
  });

  it("falls back to the raw string for unparseable input", () => {
    expect(formatValue("not-a-date", "date")).toBe("not-a-date");
    expect(formatValue("not-a-date", "datetime")).toBe("not-a-date");
  });
});

describe("coerceWidgetTimestamp", () => {
  it("returns null for null/undefined/empty/invalid inputs", () => {
    expect(coerceWidgetTimestamp(null)).toBeNull();
    expect(coerceWidgetTimestamp(undefined)).toBeNull();
    expect(coerceWidgetTimestamp("")).toBeNull();
    expect(coerceWidgetTimestamp("   nope   ")).toBeNull();
  });

  it("parses ISO strings and Date instances", () => {
    const iso = "2026-06-02T10:01:10.561Z";
    expect(coerceWidgetTimestamp(iso)?.toISOString()).toBe(iso);
    const date = new Date(iso);
    expect(coerceWidgetTimestamp(date)).toBe(date);
  });

  it("treats big numbers as ms and small numbers as seconds since epoch", () => {
    const ms = new Date("2026-06-02T10:01:10.561Z").getTime();
    expect(coerceWidgetTimestamp(ms)?.getTime()).toBe(ms);
    // Seconds must be scaled up.
    const seconds = Math.floor(ms / 1000);
    expect(coerceWidgetTimestamp(seconds)?.getTime()).toBe(seconds * 1000);
  });

  it("parses numeric epoch strings as seconds or milliseconds", () => {
    const ms = new Date("2026-06-02T10:01:10.561Z").getTime();
    const seconds = Math.floor(ms / 1000);
    // JSON / CEL often emit epochs as strings; Date.parse alone rejects them.
    expect(coerceWidgetTimestamp(String(ms))?.getTime()).toBe(ms);
    expect(coerceWidgetTimestamp(String(seconds))?.getTime()).toBe(seconds * 1000);
    expect(coerceWidgetTimestamp(`  ${seconds}  `)?.getTime()).toBe(seconds * 1000);
  });

  it("rejects short digit strings and small numbers that are not plausible epochs", () => {
    for (const value of ["12", "200", "404", "9999", 12, 200, 404, 9999]) {
      expect(coerceWidgetTimestamp(value)).toBeNull();
    }
    expect(formatValue("404", "date")).toBe("404");
    expect(formatValue(404, "datetime")).toBe("404");
    expect(formatValue("12", "relative")).toBe("12");
  });

  it("scales negative epoch milliseconds by magnitude, not sign", () => {
    const ms = -1_500_000_000_000; // ~1922-07-05 in UTC
    expect(coerceWidgetTimestamp(ms)?.getTime()).toBe(ms);
    expect(coerceWidgetTimestamp(String(ms))?.getTime()).toBe(ms);
    const seconds = -1_500_000_000;
    expect(coerceWidgetTimestamp(seconds)?.getTime()).toBe(seconds * 1000);
  });

  it("accepts millisecond epochs in the former 1e11–1e12 gap (1973–2001)", () => {
    // 2000-01-01T00:00:00.000Z as ms sits between the old seconds and ms bands.
    const y2kMs = 946_684_800_000;
    expect(coerceWidgetTimestamp(y2kMs)?.toISOString()).toBe("2000-01-01T00:00:00.000Z");
    expect(coerceWidgetTimestamp(String(y2kMs))?.toISOString()).toBe("2000-01-01T00:00:00.000Z");
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
