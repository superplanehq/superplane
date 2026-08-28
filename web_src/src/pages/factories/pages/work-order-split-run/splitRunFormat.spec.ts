import { describe, expect, it } from "vitest";

import { formatTokenCount } from "./splitRunFormat";

describe("formatTokenCount", () => {
  it("returns undefined for missing or invalid counts", () => {
    expect(formatTokenCount()).toBeUndefined();
    expect(formatTokenCount("")).toBeUndefined();
    expect(formatTokenCount("not-a-number")).toBeUndefined();
  });

  it("keeps the raw count below one thousand", () => {
    expect(formatTokenCount("0")).toBe("0 tokens");
    expect(formatTokenCount("999")).toBe("999 tokens");
  });

  it("uses k notation for thousands", () => {
    expect(formatTokenCount("1000")).toBe("1k tokens");
    expect(formatTokenCount("2700")).toBe("2.7k tokens");
  });

  it("uses millions notation at or above one million", () => {
    expect(formatTokenCount("1000000")).toBe("1M tokens");
    expect(formatTokenCount("14616000")).toBe("14.6M tokens");
  });
});
