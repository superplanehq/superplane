import { describe, expect, it } from "vitest";

import { centsToDollarInput, dollarInputToCents, parseDollarInputToCents } from "./hostedCredit";

describe("parseDollarInputToCents", () => {
  it("returns null for empty or invalid input", () => {
    expect(parseDollarInputToCents("")).toBeNull();
    expect(parseDollarInputToCents("   ")).toBeNull();
    expect(parseDollarInputToCents("abc")).toBeNull();
    expect(parseDollarInputToCents("-1")).toBeNull();
  });

  it("parses zero and positive dollar amounts", () => {
    expect(parseDollarInputToCents("0")).toBe(0);
    expect(parseDollarInputToCents("0.00")).toBe(0);
    expect(parseDollarInputToCents("50")).toBe(5000);
    expect(parseDollarInputToCents("12.34")).toBe(1234);
  });
});

describe("dollarInputToCents", () => {
  it("maps empty or invalid input to zero", () => {
    expect(dollarInputToCents("")).toBe(0);
    expect(dollarInputToCents("abc")).toBe(0);
  });
});

describe("centsToDollarInput", () => {
  it("formats cents as a two-decimal dollar string", () => {
    expect(centsToDollarInput(0)).toBe("0.00");
    expect(centsToDollarInput(2500)).toBe("25.00");
  });
});
