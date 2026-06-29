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
  it("getTitleAndSubtitle puts the condition in the title and the time in the subtitle", () => {
    const context: TriggerEventContext = { event: event({ state: "open", conditionName: "High CPU" }) };
    const { title, subtitle } = onAlertTriggerRenderer.getTitleAndSubtitle(context);
    expect(title).toBe("Alerting incident · High CPU");
    // The subtitle is a relative-time node (renderTimeAgo), not the incident content.
    expect(subtitle).not.toBe("");
  });

  it("getTitleAndSubtitle falls back to the policy name's last segment in the title", () => {
    const context: TriggerEventContext = {
      event: event({ state: "closed", policyName: "projects/p/alertPolicies/123" }),
    };
    expect(onAlertTriggerRenderer.getTitleAndSubtitle(context).title).toBe("Alerting incident · 123");
  });

  it("getTitleAndSubtitle is a bare title when there is no condition or policy", () => {
    const context: TriggerEventContext = { event: event({ state: "open" }) };
    expect(onAlertTriggerRenderer.getTitleAndSubtitle(context).title).toBe("Alerting incident");
  });

  it("getRootEventValues returns at most 6 curated fields with Emitted At first", () => {
    const context: TriggerEventContext = {
      event: event({
        incidentId: "i-1",
        state: "open",
        conditionName: "High CPU",
        summary: "CPU is high",
        policyName: "projects/p/alertPolicies/123",
        url: "https://console.cloud.google.com/...",
        resourceName: "my-vm",
        resourceDisplayName: "my-vm",
        observedValue: "0.93",
        thresholdValue: "0.8",
      }),
    };
    const values = onAlertTriggerRenderer.getRootEventValues(context);
    const keys = Object.keys(values);
    expect(keys[0]).toBe("Emitted At");
    expect(keys.length).toBeLessThanOrEqual(6);
    expect(values["State"]).toBe("open");
    expect(values["Condition"]).toBe("High CPU");
    expect(values["Summary"]).toBe("CPU is high");
    expect(values["Resource"]).toBe("my-vm");
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
    expect(props.lastEventData?.title).toBe("Alerting incident · High CPU");
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

  it("getTriggerProps surfaces the configured state filter on the node", () => {
    const context: TriggerRendererContext = {
      node: {
        id: "n1",
        name: "On Alert",
        componentName: "gcp.monitoring.onAlert",
        isCollapsed: false,
        configuration: { states: ["open", "closed"] },
        metadata: { notificationChannel: "projects/elffie/notificationChannels/4175146062038206967" },
      },
      definition,
      lastEvent: undefined,
    };
    const props = onAlertTriggerRenderer.getTriggerProps(context);
    // Shows the selected filter, not internal setup metadata like the channel name.
    expect(props.metadata?.some((m) => m.label === "Open, Closed")).toBe(true);
    expect(props.metadata?.some((m) => m.label === "4175146062038206967")).toBe(false);
  });
});
