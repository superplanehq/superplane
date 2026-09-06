import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { calculateNextTrigger } from "./schedule";

describe("calculateNextTrigger months schedule", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  function setNow(iso: string) {
    vi.setSystemTime(new Date(iso));
  }

  it("does not skip a month when now falls on day 31 (Jan -> Feb)", () => {
    setNow("2026-01-31T00:00:00.000Z");

    const next = calculateNextTrigger({
      type: "months",
      monthsInterval: 1,
      dayOfMonth: 15,
      hour: 9,
      minute: 0,
      timezone: "0",
    });

    expect(next).not.toBeNull();
    expect(next!.getFullYear()).toBe(2026);
    expect(next!.getMonth()).toBe(1); // February
    expect(next!.getDate()).toBe(15);
  });

  it("does not skip a month when now falls on day 31 (Mar -> Apr)", () => {
    setNow("2026-03-31T00:00:00.000Z");

    const next = calculateNextTrigger({
      type: "months",
      monthsInterval: 1,
      dayOfMonth: 15,
      hour: 9,
      minute: 0,
      timezone: "0",
    });

    expect(next).not.toBeNull();
    expect(next!.getFullYear()).toBe(2026);
    expect(next!.getMonth()).toBe(3); // April
    expect(next!.getDate()).toBe(15);
  });

  it("does not skip a month when now falls on day 31 (May -> Jun)", () => {
    setNow("2026-05-31T00:00:00.000Z");

    const next = calculateNextTrigger({
      type: "months",
      monthsInterval: 1,
      dayOfMonth: 15,
      hour: 9,
      minute: 0,
      timezone: "0",
    });

    expect(next).not.toBeNull();
    expect(next!.getFullYear()).toBe(2026);
    expect(next!.getMonth()).toBe(5); // June
    expect(next!.getDate()).toBe(15);
  });

  it("clamps dayOfMonth 31 to April's 30 days", () => {
    setNow("2026-03-10T00:00:00.000Z");

    const next = calculateNextTrigger({
      type: "months",
      monthsInterval: 1,
      dayOfMonth: 31,
      hour: 8,
      minute: 0,
      timezone: "0",
    });

    expect(next).not.toBeNull();
    expect(next!.getFullYear()).toBe(2026);
    expect(next!.getMonth()).toBe(3); // April
    expect(next!.getDate()).toBe(30);
  });

  it("clamps dayOfMonth 31 to February's 28 days", () => {
    setNow("2026-01-10T00:00:00.000Z");

    const next = calculateNextTrigger({
      type: "months",
      monthsInterval: 1,
      dayOfMonth: 31,
      hour: 8,
      minute: 0,
      timezone: "0",
    });

    expect(next).not.toBeNull();
    expect(next!.getFullYear()).toBe(2026);
    expect(next!.getMonth()).toBe(1); // February
    expect(next!.getDate()).toBe(28);
  });

  it("clamps dayOfMonth 30 to February's 28 days", () => {
    setNow("2026-01-10T00:00:00.000Z");

    const next = calculateNextTrigger({
      type: "months",
      monthsInterval: 1,
      dayOfMonth: 30,
      hour: 8,
      minute: 0,
      timezone: "0",
    });

    expect(next).not.toBeNull();
    expect(next!.getFullYear()).toBe(2026);
    expect(next!.getMonth()).toBe(1); // February
    expect(next!.getDate()).toBe(28);
  });
});
