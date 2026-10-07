import { describe, expect, it } from "vitest";

import { composerChipsWorking, nextPlanChipStatus } from "./planChipStatus";

describe("nextPlanChipStatus", () => {
  it("marks the first spec as ready", () => {
    expect(nextPlanChipStatus(null, "# First spec")).toEqual({ body: "# First spec", status: "ready" });
  });

  it("marks a later spec as updated", () => {
    const first = nextPlanChipStatus(null, "# First spec");
    expect(nextPlanChipStatus(first, "# Second spec")).toEqual({ body: "# Second spec", status: "updated" });
  });

  it("keeps ready when the spec does not change", () => {
    const first = nextPlanChipStatus(null, "# First spec");
    expect(nextPlanChipStatus(first, "# First spec")).toBe(first);
  });

  it("clears the status when the spec is gone", () => {
    expect(nextPlanChipStatus({ body: "# First spec", status: "ready" }, undefined)).toBeNull();
  });

  it("clears the badge after the plan is opened", () => {
    const first = nextPlanChipStatus(null, "# First spec");
    expect(nextPlanChipStatus(first, "# First spec", true)).toEqual({ body: "# First spec", status: undefined });
  });

  it("does not restore the badge when the plan is closed again", () => {
    const seen = nextPlanChipStatus({ body: "# First spec", status: "ready" }, "# First spec", true);
    expect(nextPlanChipStatus(seen, "# First spec", false)).toEqual({ body: "# First spec", status: undefined });
  });

  it("shows Updated after a new spec arrives while the plan is closed", () => {
    const seen = nextPlanChipStatus({ body: "# First spec", status: undefined }, "# First spec", false);
    expect(nextPlanChipStatus(seen, "# Second spec", false)).toEqual({ body: "# Second spec", status: "updated" });
  });

  it("does not show a badge when the spec changes while the plan is open", () => {
    const seen = nextPlanChipStatus({ body: "# First spec", status: undefined }, "# First spec", true);
    expect(nextPlanChipStatus(seen, "# Second spec", true)).toEqual({ body: "# Second spec", status: undefined });
  });
});

describe("composerChipsWorking", () => {
  it("is working while the machine runs even after a score exists", () => {
    expect(composerChipsWorking({ isAnalyzing: false, score: 2, machineStatus: "running" })).toBe(true);
  });

  it("is idle while waiting with a score", () => {
    expect(composerChipsWorking({ isAnalyzing: true, score: 2, machineStatus: "waiting" })).toBe(false);
  });

  it("is working on the first pass before a score exists", () => {
    expect(composerChipsWorking({ isAnalyzing: true, machineStatus: "waiting" })).toBe(true);
  });
});
