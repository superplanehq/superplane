import { describe, expect, it } from "vitest";

import { getUsageMapper } from "./get_usage";
import { buildDetailsCtx, buildOutput } from "./test_helpers";

const PAYLOAD_TYPE = "opencodego.getUsage.result";

function detailsFor(data: unknown) {
  return getUsageMapper.getExecutionDetails!(buildDetailsCtx(buildOutput(PAYLOAD_TYPE, data)));
}

describe("opencodego getUsageMapper execution details", () => {
  it("shows percent used and reset time for each window", () => {
    const details = detailsFor({
      rollingStatus: "ok",
      rollingPercent: 0,
      rollingResetsAt: "2026-08-22T23:29:53.432Z",
      weeklyStatus: "ok",
      weeklyPercent: 51,
      weeklyResetsAt: "2026-08-24T00:00:00.432Z",
      monthlyStatus: "ok",
      monthlyPercent: 48,
      monthlyResetsAt: "2026-09-16T15:27:05.432Z",
    });

    expect(Object.keys(details)).toEqual(["Rolling", "Weekly", "Monthly"]);
    expect(details["Rolling"]).toMatch(/^0% used · resets /);
    expect(details["Weekly"]).toMatch(/^51% used · resets /);
    expect(details["Monthly"]).toMatch(/^48% used · resets /);
  });

  it("surfaces a non-ok status as its own detail", () => {
    const details = detailsFor({
      rollingStatus: "limit_exceeded",
      rollingPercent: 100,
      rollingResetsAt: "2026-08-23T01:00:00Z",
      weeklyStatus: "ok",
      weeklyPercent: 51,
      monthlyStatus: "ok",
      monthlyPercent: 12,
    });

    expect(details["Rolling Status"]).toBe("limit_exceeded");
    expect(Object.keys(details).some((key) => key.endsWith(" Status") && key !== "Rolling Status")).toBe(false);
  });

  it("omits windows that have no percent yet", () => {
    const details = detailsFor({ rollingPercent: 0, rollingStatus: "ok" });
    expect(Object.keys(details)).toEqual(["Rolling"]);
  });

  it("returns nothing when the execution has no output", () => {
    expect(getUsageMapper.getExecutionDetails!(buildDetailsCtx())).toEqual({});
  });
});
