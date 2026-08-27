import { describe, expect, it } from "vitest";

import { logStatusTimeLabel, runningSpinnerFrame, tickingRunningClock } from "./logStatusTime";

describe("logStatusTimeLabel", () => {
  it("puts the status word and the clock on one right-hand label", () => {
    expect(logStatusTimeLabel("passed", "1m 20s")).toBe("Passed 01:20");
    expect(logStatusTimeLabel("running", "4m so far")).toBe("Running 04:00");
    expect(logStatusTimeLabel("failed", "12s")).toBe("Failed 00:12");
  });

  it("keeps the status word when there is no clock", () => {
    expect(logStatusTimeLabel("waiting")).toBe("Waiting");
    expect(logStatusTimeLabel("pending")).toBe("");
  });
});

describe("tickingRunningClock", () => {
  it("adds elapsed time to the sampled duration", () => {
    expect(tickingRunningClock("4m so far", 1_000, 1_000)).toBe("04:00");
    expect(tickingRunningClock("4m so far", 1_000, 2_000)).toBe("04:01");
    expect(tickingRunningClock(undefined, 1_000, 3_500)).toBe("00:02");
  });
});

describe("runningSpinnerFrame", () => {
  it("rotates the line through four frames", () => {
    expect(runningSpinnerFrame(0)).toBe("|");
    expect(runningSpinnerFrame(250)).toBe("/");
    expect(runningSpinnerFrame(500)).toBe("-");
    expect(runningSpinnerFrame(750)).toBe("\\");
    expect(runningSpinnerFrame(1_000)).toBe("|");
  });
});
