import { describe, expect, it } from "vitest";
import { getNextCronExecution, getNextCronExecutions } from "@/lib/cron";

function format(date: Date | null | undefined): string {
  if (!date) {
    return "none";
  }

  const pad = (value: number) => value.toString().padStart(2, "0");

  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(
    date.getMinutes(),
  )}:${pad(date.getSeconds())}`;
}

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

  it("supports 6-field expressions including the seconds field", () => {
    const fromTime = new Date(2026, 2, 29, 10, 4, 30);
    const nextExecution = getNextCronExecution("0 10 11 * * *", fromTime);

    expect(nextExecution).not.toBeNull();
    expect(nextExecution?.getDate()).toBe(29);
    expect(nextExecution?.getHours()).toBe(11);
    expect(nextExecution?.getMinutes()).toBe(10);
    expect(nextExecution?.getSeconds()).toBe(0);

    expect(format(getNextCronExecution("30 * * * * *", new Date(2026, 2, 29, 10, 4, 10)))).toBe("2026-03-29 10:04:30");
    expect(format(getNextCronExecution("30 * * * * *", new Date(2026, 2, 29, 10, 4, 30)))).toBe("2026-03-29 10:05:30");
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

  it("resolves schedules that are more than a year away", () => {
    // Feb 29 only exists on leap years; the next one after 2026 is in 2028.
    expect(format(getNextCronExecution("0 0 29 2 *", new Date(2026, 2, 1, 0, 0, 0)))).toBe("2028-02-29 00:00:00");
    expect(format(getNextCronExecution("0 0 1 1 *", new Date(2026, 0, 2, 10, 0, 0)))).toBe("2027-01-01 00:00:00");
  });

  it("returns null for schedules that never fire", () => {
    expect(getNextCronExecution("0 0 30 2 *", new Date(2026, 0, 1, 0, 0, 0))).toBeNull();
  });

  it("returns null for invalid expressions", () => {
    const fromTime = new Date(2026, 2, 29, 10, 4, 30);

    expect(getNextCronExecution("* * *", fromTime)).toBeNull();
    expect(getNextCronExecution("", fromTime)).toBeNull();
    expect(getNextCronExecution("70 * * * *", fromTime)).toBeNull();
    expect(getNextCronExecution("* 25 * * *", fromTime)).toBeNull();
    expect(getNextCronExecution("0 0 * * 7", fromTime)).toBeNull();
    expect(getNextCronExecution("*/0 * * * *", fromTime)).toBeNull();
    expect(getNextCronExecution("*/abc * * * *", fromTime)).toBeNull();
    expect(getNextCronExecution("10-5 * * * *", fromTime)).toBeNull();
    expect(getNextCronExecution("0 0 * FOO *", fromTime)).toBeNull();
  });

  it("supports lists, ranges and steps", () => {
    const fromTime = new Date(2026, 2, 29, 10, 4, 30);

    expect(format(getNextCronExecution("0 9,17 * * *", fromTime))).toBe("2026-03-29 17:00:00");
    expect(format(getNextCronExecution("*/15 * * * *", fromTime))).toBe("2026-03-29 10:15:00");
    // A step on a single value runs from that value to the end of the range,
    // so `5/15` fires at :05, :20, :35 and :50.
    expect(format(getNextCronExecution("5/15 * * * *", fromTime))).toBe("2026-03-29 10:05:00");
    expect(format(getNextCronExecution("5/15 * * * *", new Date(2026, 2, 29, 10, 5, 0)))).toBe("2026-03-29 10:20:00");
    expect(format(getNextCronExecution("0 8-10 * * *", new Date(2026, 2, 29, 8, 30, 0)))).toBe("2026-03-29 09:00:00");
  });

  it("treats ? like a wildcard", () => {
    const fromTime = new Date(2026, 2, 29, 10, 4, 30);

    expect(format(getNextCronExecution("0 12 ? * *", fromTime))).toBe("2026-03-29 12:00:00");
  });

  it("matches either day-of-month or weekday when both are restricted", () => {
    // The next Friday (Apr 3) comes before the 13th, and either field matching
    // is enough.
    expect(format(getNextCronExecution("0 0 13 * FRI", new Date(2026, 3, 1, 0, 0, 0)))).toBe("2026-04-03 00:00:00");
  });

  it("requires both day fields to match when one of them is a wildcard", () => {
    // Plain `*` day-of-month: the weekday has to match too.
    expect(format(getNextCronExecution("0 0 * * MON", new Date(2026, 3, 1, 0, 0, 0)))).toBe("2026-04-06 00:00:00");
    // A step makes the field restricted, so either field matching is enough:
    // `*/2` covers the odd days, and Apr 3 is one of them.
    expect(format(getNextCronExecution("0 0 */2 * MON", new Date(2026, 3, 1, 0, 0, 0)))).toBe("2026-04-03 00:00:00");
  });

  it("returns a series of upcoming executions", () => {
    const executions = getNextCronExecutions("0 9 * * MON-FRI", new Date(2026, 3, 2, 12, 0, 0), 4);

    expect(executions.map(format)).toEqual([
      "2026-04-03 09:00:00",
      "2026-04-06 09:00:00",
      "2026-04-07 09:00:00",
      "2026-04-08 09:00:00",
    ]);
  });

  // Expectations captured from the backend scheduler
  // (github.com/robfig/cron/v3, the parser used by pkg/triggers/schedule).
  it.each([
    ["5 10 * * *", new Date(2026, 2, 29, 10, 4, 30), "2026-03-29 10:05:00"],
    ["0 10 11 * * *", new Date(2026, 2, 29, 10, 4, 30), "2026-03-29 11:10:00"],
    ["30 * * * * *", new Date(2026, 2, 29, 10, 4, 30), "2026-03-29 10:05:30"],
    ["0 9 * APR MON", new Date(2026, 2, 29, 10, 4, 30), "2026-04-06 09:00:00"],
    ["0 9,17 * * *", new Date(2026, 2, 29, 10, 4, 30), "2026-03-29 17:00:00"],
    ["*/15 * * * *", new Date(2026, 2, 29, 10, 4, 30), "2026-03-29 10:15:00"],
    ["5/15 * * * *", new Date(2026, 2, 29, 10, 4, 30), "2026-03-29 10:05:00"],
    ["0 8-10 * * *", new Date(2026, 2, 29, 8, 30, 0), "2026-03-29 09:00:00"],
    ["0 12 ? * *", new Date(2026, 2, 29, 10, 4, 30), "2026-03-29 12:00:00"],
    ["0 0 13 * FRI", new Date(2026, 3, 1, 0, 0, 0), "2026-04-03 00:00:00"],
    ["0 0 */2 * MON", new Date(2026, 3, 1, 0, 0, 0), "2026-04-03 00:00:00"],
    ["0 9 * * MON-FRI", new Date(2026, 3, 2, 12, 0, 0), "2026-04-03 09:00:00"],
    ["0 0 29 2 *", new Date(2026, 2, 1, 0, 0, 0), "2028-02-29 00:00:00"],
    ["0 0 1 1 *", new Date(2026, 0, 2, 10, 0, 0), "2027-01-01 00:00:00"],
    ["0 0 30 2 *", new Date(2026, 0, 1, 0, 0, 0), "none"],
    ["70 * * * *", new Date(2026, 2, 29, 10, 4, 30), "none"],
    ["* 25 * * *", new Date(2026, 2, 29, 10, 4, 30), "none"],
    ["0 0 * * 7", new Date(2026, 2, 29, 10, 4, 30), "none"],
    ["*/0 * * * *", new Date(2026, 2, 29, 10, 4, 30), "none"],
    ["10-5 * * * *", new Date(2026, 2, 29, 10, 4, 30), "none"],
  ])("matches the backend scheduler for %s", (expression, fromTime, expected) => {
    expect(format(getNextCronExecution(expression, fromTime))).toBe(expected);
  });

  it("resolves schedules without scanning minute by minute", () => {
    const start = performance.now();
    for (let i = 0; i < 200; i++) {
      getNextCronExecution("0 0 29 2 *", new Date(2026, 0, 1, 0, 0, 0));
      getNextCronExecution("0 0 30 2 *", new Date(2026, 0, 1, 0, 0, 0));
    }

    // The previous minute-by-minute scan took ~60ms for a single one of these.
    expect(performance.now() - start).toBeLessThan(1000);
  });
});
