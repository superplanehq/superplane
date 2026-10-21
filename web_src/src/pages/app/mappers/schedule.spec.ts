import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { calculateNextTrigger } from "./schedule";

// System time is set with a local-time Date and results are asserted on
// local-time fields, so the tests are independent of the runner's timezone.
function setNow(year: number, monthIndex: number, day: number, hour = 10) {
  vi.setSystemTime(new Date(year, monthIndex, day, hour, 0, 0));
}

function localFields(date: Date | null) {
  expect(date).not.toBeNull();
  return {
    year: date!.getFullYear(),
    month: date!.getMonth(),
    day: date!.getDate(),
    hour: date!.getHours(),
    minute: date!.getMinutes(),
  };
}

describe("calculateNextTrigger months schedule", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  const monthsConfig = (dayOfMonth: number, monthsInterval = 1) => ({
    type: "months" as const,
    monthsInterval,
    dayOfMonth,
    hour: 9,
    minute: 0,
  });

  it("advances one month from a mid-month tick", () => {
    setNow(2025, 0, 10); // Jan 10
    const result = calculateNextTrigger(monthsConfig(15));
    expect(localFields(result)).toEqual({ year: 2025, month: 1, day: 15, hour: 9, minute: 0 });
  });

  it("does not skip February when now is Jan 31", () => {
    setNow(2025, 0, 31); // Jan 31: +1 month must be Feb 15, not Mar 15
    const result = calculateNextTrigger(monthsConfig(15));
    expect(localFields(result)).toEqual({ year: 2025, month: 1, day: 15, hour: 9, minute: 0 });
  });

  it("does not skip April when now is Mar 30", () => {
    setNow(2025, 2, 30);
    const result = calculateNextTrigger(monthsConfig(15));
    expect(localFields(result)).toEqual({ year: 2025, month: 3, day: 15, hour: 9, minute: 0 });
  });

  it("clamps dayOfMonth 31 to Apr 30", () => {
    setNow(2025, 2, 10); // Mar 10
    const result = calculateNextTrigger(monthsConfig(31));
    expect(localFields(result)).toEqual({ year: 2025, month: 3, day: 30, hour: 9, minute: 0 });
  });

  it("clamps dayOfMonth 31 to Feb 28", () => {
    setNow(2025, 0, 10); // Jan 10
    const result = calculateNextTrigger(monthsConfig(31));
    expect(localFields(result)).toEqual({ year: 2025, month: 1, day: 28, hour: 9, minute: 0 });
  });

  it("clamps dayOfMonth 31 to Feb 29 on leap years", () => {
    setNow(2024, 0, 10);
    const result = calculateNextTrigger(monthsConfig(31));
    expect(localFields(result)).toEqual({ year: 2024, month: 1, day: 29, hour: 9, minute: 0 });
  });

  it("wraps December into the next year", () => {
    setNow(2025, 11, 10); // Dec 10
    const result = calculateNextTrigger(monthsConfig(15));
    expect(localFields(result)).toEqual({ year: 2026, month: 0, day: 15, hour: 9, minute: 0 });
  });

  it("crosses the year boundary on multi-month intervals", () => {
    setNow(2025, 10, 10); // Nov 10, every 6 months
    const result = calculateNextTrigger(monthsConfig(31, 6));
    expect(localFields(result)).toEqual({ year: 2026, month: 4, day: 31, hour: 9, minute: 0 });
  });
});
