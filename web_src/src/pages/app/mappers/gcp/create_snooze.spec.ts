import { describe, expect, it } from "vitest";

import { eventStateRegistry } from ".";
import { buildComponentCtx, buildDetailsCtx, buildExecution, buildOutput } from "./vm_mapper_test_helpers";
import { createSnoozeMapper } from "./create_snooze";

const EXECUTED_AT = "2026-06-08T09:01:00Z";

describe("createSnoozeMapper.props", () => {
  it("surfaces display name, policy count, and duration", () => {
    const props = createSnoozeMapper.props(
      buildComponentCtx(
        { configuration: { displayName: "Deploy window", policies: ["p1", "p2"], duration: "1h" } },
        "gcp.monitoring.createSnooze",
      ),
    );
    expect(props.metadata).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ icon: "bell-off", label: "Deploy window" }),
        expect.objectContaining({ icon: "bell", label: "2 policies" }),
        expect.objectContaining({ icon: "clock", label: "1h" }),
      ]),
    );
  });
});

describe("createSnoozeMapper.getExecutionDetails", () => {
  it("maps the snooze output", () => {
    const out = {
      name: "projects/my-project/snoozes/55",
      id: "55",
      displayName: "Deploy window",
      policiesCount: 2,
      startTime: "2025-01-01T00:00:00Z",
      endTime: "2025-01-01T01:00:00Z",
    };
    const details = createSnoozeMapper.getExecutionDetails(
      buildDetailsCtx({ execution: { createdAt: EXECUTED_AT, outputs: { default: [buildOutput(out)] } } }),
    );
    expect(details).toEqual({
      "Executed At": new Date(EXECUTED_AT).toLocaleString(),
      "Display Name": "Deploy window",
      Policies: "2",
      Start: new Date(out.startTime).toLocaleString(),
      End: new Date(out.endTime).toLocaleString(),
    });
    expect(details["Snooze ID"]).toBeUndefined();
  });
});

describe("eventStateRegistry.monitoring.createSnooze", () => {
  it("maps success to created", () => {
    expect(eventStateRegistry["monitoring.createSnooze"].getState(buildExecution())).toBe("created");
  });
});
