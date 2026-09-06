import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { calculateNextTrigger, type ScheduleConfiguration } from "./schedule";

function buildMonthsConfig(interval: number, dayOfMonth: number, hour = 0, minute = 0): ScheduleConfiguration {
  return {
    type: "months",
    monthsInterval: interval,
    dayOfMonth,
    hour,
    minute,
  };
}

describe("calculateNextTrigger (months)", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  function expectDate(
    result: Date | null,
    expected: { year: number; month: number; day: number; hour: number; minute: number },
  ) {
    expect(result).not.toBeNull();
    expect(result!.getFullYear()).toBe(expected.year);
    expect(result!.getMonth()).toBe(expected.month);
    expect(result!.getDate()).toBe(expected.day);
    expect(result!.getHours()).toBe(expected.hour);
    expect(result!.getMinutes()).toBe(expected.minute);
  }

  it("does not skip a month when now is on day 31", () => {
    vi.setSystemTime(new Date(2026, 0, 31, 0, 0, 0)); // 2026-01-31
    const result = calculateNextTrigger(buildMonthsConfig(1, 15));
    // month is 0-indexed: February = 1
    expectDate(result, { year: 2026, month: 1, day: 15, hour: 0, minute: 0 });
  });

  it("does not skip a month when now is on day 31 of a 31-day month (March)", () => {
    vi.setSystemTime(new Date(2026, 2, 31, 0, 0, 0)); // 2026-03-31
    const result = calculateNextTrigger(buildMonthsConfig(1, 15));
    expectDate(result, { year: 2026, month: 3, day: 15, hour: 0, minute: 0 }); // April = 3
  });

  it("does not skip a month when now is on day 31 of another 31-day month (May)", () => {
    vi.setSystemTime(new Date(2026, 4, 31, 0, 0, 0)); // 2026-05-31
    const result = calculateNextTrigger(buildMonthsConfig(1, 15));
    expectDate(result, { year: 2026, month: 5, day: 15, hour: 0, minute: 0 }); // June = 5
  });

  it("clamps dayOfMonth 31 to the last day of a 30-day target month", () => {
    vi.setSystemTime(new Date(2026, 2, 10, 0, 0, 0)); // 2026-03-10
    const result = calculateNextTrigger(buildMonthsConfig(1, 31));
    expectDate(result, { year: 2026, month: 3, day: 30, hour: 0, minute: 0 }); // April 30
  });

  it("clamps dayOfMonth 31 to the last day of February (non-leap year)", () => {
    vi.setSystemTime(new Date(2026, 0, 10, 0, 0, 0)); // 2026-01-10
    const result = calculateNextTrigger(buildMonthsConfig(1, 31));
    expectDate(result, { year: 2026, month: 1, day: 28, hour: 0, minute: 0 }); // February 28
  });

  it("clamps dayOfMonth 30 to the last day of February (non-leap year)", () => {
    vi.setSystemTime(new Date(2026, 0, 10, 0, 0, 0)); // 2026-01-10
    const result = calculateNextTrigger(buildMonthsConfig(1, 30));
    expectDate(result, { year: 2026, month: 1, day: 28, hour: 0, minute: 0 }); // February 28
  });

  it("rolls over the year from December to January", () => {
    vi.setSystemTime(new Date(2026, 11, 31, 0, 0, 0)); // 2026-12-31
    const result = calculateNextTrigger(buildMonthsConfig(1, 31));
    expectDate(result, { year: 2027, month: 0, day: 31, hour: 0, minute: 0 }); // 2027-01-31
  });

  it("handles a multi-month interval crossing a year boundary", () => {
    vi.setSystemTime(new Date(2026, 10, 5, 0, 0, 0)); // 2026-11-05
    const result = calculateNextTrigger(buildMonthsConfig(3, 15, 9, 0));
    expectDate(result, { year: 2027, month: 1, day: 15, hour: 9, minute: 0 }); // 2027-02-15 09:00
  });

  it("keeps the existing mid-month behavior unchanged", () => {
    vi.setSystemTime(new Date(2025, 0, 1, 10, 0, 0)); // 2025-01-01
    const result = calculateNextTrigger(buildMonthsConfig(1, 15, 14, 30));
    expectDate(result, { year: 2025, month: 1, day: 15, hour: 14, minute: 30 }); // 2025-02-15 14:30
  });
});
