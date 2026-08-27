import { describe, expect, it } from "vitest";

import { logStatusTimeLabel, tickingRunningClock } from "./logStatusTime";

describe("logStatusTimeLabel", () => {
  it("returns only the clock", () => {
    expect(logStatusTimeLabel("1m 20s")).toBe("01:20");
    expect(logStatusTimeLabel("4m so far")).toBe("04:00");
    expect(logStatusTimeLabel("12s")).toBe("00:12");
  });

  it("is empty when there is no clock", () => {
    expect(logStatusTimeLabel()).toBe("");
    expect(logStatusTimeLabel("")).toBe("");
  });
});

describe("tickingRunningClock", () => {
  it("adds elapsed time to the sampled duration", () => {
    expect(tickingRunningClock("4m so far", 1_000, 1_000)).toBe("04:00");
    expect(tickingRunningClock("4m so far", 1_000, 2_000)).toBe("04:01");
    expect(tickingRunningClock(undefined, 1_000, 3_500)).toBe("00:02");
  });
});
