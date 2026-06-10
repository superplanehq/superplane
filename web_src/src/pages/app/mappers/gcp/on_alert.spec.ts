import { describe, expect, it } from "vitest";

import type { EventInfo, TriggerEventContext, TriggerRendererContext } from "../types";
import { onAlertTriggerRenderer } from "./on_alert";

const definition = {
  name: "gcp.monitoring.onAlert",
  label: "On Alert",
  description: "",
  icon: "bell",
  color: "blue",
};

function event(data: Record<string, unknown>): EventInfo {
  return {
    id: "evt-1",
    createdAt: new Date("2026-01-01T00:00:00Z").toISOString(),
    nodeId: "node-1",
    type: "gcp.monitoring.alert.incident",
    data,
  };
}

describe("onAlertTriggerRenderer", () => {
  it("getTitleAndSubtitle prefers the incident summary", () => {
    const context: TriggerEventContext = { event: event({ summary: "CPU > 80% on prod", state: "open" }) };
    const { title, subtitle } = onAlertTriggerRenderer.getTitleAndSubtitle(context);
    expect(title).toBe("Alerting incident");
    expect(subtitle).toBe("CPU > 80% on prod");
  });

  it("getTitleAndSubtitle falls back to state + condition name", () => {
    const context: TriggerEventContext = { event: event({ state: "open", conditionName: "High CPU" }) };
    expect(onAlertTriggerRenderer.getTitleAndSubtitle(context).subtitle).toBe("OPEN — High CPU");
  });

  it("getTitleAndSubtitle uses the policy name's last segment when no condition is present", () => {
    const context: TriggerEventContext = {
      event: event({ state: "closed", policyName: "projects/p/alertPolicies/123" }),
    };
    expect(onAlertTriggerRenderer.getTitleAndSubtitle(context).subtitle).toBe("CLOSED — 123");
  });

  it("getTitleAndSubtitle is empty when there is no data", () => {
    const context: TriggerEventContext = { event: event({}) };
    expect(onAlertTriggerRenderer.getTitleAndSubtitle(context).subtitle).toBe("");
  });

  it("getRootEventValues flattens the event data", () => {
    const context: TriggerEventContext = {
      event: event({ state: "open", policyName: "projects/p/alertPolicies/123" }),
    };
    const values = onAlertTriggerRenderer.getRootEventValues(context);
    expect(values.state).toBe("open");
    expect(values.policyName).toBe("projects/p/alertPolicies/123");
  });

  it("getTriggerProps uses the node name and surfaces the last event", () => {
    const context: TriggerRendererContext = {
      node: {
        id: "n1",
        name: "My Alert Trigger",
        componentName: "gcp.monitoring.onAlert",
        isCollapsed: false,
        configuration: {},
        metadata: {},
      },
      definition,
      lastEvent: event({ state: "open", conditionName: "High CPU" }),
    };
    const props = onAlertTriggerRenderer.getTriggerProps(context);
    expect(props.title).toBe("My Alert Trigger");
    expect(props.lastEventData?.title).toBe("Alerting incident");
    expect(props.lastEventData?.subtitle).toBe("OPEN — High CPU");
    expect(props.lastEventData?.state).toBe("triggered");
  });

  it("getTriggerProps falls back to the definition label and omits lastEventData when no event", () => {
    const context: TriggerRendererContext = {
      node: {
        id: "n1",
        name: "",
        componentName: "gcp.monitoring.onAlert",
        isCollapsed: false,
        configuration: {},
        metadata: {},
      },
      definition,
      lastEvent: undefined,
    };
    const props = onAlertTriggerRenderer.getTriggerProps(context);
    expect(props.title).toBe("On Alert");
    expect(props.lastEventData).toBeUndefined();
  });
});
