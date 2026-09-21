import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { calculateNextTrigger } from "./schedule";

// 2025-01-08 is a Wednesday. Dates are built in the machine's local timezone
// with no configured offset, so expectations are timezone-independent.
function weeksConfig(overrides: Record<string, unknown> = {}) {
  return {
    type: "weeks" as const,
    weeksInterval: 1,
    weekDays: ["monday", "wednesday", "friday"],
    hour: 9,
    minute: 0,
    ...overrides,
  };
}

describe("calculateNextTrigger weeks schedule", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("prefers the backend-provided nextTrigger when present", () => {
    vi.setSystemTime(new Date(2025, 0, 8, 10, 0, 0));
    const result = calculateNextTrigger(weeksConfig(), "2025-01-10T09:00:00Z");
    expect(result?.toISOString()).toBe("2025-01-10T09:00:00.000Z");
  });

  it("fires later the same day when the slot is still ahead", () => {
    vi.setSystemTime(new Date(2025, 0, 8, 8, 0, 0)); // Wednesday before 09:00
    const result = calculateNextTrigger(weeksConfig({ weekDays: ["wednesday"] }));
    expect(result).toEqual(new Date(2025, 0, 8, 9, 0, 0));
  });

  it("fires on the next configured day of the current week", () => {
    vi.setSystemTime(new Date(2025, 0, 8, 10, 0, 0)); // Wednesday after 09:00
    const result = calculateNextTrigger(weeksConfig());
    expect(result).toEqual(new Date(2025, 0, 10, 9, 0, 0)); // Friday
  });

  it("jumps a full interval once the active week is exhausted", () => {
    vi.setSystemTime(new Date(2025, 0, 8, 10, 0, 0)); // Wednesday
    const result = calculateNextTrigger(weeksConfig({ weekDays: ["monday"], weeksInterval: 2 }));
    expect(result).toEqual(new Date(2025, 0, 20, 9, 0, 0)); // Monday, one off week skipped
  });

  it("keeps the remaining days of the active week with interval > 1", () => {
    vi.setSystemTime(new Date(2025, 0, 8, 10, 0, 0)); // Wednesday
    const result = calculateNextTrigger(weeksConfig({ weekDays: ["friday"], weeksInterval: 2 }));
    expect(result).toEqual(new Date(2025, 0, 10, 9, 0, 0));
  });

  it("treats sunday as the first day of the week", () => {
    vi.setSystemTime(new Date(2025, 0, 8, 10, 0, 0)); // Wednesday
    const result = calculateNextTrigger(weeksConfig({ weekDays: ["sunday"] }));
    expect(result).toEqual(new Date(2025, 0, 12, 9, 0, 0));
  });
});
