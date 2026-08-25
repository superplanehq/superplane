import { describe, expect, it } from "vitest";

import { formatDuration, formatMinutesSecondsDuration } from "@/lib/duration";

import { durationForExecution, elapsedForDisplay } from "./splitRunWorkOrderDisplay";

const START = "2026-08-21T12:00:00.000Z";
const FOUR_MINUTES = 4 * 60 * 1000;

describe("elapsedForDisplay", () => {
  it("keeps draft and waiting labels", () => {
    expect(elapsedForDisplay("draft")).toBe("Not started");
    expect(elapsedForDisplay("waiting", { createdAt: START, updatedAt: START })).toBe("Waiting");
  });

  it("formats running elapsed time from createdAt", () => {
    const now = Date.parse(START) + FOUR_MINUTES;
    expect(elapsedForDisplay("running", { createdAt: START, updatedAt: START }, now)).toBe(
      `${formatDuration(FOUR_MINUTES, { precision: "second" })} so far`,
    );
  });

  it("formats completed elapsed time from createdAt to updatedAt", () => {
    const updatedAt = new Date(Date.parse(START) + FOUR_MINUTES).toISOString();
    expect(elapsedForDisplay("completed", { createdAt: START, updatedAt })).toBe(
      formatDuration(FOUR_MINUTES, { precision: "second" }),
    );
  });
});

describe("durationForExecution", () => {
  it("formats a finished step from createdAt to updatedAt", () => {
    const updatedAt = new Date(Date.parse(START) + FOUR_MINUTES).toISOString();
    expect(durationForExecution({ createdAt: START, updatedAt }, "passed")).toBe(
      formatMinutesSecondsDuration(FOUR_MINUTES),
    );
  });

  it("formats a running step against now", () => {
    const now = Date.parse(START) + FOUR_MINUTES;
    expect(durationForExecution({ createdAt: START, updatedAt: START }, "running", now)).toBe(
      formatMinutesSecondsDuration(FOUR_MINUTES),
    );
  });

  it("shows a short duration when a finished step has no elapsed time", () => {
    expect(durationForExecution({ createdAt: START, updatedAt: START }, "passed")).toBe("<1s");
  });
});
