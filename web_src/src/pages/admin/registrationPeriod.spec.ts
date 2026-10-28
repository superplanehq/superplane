import { expect, it } from "bun:test";
import { registrationPeriod } from "./registrationPeriod";

it("uses inclusive calendar days, Sunday weeks, and calendar months", () => {
  const now = new Date(2026, 9, 28, 14);
  const cases = [
    ["today", new Date(2026, 9, 28), new Date(2026, 9, 29)],
    ["week", new Date(2026, 9, 25), new Date(2026, 10, 1)],
    ["month", new Date(2026, 9, 1), new Date(2026, 10, 1)],
    ["7days", new Date(2026, 9, 22), new Date(2026, 9, 29)],
    ["30days", new Date(2026, 8, 29), new Date(2026, 9, 29)],
  ] as const;
  for (const [preset, start, end] of cases) {
    expect(registrationPeriod(preset, now)).toEqual({ start: start.toISOString(), end: end.toISOString() });
  }
  expect(registrationPeriod("week", new Date(2026, 9, 25, 12)).start).toBe(new Date(2026, 9, 25).toISOString());
  expect(registrationPeriod("week", new Date(2026, 9, 31, 12)).end).toBe(new Date(2026, 10, 1).toISOString());
});
