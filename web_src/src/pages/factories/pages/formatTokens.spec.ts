import { describe, expect, it } from "vitest";

import { formatTokens } from "./formatTokens";

describe("formatTokens", () => {
  it("keeps the raw count below one thousand", () => {
    expect(formatTokens(0)).toBe("0");
    expect(formatTokens(999)).toBe("999");
  });

  it("uses k notation for thousands", () => {
    expect(formatTokens(1000)).toBe("1k");
    expect(formatTokens(2700)).toBe("2.7k");
    expect(formatTokens(18400)).toBe("18.4k");
  });

  it("uses millions notation at or above one million", () => {
    expect(formatTokens(1_000_000)).toBe("1M");
    expect(formatTokens(14_616_000)).toBe("14.6M");
  });
});
