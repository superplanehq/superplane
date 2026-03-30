import { describe, expect, it } from "vitest";
import { getBackgroundColorClass, getColorClass } from "@/lib/colors";

describe("colors", () => {
  it("returns the mapped text color class", () => {
    expect(getColorClass("emerald")).toBe("text-emerald-600 dark:text-emerald-400");
  });

  it("falls back to gray text classes for unknown colors", () => {
    expect(getColorClass("unknown")).toBe("text-gray-500 dark:text-gray-400");
  });

  it("returns the mapped background color class", () => {
    expect(getBackgroundColorClass("violet")).toBe("bg-violet-100");
  });

  it("falls back to gray background classes for unknown colors", () => {
    expect(getBackgroundColorClass("unknown")).toBe("bg-gray-100");
  });
});
