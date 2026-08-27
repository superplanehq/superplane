import { describe, expect, it } from "vitest";

import { resolvePhaseRunStatus } from "./linePhaseRunStatus";

describe("resolvePhaseRunStatus", () => {
  it("maps execution states to board labels", () => {
    expect(resolvePhaseRunStatus({ state: "STATE_STARTED" })).toEqual({ kind: "running", label: "Executing" });
    expect(resolvePhaseRunStatus({ state: "STATE_CANCELLING" })).toEqual({ kind: "running", label: "Cancelling" });
    expect(resolvePhaseRunStatus({ state: "STATE_PENDING" })).toEqual({ kind: "queued", label: "Queued" });
    expect(resolvePhaseRunStatus({ state: "STATE_FINISHED", result: "RESULT_PASSED" })).toEqual({
      kind: "idle",
      label: "Passed",
    });
    expect(resolvePhaseRunStatus({ state: "STATE_FINISHED", result: "RESULT_FAILED" })).toEqual({
      kind: "failed",
      label: "Failed",
    });
  });
});
