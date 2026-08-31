import { describe, expect, it } from "vitest";

import {
  EARLY_USAGE_CLOSED_TASKS,
  buildEarlyUsageVelocityPoints,
  buildVelocityPoints,
  summarizePoints,
} from "./velocityPrototypeData";

describe("buildEarlyUsageVelocityPoints", () => {
  it("keeps people merges across the period and gives SuperPlane one day of output", () => {
    const points = buildEarlyUsageVelocityPoints(14);
    const today = points[points.length - 1]!;
    const earlier = points.slice(0, -1);

    expect(points).toHaveLength(14);
    expect(earlier.every((point) => point.people > 0)).toBe(true);
    expect(earlier.every((point) => point.superplane === 0)).toBe(true);
    expect(today.superplane).toBeGreaterThan(0);
  });

  it("closes the same number of tasks as the sample note reports", () => {
    const today = buildEarlyUsageVelocityPoints(14).at(-1)!;

    expect(today.superplane + today.waste).toBe(EARLY_USAGE_CLOSED_TASKS);
  });

  it("counts merged pull requests as people plus SuperPlane on every day", () => {
    const points = buildEarlyUsageVelocityPoints(30);

    for (const point of points) {
      expect(point.merged).toBe(point.people + point.superplane);
    }
  });

  it("reports no task time or cost on days without SuperPlane output", () => {
    const earlier = buildEarlyUsageVelocityPoints(14).slice(0, -1);

    expect(earlier.every((point) => point.runningHours === 0)).toBe(true);
    expect(earlier.every((point) => point.waitingHours === 0)).toBe(true);
    expect(earlier.every((point) => point.costUsd === 0)).toBe(true);
  });
});

describe("summarizePoints", () => {
  it("takes medians from days with closed tasks, so idle days do not hide task time", () => {
    const points = buildEarlyUsageVelocityPoints(14);
    const today = points.at(-1)!;

    const summary = summarizePoints(points);

    expect(summary.runningHours).toBe(today.runningHours);
    expect(summary.waitingHours).toBe(today.waitingHours);
    expect(summary.cycleHours).toBe(today.runningHours + today.waitingHours);
  });

  it("keeps waste rate and cost per merge scoped to SuperPlane output", () => {
    const summary = summarizePoints(buildEarlyUsageVelocityPoints(14));
    const today = buildEarlyUsageVelocityPoints(14).at(-1)!;

    expect(summary.wasteRate).toBe(Math.round((today.waste / (today.superplane + today.waste)) * 100));
    expect(summary.costPerMerge).toBeCloseTo(today.costUsd / today.superplane, 2);
  });

  it("still summarizes a full period where every day has closed tasks", () => {
    const summary = summarizePoints(buildVelocityPoints(14, 14));

    expect(summary.merged).toBeGreaterThan(0);
    expect(summary.cycleHours).toBeGreaterThan(0);
  });
});
