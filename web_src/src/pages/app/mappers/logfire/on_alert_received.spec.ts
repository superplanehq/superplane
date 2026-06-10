import { describe, expect, it } from "vitest";

import { onAlertReceivedTriggerRenderer } from "./on_alert_received";
import type { NodeInfo, TriggerEventContext, TriggerRendererContext, ComponentDefinition, EventInfo } from "../types";

// ── Helpers ──────────────────────────────────────────────────────────

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "On Alert Received",
    componentName: "logfire.onAlertReceived",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildDefinition(overrides?: Partial<ComponentDefinition>): ComponentDefinition {
  return {
    name: "onAlertReceived",
    label: "On Alert Received",
    description: "",
    icon: "logfire",
    color: "orange",
    ...overrides,
  };
}

function buildEvent(overrides?: Partial<NonNullable<EventInfo>>): EventInfo {
  return {
    id: "evt-1",
    createdAt: new Date().toISOString(),
    data: {},
    nodeId: "node-1",
    type: "logfire.onAlertReceived",
    ...overrides,
  };
}

// ── getTitleAndSubtitle ─────────────────────────────────────────────

describe("onAlertReceivedTriggerRenderer.getTitleAndSubtitle", () => {
  it("uses alert name as title", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: { alertName: "High Error Rate" } }) };
    expect(onAlertReceivedTriggerRenderer.getTitleAndSubtitle(ctx).title).toBe("High Error Rate");
  });

  it("falls back to message when alert name is empty", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: { alertName: "", message: "5 matching rows" } }) };
    expect(onAlertReceivedTriggerRenderer.getTitleAndSubtitle(ctx).title).toBe("5 matching rows");
  });

  it("falls back to default when both alertName and message are missing", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: {} }) };
    expect(onAlertReceivedTriggerRenderer.getTitleAndSubtitle(ctx).title).toBe("Logfire alert received");
  });

  it("builds subtitle from eventType and severity", () => {
    const ctx: TriggerEventContext = {
      event: buildEvent({ data: { alertName: "Test", eventType: "alert.fired", severity: "critical" } }),
    };
    const { subtitle } = onAlertReceivedTriggerRenderer.getTitleAndSubtitle(ctx);
    expect(subtitle).toContain("alert.fired");
    expect(subtitle).toContain("critical");
  });

  it("handles undefined event data gracefully", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ data: undefined }) };
    expect(() => onAlertReceivedTriggerRenderer.getTitleAndSubtitle(ctx)).not.toThrow();
  });
});

// ── getRootEventValues ──────────────────────────────────────────────

describe("onAlertReceivedTriggerRenderer.getRootEventValues", () => {
  it("extracts alert details from event data", () => {
    const ctx: TriggerEventContext = {
      event: buildEvent({
        data: {
          alertName: "High Error Rate",
          severity: "critical",
          message: "10 matching rows found",
          url: "https://logfire.pydantic.dev/alert/123",
        },
      }),
    };
    const values = onAlertReceivedTriggerRenderer.getRootEventValues(ctx);
    expect(values["Alert Name"]).toBe("High Error Rate");
    expect(values["Severity"]).toBe("critical");
    expect(values["Message"]).toBe("10 matching rows found");
    expect(values["Matching Rows"]).toBe("10");
    expect(values["View in Logfire"]).toBe("https://logfire.pydantic.dev/alert/123");
  });

  it("omits matching rows when message has no row count", () => {
    const ctx: TriggerEventContext = {
      event: buildEvent({ data: { alertName: "Test", message: "Something happened" } }),
    };
    expect(onAlertReceivedTriggerRenderer.getRootEventValues(ctx)["Matching Rows"]).toBeUndefined();
  });

  it("returns empty strings when event data is missing", () => {
    const values = onAlertReceivedTriggerRenderer.getRootEventValues({ event: buildEvent({ data: {} }) });
    expect(values["Alert Name"]).toBe("");
    expect(values["Severity"]).toBe("");
    expect(values["Message"]).toBe("");
  });

  it("includes received at from event createdAt", () => {
    const ctx: TriggerEventContext = { event: buildEvent({ createdAt: new Date().toISOString() }) };
    const values = onAlertReceivedTriggerRenderer.getRootEventValues(ctx);
    expect(values["Received At"]).toBeDefined();
    expect(values["Received At"]).not.toBe("");
  });

  it("falls back to event data timestamp when createdAt is missing", () => {
    const ctx: TriggerEventContext = {
      event: buildEvent({ createdAt: "", data: { timestamp: "2024-01-01T00:00:00Z" } }),
    };
    expect(onAlertReceivedTriggerRenderer.getRootEventValues(ctx)["Received At"]).toBe("2024-01-01T00:00:00Z");
  });
});

// ── getTriggerProps ─────────────────────────────────────────────────

describe("onAlertReceivedTriggerRenderer.getTriggerProps", () => {
  it("returns props with correct title from node name", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({ name: "My Alert Trigger" }),
      definition: buildDefinition(),
      lastEvent: buildEvent(),
    };
    expect(onAlertReceivedTriggerRenderer.getTriggerProps(ctx).title).toBe("My Alert Trigger");
  });

  it("falls back to definition label when node name is empty", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({ name: "" }),
      definition: buildDefinition({ label: "On Alert Received" }),
      lastEvent: buildEvent(),
    };
    expect(onAlertReceivedTriggerRenderer.getTriggerProps(ctx).title).toBe("On Alert Received");
  });

  it("includes project and alert metadata", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({
        metadata: { project: { id: "p1", name: "My Project" }, alert: { id: "a1", name: "My Alert" } },
      }),
      definition: buildDefinition(),
      lastEvent: buildEvent(),
    };
    const props = onAlertReceivedTriggerRenderer.getTriggerProps(ctx);
    expect(props.metadata?.find((m) => String(m.label).includes("Project"))).toBeDefined();
    expect(props.metadata?.find((m) => String(m.label).includes("Alert"))).toBeDefined();
  });

  it("limits metadata to 3 items", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({
        metadata: { project: { name: "Proj" }, alert: { name: "Alert" } },
        configuration: { project: "p", alert: "a" },
      }),
      definition: buildDefinition(),
      lastEvent: buildEvent(),
    };
    expect(onAlertReceivedTriggerRenderer.getTriggerProps(ctx).metadata!.length).toBeLessThanOrEqual(3);
  });

  it("includes lastEventData when lastEvent is provided", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode(),
      definition: buildDefinition(),
      lastEvent: buildEvent({ data: { alertName: "Test Alert", severity: "warning" } }),
    };
    const props = onAlertReceivedTriggerRenderer.getTriggerProps(ctx);
    expect(props.lastEventData).toBeDefined();
    expect(props.lastEventData!.title).toBe("Test Alert");
    expect(props.lastEventData!.state).toBe("triggered");
  });

  it("omits metadata when project and alert are not configured", () => {
    const ctx: TriggerRendererContext = {
      node: buildNode({ metadata: {}, configuration: {} }),
      definition: buildDefinition(),
      lastEvent: buildEvent(),
    };
    expect(onAlertReceivedTriggerRenderer.getTriggerProps(ctx).metadata).toEqual([]);
  });
});
