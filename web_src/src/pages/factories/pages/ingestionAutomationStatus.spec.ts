import { describe, expect, it } from "vitest";

import { findActiveRun, formatNextCycle, nextScheduledCycle } from "./ingestionAutomationStatus";

describe("ingestionAutomationStatus", () => {
  it("returns the active run", () => {
    expect(
      findActiveRun([
        { id: "run-1", state: "STATE_FINISHED" },
        { id: "run-2", state: "STATE_STARTED" },
      ])?.id,
    ).toBe("run-2");
    expect(findActiveRun([{ id: "run-1", state: "STATE_FINISHED" }])).toBeUndefined();
  });

  it("uses the next trigger from schedule metadata", () => {
    const next = nextScheduledCycle(
      [
        {
          component: "schedule",
          configuration: { type: "minutes", minutesInterval: 10 },
          metadata: { nextTrigger: "2026-08-21T12:10:00Z" },
        },
      ],
      [],
      undefined,
      new Date("2026-08-21T12:04:00Z"),
    );

    expect(next?.toISOString()).toBe("2026-08-21T12:10:00.000Z");
    expect(formatNextCycle(next!, new Date("2026-08-21T12:04:00Z"))).toBe("Next scan in 6 min");
  });

  it("advances a stale trigger to the next cycle", () => {
    const next = nextScheduledCycle(
      [
        {
          component: "schedule",
          configuration: { type: "minutes", minutesInterval: 10 },
          metadata: { nextTrigger: "2026-08-21T12:00:00Z" },
        },
      ],
      [],
      undefined,
      new Date("2026-08-21T12:16:00Z"),
    );

    expect(next?.toISOString()).toBe("2026-08-21T12:20:00.000Z");
  });
});
