import { describe, expect, it } from "bun:test";
import { getBackgroundColorClass, getColorClass, resolveNodeIconColorClass } from "@/lib/colors";

describe("colors", () => {
  it.each([
    ["slate", "text-slate-600 dark:text-slate-400", "bg-slate-100"],
    ["gray", "text-gray-500 dark:text-gray-400", "bg-gray-100"],
    ["zinc", "text-gray-500 dark:text-gray-400", "bg-gray-100"],
    ["neutral", "text-neutral-600 dark:text-neutral-400", "bg-neutral-100"],
    ["stone", "text-stone-600 dark:text-stone-400", "bg-stone-100"],
    ["red", "text-red-600 dark:text-red-400", "bg-red-100"],
    ["orange", "text-orange-600 dark:text-orange-400", "bg-orange-100"],
    ["amber", "text-amber-600 dark:text-amber-400", "bg-amber-100"],
    ["yellow", "text-yellow-600 dark:text-yellow-400", "bg-yellow-100"],
    ["lime", "text-lime-600 dark:text-lime-400", "bg-lime-100"],
    ["green", "text-green-600 dark:text-green-400", "bg-green-100"],
    ["emerald", "text-emerald-600 dark:text-emerald-400", "bg-emerald-100"],
    ["teal", "text-teal-600 dark:text-teal-400", "bg-teal-100"],
    ["cyan", "text-cyan-600 dark:text-cyan-400", "bg-cyan-100"],
    ["sky", "text-sky-600 dark:text-sky-400", "bg-sky-100"],
    ["blue", "text-blue-600 dark:text-blue-400", "bg-blue-100"],
    ["indigo", "text-indigo-600 dark:text-indigo-400", "bg-indigo-100"],
    ["violet", "text-violet-600 dark:text-violet-400", "bg-violet-100"],
    ["purple", "text-purple-600 dark:text-purple-400", "bg-purple-100"],
    ["fuchsia", "text-fuchsia-600 dark:text-fuchsia-400", "bg-fuchsia-100"],
    ["pink", "text-pink-600 dark:text-pink-400", "bg-pink-100"],
    ["rose", "text-rose-600 dark:text-rose-400", "bg-rose-100"],
    ["white", "text-white dark:text-white", "bg-white"],
    ["black", "text-gray-900 dark:text-gray-300", "bg-black"],
  ])("returns the mapped text and background classes for %s", (color, textClass, backgroundClass) => {
    expect(getColorClass(color)).toBe(textClass);
    expect(getBackgroundColorClass(color)).toBe(backgroundClass);
  });

  it.each([undefined, "", "unknown", "toString", "constructor", "hasOwnProperty", "__proto__", "Blue", " blue "])(
    "falls back to gray text and background classes for %s",
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
});
