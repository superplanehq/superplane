import { afterEach, describe, expect, it, vi } from "vitest";
import { Building2, Puzzle, StickyNote, X } from "lucide-react";
import { calcRelativeTimeFromDiff, cn, flattenObject, formatTimestamp, isUrl, resolveIcon } from "@/lib/utils";

describe("utils", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("merges class names with tailwind precedence", () => {
    expect(cn("px-2", "px-4", { hidden: false, block: true })).toBe("px-4 block");
  });

  it("resolves known icon aliases and falls back to Puzzle", () => {
    expect(resolveIcon("x-mark")).toBe(X);
    expect(resolveIcon("business")).toBe(Building2);
    expect(resolveIcon("sticky-note")).toBe(StickyNote);
    expect(resolveIcon("does-not-exist")).toBe(Puzzle);
  });

  it("formats relative time from millisecond diffs", () => {
    expect(calcRelativeTimeFromDiff(15_000)).toBe("15s");
    expect(calcRelativeTimeFromDiff(3_600_000)).toBe("1h");
    expect(calcRelativeTimeFromDiff(172_800_000)).toBe("2d");
  });

  it("formats timestamps using toLocaleTimeString", () => {
    const toLocaleTimeStringSpy = vi.spyOn(Date.prototype, "toLocaleTimeString").mockReturnValue("14:30");

    expect(formatTimestamp(new Date("2026-03-29T14:30:00.000Z"))).toBe("14:30");
    expect(toLocaleTimeStringSpy).toHaveBeenCalled();
  });

  it("flattens nested objects and arrays into primitive keys", () => {
    expect(
      flattenObject({
        user: { name: "Ada", profile: { team: "Core" } },
        tags: ["one", "two"],
        items: [{ id: 1 }, { id: 2 }],
      }),
    ).toEqual({
      name: "Ada",
      team: "Core",
      tags_0: "one",
      tags_1: "two",
      id: 2,
    });
  });

  it("validates only http and https urls", () => {
    expect(isUrl("https://example.com")).toBe(true);
    expect(isUrl("http://example.com")).toBe(true);
    expect(isUrl("ftp://example.com")).toBe(false);
    expect(isUrl("not-a-url")).toBe(false);
  });
});
