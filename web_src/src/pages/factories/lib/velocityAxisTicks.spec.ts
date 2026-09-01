import { describe, expect, it } from "vitest";

import { pickVelocityAxisTicks } from "./velocityAxisTicks";

function dayLabels(count: number): string[] {
  return Array.from({ length: count }, (_, index) => `Day ${index}`);
}

describe("pickVelocityAxisTicks", () => {
  it("returns every label unchanged when they already fit", () => {
    const labels = dayLabels(5);

    expect(pickVelocityAxisTicks(labels, 500)).toEqual(labels);
  });

  it("thins a 14-day window at a narrow 500px width", () => {
    const labels = dayLabels(14);
    const ticks = pickVelocityAxisTicks(labels, 500);

    expect(ticks.length).toBeLessThan(labels.length);
    expect(ticks[0]).toBe(labels[0]);
    expect(ticks[ticks.length - 1]).toBe(labels[labels.length - 1]);

    // No two picked labels are closer than the minimum label width allows.
    const indices = ticks.map((label) => labels.indexOf(label));
    for (let i = 1; i < indices.length; i++) {
      const pxPerDay = 500 / (labels.length - 1);
      const gapPx = (indices[i] - indices[i - 1]) * pxPerDay;
      expect(gapPx).toBeGreaterThanOrEqual(56 - 1);
    }
  });

  it("thins a 30-day window less aggressively at a wider width", () => {
    const labels = dayLabels(30);
    const narrow = pickVelocityAxisTicks(labels, 500);
    const wide = pickVelocityAxisTicks(labels, 760);

    expect(narrow[0]).toBe(labels[0]);
    expect(narrow[narrow.length - 1]).toBe(labels[labels.length - 1]);
    expect(wide[0]).toBe(labels[0]);
    expect(wide[wide.length - 1]).toBe(labels[labels.length - 1]);
    expect(wide.length).toBeGreaterThan(narrow.length);
    expect(wide.length).toBeLessThan(labels.length);
  });

  it("keeps at least the first and last label at the narrowest width", () => {
    const labels = dayLabels(30);

    expect(pickVelocityAxisTicks(labels, 1)).toEqual([labels[0], labels[labels.length - 1]]);
    expect(pickVelocityAxisTicks(labels, 0)).toEqual([labels[0], labels[labels.length - 1]]);
    expect(pickVelocityAxisTicks(labels, Number.NaN)).toEqual([labels[0], labels[labels.length - 1]]);
  });

  it("never duplicates a tick even when rounding collides", () => {
    const labels = dayLabels(3);

    // width fits exactly 3 ticks: no thinning, and no duplicate indices.
    const ticks = pickVelocityAxisTicks(labels, 168);
    expect(new Set(ticks).size).toBe(ticks.length);
  });
});
