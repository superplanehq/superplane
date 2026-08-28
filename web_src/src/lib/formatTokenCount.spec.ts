import { describe, expect, it } from "vitest";

import { formatCompactTokenLabel, formatCompactTokenValue, parseAndFormatTokenCount } from "./formatTokenCount";

describe("formatCompactTokenValue", () => {
  it("returns the plain number below 1,000", () => {
    expect(formatCompactTokenValue(500)).toBe("500");
  });

  it("uses whole-number k notation at 1,000", () => {
    expect(formatCompactTokenValue(1_000)).toBe("1k");
  });

  it("rounds k-range values to the nearest whole thousand", () => {
    expect(formatCompactTokenValue(1_500)).toBe("2k");
  });

  it("switches to M notation for the reported bug case", () => {
    expect(formatCompactTokenValue(14_616_000)).toBe("14.6M");
  });

  it("strips a trailing .0 for exact millions", () => {
    expect(formatCompactTokenValue(3_000_000)).toBe("3M");
  });

  it("switches to B notation for billions", () => {
    expect(formatCompactTokenValue(2_500_000_000)).toBe("2.5B");
  });

  it("strips a trailing .0 for exact billions", () => {
    expect(formatCompactTokenValue(4_000_000_000)).toBe("4B");
  });
});

describe("formatCompactTokenLabel", () => {
  it("appends the tokens suffix", () => {
    expect(formatCompactTokenLabel(14_616_000)).toBe("14.6M tokens");
  });
});

describe("parseAndFormatTokenCount", () => {
  it("returns undefined for missing input", () => {
    expect(parseAndFormatTokenCount(undefined)).toBeUndefined();
    expect(parseAndFormatTokenCount("")).toBeUndefined();
  });

  it("returns undefined for non-numeric input", () => {
    expect(parseAndFormatTokenCount("not-a-number")).toBeUndefined();
  });

  it("parses numeric strings", () => {
    expect(parseAndFormatTokenCount("14616000")).toBe("14.6M tokens");
  });

  it("accepts numeric values directly", () => {
    expect(parseAndFormatTokenCount(500)).toBe("500 tokens");
  });
});
