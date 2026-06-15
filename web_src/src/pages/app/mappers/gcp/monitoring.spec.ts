import { describe, expect, it } from "vitest";

import { eventStateRegistry } from ".";
import { buildComponentCtx, buildDetailsCtx, buildExecution, buildOutput } from "./vm_mapper_test_helpers";
import {
  createAlertingPolicyMapper,
  createSnoozeMapper,
  expireSnoozeMapper,
  getSnoozeMapper,
} from "./monitoring";

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

  it("shows nothing for an unresolved expression value", () => {
    const props = expireSnoozeMapper.props(
      buildComponentCtx(
        { configuration: { snooze: '{{ $["Create Snooze"].data.name }}' }, metadata: {} },
        "gcp.monitoring.expireSnooze",
      ),
    );
    expect(props.metadata).toEqual([]);
  });
});

describe("snooze getExecutionDetails", () => {
  const out = {
    name: "projects/my-project/snoozes/55",
    id: "55",
    displayName: "Deploy window",
    policiesCount: 2,
    startTime: "2025-01-01T00:00:00Z",
    endTime: "2025-01-01T01:00:00Z",
  };

  it("maps the snooze output", () => {
    const details = getSnoozeMapper.getExecutionDetails(
      buildDetailsCtx({ execution: { createdAt: EXECUTED_AT, outputs: { default: [buildOutput(out)] } } }),
    );
    expect(details).toEqual({
      "Executed At": new Date(EXECUTED_AT).toLocaleString(),
      "Display Name": "Deploy window",
      "Snooze ID": "55",
      Policies: "2",
      Start: new Date(out.startTime).toLocaleString(),
      End: new Date(out.endTime).toLocaleString(),
    });
  });

  it("does not throw and omits snooze fields when outputs are missing", () => {
    const ctx = buildDetailsCtx({ execution: { createdAt: EXECUTED_AT, outputs: undefined } });
    expect(() => expireSnoozeMapper.getExecutionDetails(ctx)).not.toThrow();
    const details = expireSnoozeMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Snooze ID"]).toBeUndefined();
  });
});

describe("createAlertingPolicyMapper.getExecutionDetails (PromQL)", () => {
  it("surfaces the created PromQL policy details", () => {
    const details = createAlertingPolicyMapper.getExecutionDetails(
      buildDetailsCtx({
        execution: {
          createdAt: EXECUTED_AT,
          outputs: {
            default: [
              buildOutput({ displayName: "Demo PromQL Policy", id: "123", enabled: true, severity: "WARNING", conditionsCount: 1 }),
            ],
          },
        },
      }),
    );
    expect(details["Display Name"]).toBe("Demo PromQL Policy");
    expect(details["Policy ID"]).toBe("123");
    expect(details["Enabled"]).toBe("Yes");
    expect(details["Severity"]).toBe("WARNING");
    expect(details["Conditions"]).toBe("1");
  });
});

describe("eventStateRegistry snooze actions", () => {
  it("maps create/get/expire snooze success states", () => {
    expect(eventStateRegistry["monitoring.createSnooze"].getState(buildExecution())).toBe("created");
    expect(eventStateRegistry["monitoring.getSnooze"].getState(buildExecution())).toBe("fetched");
    expect(eventStateRegistry["monitoring.expireSnooze"].getState(buildExecution())).toBe("expired");
  });
});
