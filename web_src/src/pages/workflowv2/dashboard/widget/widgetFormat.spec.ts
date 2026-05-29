import { describe, expect, it } from "vitest";

import { formatValue } from "./widgetFormat";

/**
 * `format: "duration"` always interprets its input as milliseconds. The old
 * magnitude-based heuristic mis-classified small ms values (4527 ms used to
 * print as `1h 15m` because anything <= 10_000 was assumed to be seconds).
 */
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
