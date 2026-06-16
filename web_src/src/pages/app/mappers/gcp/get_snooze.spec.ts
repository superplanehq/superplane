import { describe, expect, it } from "vitest";

import { eventStateRegistry } from ".";
import { buildComponentCtx, buildDetailsCtx, buildExecution, buildOutput } from "./vm_mapper_test_helpers";
import { getSnoozeMapper } from "./get_snooze";

const EXECUTED_AT = "2026-06-08T09:01:00Z";

describe("getSnoozeMapper.props (snooze name resolution)", () => {
  it("prefers the resolved display name from node metadata over the ID", () => {
    const props = getSnoozeMapper.props(
      buildComponentCtx(
        {
          configuration: { snooze: "projects/my-project/snoozes/55" },
          metadata: { snoozeName: "projects/my-project/snoozes/55", displayName: "Deploy window", id: "55" },
        },
        "gcp.monitoring.getSnooze",
      ),
    );
    expect(props.metadata).toEqual([expect.objectContaining({ icon: "bell-off", label: "Deploy window" })]);
  });

  it("falls back to the snooze ID when no metadata was resolved", () => {
    const props = getSnoozeMapper.props(
      buildComponentCtx(
        { configuration: { snooze: "projects/my-project/snoozes/55" }, metadata: {} },
        "gcp.monitoring.getSnooze",
      ),
    );
    expect(props.metadata).toEqual([expect.objectContaining({ icon: "bell-off", label: "55" })]);
  });
});

describe("getSnoozeMapper.getExecutionDetails", () => {
  it("maps the snooze output", () => {
    const out = {
      name: "projects/my-project/snoozes/55",
      id: "55",
      displayName: "Deploy window",
      policiesCount: 2,
      startTime: "2025-01-01T00:00:00Z",
      endTime: "2025-01-01T01:00:00Z",
    };
    const details = getSnoozeMapper.getExecutionDetails(
      buildDetailsCtx({ execution: { createdAt: EXECUTED_AT, outputs: { default: [buildOutput(out)] } } }),
    );
    expect(details["Display Name"]).toBe("Deploy window");
    expect(details["Snooze ID"]).toBeUndefined();
    expect(details["Policies"]).toBe("2");
  });
});

describe("eventStateRegistry.monitoring.getSnooze", () => {
  it("maps success to fetched", () => {
    expect(eventStateRegistry["monitoring.getSnooze"].getState(buildExecution())).toBe("fetched");
  });
});
