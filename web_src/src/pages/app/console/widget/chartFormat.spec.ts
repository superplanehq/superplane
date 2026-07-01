import { describe, expect, it } from "vitest";

import { formatPercentOfTotal, formatSeriesValue } from "./chartFormat";

describe("formatSeriesValue", () => {
  it("wraps numeric values with prefix and suffix", () => {
    expect(formatSeriesValue(1234.5, { format: "number", prefix: "$" })).toBe("$1,234.5");
    expect(formatSeriesValue(42, { suffix: " MWh" })).toBe("42 MWh");
  });

  it("formats fractions as percentages when format is percent", () => {
    expect(formatSeriesValue(0.42, { format: "percent" })).toBe("42%");
  });

  it("uses default number formatting when no format is provided", () => {
    expect(formatSeriesValue(2500, {})).toBe("2,500");
  });

  it("renders an em-dash for null or undefined values", () => {
    expect(formatSeriesValue(null, { prefix: "$" })).toBe("—");
    expect(formatSeriesValue(undefined, {})).toBe("—");
  });

  it("falls through to string conversion for non-numeric values", () => {
    expect(formatSeriesValue("hello", { format: "text" })).toBe("hello");
  });
});

describe("formatPercentOfTotal", () => {
  it("returns a percent suffix for positive totals", () => {
    expect(formatPercentOfTotal(25, 100)).toBe(" (25%)");
    expect(formatPercentOfTotal(1, 8)).toBe(" (12.5%)");
  });

  it("returns an empty string when total is zero or invalid", () => {
    expect(formatPercentOfTotal(10, 0)).toBe("");
    expect(formatPercentOfTotal(10, NaN)).toBe("");
  });

  it("returns an empty string when the value is not numeric", () => {
    expect(formatPercentOfTotal("not a number", 100)).toBe("");
  });
});
