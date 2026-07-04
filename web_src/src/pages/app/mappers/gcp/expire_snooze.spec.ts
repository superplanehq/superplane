import { describe, expect, it } from "vitest";

import { eventStateRegistry } from ".";
import { buildComponentCtx, buildDetailsCtx, buildExecution } from "./vm_mapper_test_helpers";
import { expireSnoozeMapper } from "./expire_snooze";

const EXECUTED_AT = "2026-06-08T09:01:00Z";

describe("expireSnoozeMapper.props (snooze name resolution)", () => {
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

describe("expireSnoozeMapper.getExecutionDetails", () => {
  it("does not throw and omits snooze fields when outputs are missing", () => {
    const ctx = buildDetailsCtx({ execution: { createdAt: EXECUTED_AT, outputs: undefined } });
    expect(() => expireSnoozeMapper.getExecutionDetails(ctx)).not.toThrow();
    const details = expireSnoozeMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Snooze ID"]).toBeUndefined();
  });
});

describe("eventStateRegistry.monitoring.expireSnooze", () => {
  it("maps success to expired", () => {
    expect(eventStateRegistry["monitoring.expireSnooze"].getState(buildExecution())).toBe("expired");
  });
});
