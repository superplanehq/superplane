import { describe, expect, it } from "bun:test";

import {
  centsToUsdInput,
  formatCentsPerMillionUsd,
  formatMatchMode,
  formatMicrosPerSecondUsdPerMinute,
  usdInputToCents,
} from "./priceBookFormat";

describe("formatCentsPerMillionUsd", () => {
  it("formats cents per million tokens as USD", () => {
    expect(formatCentsPerMillionUsd(300)).toBe("$3.00");
    expect(formatCentsPerMillionUsd(8)).toBe("$0.08");
    expect(formatCentsPerMillionUsd(0)).toBe("$0.00");
  });
});

describe("formatMicrosPerSecondUsdPerMinute", () => {
  it("formats stored VM rates as USD per minute", () => {
    expect(formatMicrosPerSecondUsdPerMinute(70)).toBe("$0.0042 / min");
    expect(formatMicrosPerSecondUsdPerMinute(3)).toBe("$0.00018 / min");
    expect(formatMicrosPerSecondUsdPerMinute(0)).toBe("$0.00 / min");
  });
});

describe("formatMatchMode", () => {
  it("labels known match modes", () => {
    expect(formatMatchMode("prefix")).toBe("Prefix");
    expect(formatMatchMode("family")).toBe("Family");
    expect(formatMatchMode("exact")).toBe("Exact");
  });
});

describe("usdInputToCents", () => {
  it("converts dollar input to cents per million", () => {
    expect(usdInputToCents("3.00")).toBe(300);
    expect(usdInputToCents("0.08")).toBe(8);
    expect(usdInputToCents("-1")).toBe(0);
  });
});

describe("centsToUsdInput", () => {
  it("formats cents per million as a dollar input", () => {
    expect(centsToUsdInput(300)).toBe("3.00");
    expect(centsToUsdInput(8)).toBe("0.08");
  });
});
