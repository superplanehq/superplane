import { describe, expect, it } from "vitest";
import { getNextCronExecution } from "@/lib/cron";

describe("cron", () => {
  it("finds the next execution for a 5-field expression", () => {
    const fromTime = new Date(2026, 2, 29, 10, 4, 30);
    const nextExecution = getNextCronExecution("5 10 * * *", fromTime);

    expect(nextExecution).not.toBeNull();
    expect(nextExecution?.getFullYear()).toBe(2026);
    expect(nextExecution?.getMonth()).toBe(2);
    expect(nextExecution?.getDate()).toBe(29);
    expect(nextExecution?.getHours()).toBe(10);
    expect(nextExecution?.getMinutes()).toBe(5);
  });

  it("supports 6-field expressions by ignoring the seconds field", () => {
    const fromTime = new Date(2026, 2, 29, 10, 4, 30);
    const nextExecution = getNextCronExecution("0 10 11 * * *", fromTime);

    expect(nextExecution).not.toBeNull();
    expect(nextExecution?.getDate()).toBe(29);
    expect(nextExecution?.getHours()).toBe(11);
    expect(nextExecution?.getMinutes()).toBe(10);
  });

  it("supports named weekdays and months", () => {
    const fromTime = new Date(2026, 2, 29, 10, 4, 30);
    const nextExecution = getNextCronExecution("0 9 * APR MON", fromTime);

    expect(nextExecution).not.toBeNull();
    expect(nextExecution?.getMonth()).toBe(3);
    expect(nextExecution?.getDate()).toBe(6);
    expect(nextExecution?.getDay()).toBe(1);
    expect(nextExecution?.getHours()).toBe(9);
    expect(nextExecution?.getMinutes()).toBe(0);
  });

  it("returns null for invalid cron field counts", () => {
    expect(getNextCronExecution("* * *", new Date("2026-03-29T10:04:30.000Z"))).toBeNull();
  });
});
