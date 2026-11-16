import { describe, expect, it } from "bun:test";
import { getBackgroundColorClass, getColorClass, resolveNodeIconColorClass } from "@/lib/colors";

describe("colors", () => {
  it("returns the mapped text color class", () => {
    expect(getColorClass("emerald")).toBe("text-emerald-600 dark:text-emerald-400");
  });

  it.each([
    ["gray", "text-gray-500 dark:text-gray-400", "bg-gray-100"],
    ["zinc", "text-gray-500 dark:text-gray-400", "bg-gray-100"],
    ["white", "text-white dark:text-white", "bg-white"],
    ["black", "text-gray-900 dark:text-gray-300", "bg-black"],
  ])("preserves special color mappings for %s", (color, textClass, backgroundClass) => {
    expect(getColorClass(color)).toBe(textClass);
    expect(getBackgroundColorClass(color)).toBe(backgroundClass);
  });

  it("falls back to gray classes when the color is omitted", () => {
    expect(getColorClass()).toBe("text-gray-500 dark:text-gray-400");
    expect(getBackgroundColorClass()).toBe("bg-gray-100");
  });

  it.each(["", "unknown", "constructor", "toString", "__proto__", "Red", " red "])(
    "falls back to gray classes for unsupported color %j",
    (color) => {
      expect(getColorClass(color)).toBe("text-gray-500 dark:text-gray-400");
      expect(getBackgroundColorClass(color)).toBe("bg-gray-100");
    },
  );

  it("adds dark mode classes for legacy icon colors", () => {
    expect(resolveNodeIconColorClass("text-gray-800")).toBe("text-gray-800 dark:text-gray-400");
    expect(resolveNodeIconColorClass("text-green-700")).toBe("text-green-700 dark:text-green-400");
    expect(resolveNodeIconColorClass("text-green-500")).toBe("text-green-500 dark:text-green-300");
  });

  it("preserves icon colors that already include dark mode classes", () => {
    expect(resolveNodeIconColorClass(getColorClass("black"))).toBe(getColorClass("black"));
  });

  it("returns the mapped background color class", () => {
    expect(getBackgroundColorClass("violet")).toBe("bg-violet-100");
  });
});
