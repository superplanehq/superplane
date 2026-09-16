import { describe, expect, it } from "bun:test";
import { parsePositiveWholeNumber } from "@/lib/positiveWholeNumber";

describe("parsePositiveWholeNumber", () => {
  it("accepts a whole number, with or without surrounding spaces", () => {
    expect(parsePositiveWholeNumber("1")).toBe(1);
    expect(parsePositiveWholeNumber("50")).toBe(50);
    expect(parsePositiveWholeNumber("  7  ")).toBe(7);
  });

  it("rejects a value that is not a whole number", () => {
    expect(parsePositiveWholeNumber("1.5")).toBeNull();
    expect(parsePositiveWholeNumber("12abc")).toBeNull();
    expect(parsePositiveWholeNumber("1e3")).toBeNull();
    expect(parsePositiveWholeNumber("")).toBeNull();
  });

  it("rejects a value below 1", () => {
    expect(parsePositiveWholeNumber("0")).toBeNull();
    expect(parsePositiveWholeNumber("-3")).toBeNull();
  });

  it("rejects a value too large to stay exact", () => {
    expect(parsePositiveWholeNumber("9007199254740993")).toBeNull();
  });
});
