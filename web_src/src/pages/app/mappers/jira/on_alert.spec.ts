import { describe, expect, it } from "vitest";
import type { ComponentDefinition, EventInfo, NodeInfo, TriggerEventContext, TriggerRendererContext } from "../types";
import { onAlertTriggerRenderer } from "./on_alert";

function event(data: Record<string, unknown>): EventInfo {
  return {
    id: "evt-1",
    createdAt: new Date("2026-07-17T09:30:00Z").toISOString(),
    nodeId: "node-1",
    type: "jira.alert",
    data,
  };
}

const alertData = {
  action: "created",
  alert: {
    alertId: "9e1c6a9e-1234-4a1a-8f1a-0a1b2c3d4e5f",
    tinyId: "123",
    message: "CPU usage above 90% on payments-api",
    priority: "P2",
    tags: ["production", "payments"],
    source: "Datadog",
  },
};

describe("onAlertTriggerRenderer.getTitleAndSubtitle", () => {
  it("derives an alert title from the event", () => {
    const context: TriggerEventContext = { event: event(alertData) };
    expect(onAlertTriggerRenderer.getTitleAndSubtitle(context).title).toBe(
      "#123 - CPU usage above 90% on payments-api",
    );
  });

  it("falls back to a generic title when no alert is present", () => {
    const context: TriggerEventContext = { event: event({}) };
    expect(onAlertTriggerRenderer.getTitleAndSubtitle(context).title).toBe("Alert event");
  });
});

describe("onAlertTriggerRenderer.getRootEventValues", () => {
  it("maps the alert fields to root event values", () => {
    const context: TriggerEventContext = { event: event(alertData) };
    const values = onAlertTriggerRenderer.getRootEventValues(context);
    expect(values["Received At"]).toBeDefined();
    expect(values["Action"]).toBe("Created");
    expect(values["Message"]).toBe("CPU usage above 90% on payments-api");
    expect(values["Priority"]).toBe("P2");
    expect(values["Source"]).toBe("Datadog");
    expect(values["Tags"]).toBe("production, payments");
  });

  it("falls back to dashes when the payload is empty", () => {
    const values = onAlertTriggerRenderer.getRootEventValues({ event: event({}) });
    expect(values["Message"]).toBe("-");
    expect(values["Priority"]).toBe("-");
  });
});

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "On Alert",
    componentName: "jira.onAlert",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildTriggerContext(overrides?: {
  node?: Partial<NodeInfo>;
  lastEvent?: EventInfo;
  definition?: Partial<ComponentDefinition>;
}): TriggerRendererContext {
  return {
    node: buildNode(overrides?.node),
    definition: {
      name: "jira.onAlert",
      label: "On Alert",
      description: "",
      icon: "jira",
      color: "red",
      ...overrides?.definition,
    },
    lastEvent: overrides?.lastEvent,
  } as TriggerRendererContext;
}

describe("onAlertTriggerRenderer.getTriggerProps", () => {
  it("renders the team and selected events", () => {
    const props = onAlertTriggerRenderer.getTriggerProps(
      buildTriggerContext({
        node: {
          configuration: { team: "team-1", events: ["created", "acknowledged"] },
          metadata: { team: { teamId: "team-1", teamName: "Platform" } },
        },
      }),
    );

    expect(props.metadata?.[0]).toEqual({ icon: "users", label: "Platform" });
    expect(props.metadata?.[1]).toEqual({ icon: "funnel", label: "Created, Acknowledged" });
  });

  it('shows "All teams" when no team is configured', () => {
    const props = onAlertTriggerRenderer.getTriggerProps(
      buildTriggerContext({ node: { configuration: { events: ["created"] } } }),
    );

    expect(props.metadata?.[0]).toEqual({ icon: "users", label: "All teams" });
  });
});
