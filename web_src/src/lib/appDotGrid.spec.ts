import { describe, expect, it } from "vitest";
import { generateAppDotGrid } from "@/lib/appDotGrid";

const MIN_VISIBLE_DOTS = 12;
const MAX_VISIBLE_DOTS = 24;
const GRID_SIZE = 6;

function countVisibleDots(dots: boolean[]): number {
  return dots.filter(Boolean).length;
}

describe("generateAppDotGrid", () => {
  it("returns a 6x6 grid", () => {
    expect(generateAppDotGrid("canvas-1")).toHaveLength(GRID_SIZE * GRID_SIZE);
  });

  it("returns the same pattern for the same seed", () => {
    const seed = "a8153c8f-4364-475c-bd79-8f3f4be5fb58";
    expect(generateAppDotGrid(seed)).toEqual(generateAppDotGrid(seed));
  });

  it("returns different patterns for different seeds", () => {
    const first = generateAppDotGrid("canvas-a");
    const second = generateAppDotGrid("canvas-b");
    const third = generateAppDotGrid("canvas-c");

    expect(first).not.toEqual(second);
    expect(second).not.toEqual(third);
    expect(first).not.toEqual(third);
  });

  it("keeps visible dot count within the readable range", () => {
    const seeds = [
      "canvas-1",
      "canvas-2",
      "a8153c8f-4364-475c-bd79-8f3f4be5fb58",
      "3791cdb9-02e7-46ad-b675-0283cb7c59c3",
      "noble-quest",
    ];

    for (const seed of seeds) {
      const visibleCount = countVisibleDots(generateAppDotGrid(seed));
      expect(visibleCount).toBeGreaterThanOrEqual(MIN_VISIBLE_DOTS);
      expect(visibleCount).toBeLessThanOrEqual(MAX_VISIBLE_DOTS);
    }
  });
});
