import { describe, expect, it } from "bun:test";

import type { EventInfo } from "../types";
import { onAlertFiresTriggerRenderer } from "./on_alert";

const createdAt = "2026-01-01T00:00:00Z";

function event(data: unknown): EventInfo {
  return {
    id: "evt-1",
    createdAt,
    nodeId: "node-1",
    type: "elastic.alert",
    data,
  };
}

describe("onAlertFiresTriggerRenderer.getRootEventValues", () => {
  it("returns no details without an event", () => {
    expect(onAlertFiresTriggerRenderer.getRootEventValues({ event: undefined })).toEqual({});
  });

  it.each([
    { label: "missing", data: undefined },
    { label: "null", data: null },
    { label: "string", data: "invalid payload" },
    { label: "array", data: [] },
  ])("retains the event timestamp for a $label payload", ({ data }) => {
    expect(onAlertFiresTriggerRenderer.getRootEventValues({ event: event(data) })).toEqual({
      "Received At": new Date(createdAt).toLocaleString(),
    });
  });

  it("prefers the payload timestamp and includes all alert details", () => {
    const timestamp = "2026-01-01T01:00:00Z";
    const values = onAlertFiresTriggerRenderer.getRootEventValues({
      event: event({
        timestamp,
        ruleName: "High CPU",
        alertName: "CPU above threshold",
        ruleId: "rule-1",
        spaceId: "production",
        status: "active",
        severity: "critical",
        tags: ["infrastructure", "cpu"],
      }),
    });

    expect(values).toEqual({
      "Received At": new Date(timestamp).toLocaleString(),
      Rule: "High CPU",
      "Alert Name": "CPU above threshold",
      Space: "production",
      Status: "active",
      Severity: "critical",
      Tags: "infrastructure, cpu",
    });
  });

  it.each([
    { ruleName: "High CPU", alertName: "High CPU", ruleId: "rule-1", expectedRule: "High CPU" },
    { ruleName: "", alertName: "CPU above threshold", ruleId: "rule-1", expectedRule: "CPU above threshold" },
    { ruleName: "", alertName: "", ruleId: "rule-1", expectedRule: "rule-1" },
  ])("selects $expectedRule without duplicate or empty details", ({ expectedRule, ...payload }) => {
    expect(
      onAlertFiresTriggerRenderer.getRootEventValues({
        event: event({ ...payload, timestamp: "", spaceId: "", status: "", severity: "", tags: [] }),
      }),
    ).toEqual({
      "Received At": new Date(createdAt).toLocaleString(),
      Rule: expectedRule,
    });
  });

  it("omits empty rule fields", () => {
    expect(
      onAlertFiresTriggerRenderer.getRootEventValues({
        event: event({ ruleName: "", alertName: "", ruleId: "" }),
      }),
    ).toEqual({ "Received At": new Date(createdAt).toLocaleString() });
  });

  it("omits non-string scalar fields and retains only string tags", () => {
    expect(
      onAlertFiresTriggerRenderer.getRootEventValues({
        event: event({
          timestamp: 123,
          ruleName: 42,
          alertName: false,
          ruleId: {},
          spaceId: [],
          status: true,
          severity: null,
          tags: ["infrastructure", 42, null, "cpu", {}],
        }),
      }),
    ).toEqual({
      "Received At": new Date(createdAt).toLocaleString(),
      Tags: "infrastructure, cpu",
    });
  });
});
