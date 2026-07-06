import { describe, expect, it } from "vitest";
import { getBackgroundColorClass, getColorClass, resolveNodeIconColorClass } from "@/lib/colors";

describe("colors", () => {
  it("returns the mapped text color class", () => {
    expect(getColorClass("emerald")).toBe("text-emerald-600 dark:text-emerald-400");
  });

  it("maps black to a light icon color in dark mode", () => {
    expect(getColorClass("black")).toBe("text-gray-900 dark:text-gray-300");
  });

  it("falls back to gray text classes for unknown colors", () => {
    expect(getColorClass("unknown")).toBe("text-gray-500 dark:text-gray-400");
  });

  it("adds dark mode classes for legacy icon colors", () => {
    expect(resolveNodeIconColorClass("text-gray-800")).toBe("text-gray-800 dark:text-gray-400");
    expect(resolveNodeIconColorClass("text-green-700")).toBe("text-green-700 dark:text-green-400");
  });

  it("preserves icon colors that already include dark mode classes", () => {
    expect(resolveNodeIconColorClass(getColorClass("black"))).toBe(getColorClass("black"));
  });

  it("returns the mapped background color class", () => {
    expect(getBackgroundColorClass("violet")).toBe("bg-violet-100");
  });

  it("falls back to gray background classes for unknown colors", () => {
    expect(getBackgroundColorClass("unknown")).toBe("bg-gray-100");
  });
});
