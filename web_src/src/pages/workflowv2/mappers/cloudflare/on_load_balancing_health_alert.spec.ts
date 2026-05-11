import { describe, expect, it } from "vitest";
import { onLoadBalancingHealthAlertTriggerRenderer } from "./on_load_balancing_health_alert";
import type { ComponentDefinition, EventInfo, NodeInfo, TriggerEventContext, TriggerRendererContext } from "../types";

const NODE: NodeInfo = {
  id: "node-1",
  name: "Cloudflare Health",
  componentName: "cloudflare.onLoadBalancingHealthAlert",
  isCollapsed: false,
  configuration: {
    pool: "pool123",
    newHealth: ["Unhealthy"],
  },
  metadata: {},
};

const DEFINITION: ComponentDefinition = {
  name: "cloudflare.onLoadBalancingHealthAlert",
  label: "On Load Balancing Health Alert",
  description: "",
  icon: "activity",
  color: "orange",
};

const EVENT: EventInfo = {
  id: "event-1",
  createdAt: "2026-03-26T19:29:35.841Z",
  data: {
    alert_type: "load_balancing_health_alert",
    event_source: "origin",
    new_health: "Unhealthy",
    pool_id: "pool123",
    pool_name: "Production pool",
  },
  nodeId: "node-1",
  type: "cloudflare.loadBalancing.healthAlert",
};

const EVENT_WITH_ORIGIN: EventInfo = {
  ...EVENT,
  data: {
    ...(EVENT.data as object),
    origin_name: "api-primary",
    load_balancer_name: "api.example.com",
  },
};

describe("onLoadBalancingHealthAlertTriggerRenderer", () => {
  it("builds title and root event values from Cloudflare health alert payload", () => {
    const context: TriggerEventContext = { event: EVENT };

    expect(onLoadBalancingHealthAlertTriggerRenderer.getTitleAndSubtitle(context).title).toBe(
      "Production pool · origin · Unhealthy",
    );

    expect(onLoadBalancingHealthAlertTriggerRenderer.getRootEventValues(context)).toEqual({
      "Triggered At": new Date(EVENT.createdAt).toLocaleString(),
      "Alert Type": "load_balancing_health_alert",
      "Event Source": "origin",
      "New Health": "Unhealthy",
      Pool: "Production pool",
    });
  });

  it("prefers origin_name in the title when the payload includes it (Details tab unchanged)", () => {
    const context: TriggerEventContext = { event: EVENT_WITH_ORIGIN };

    expect(onLoadBalancingHealthAlertTriggerRenderer.getTitleAndSubtitle(context).title).toBe(
      "api-primary · origin · Unhealthy",
    );

    expect(onLoadBalancingHealthAlertTriggerRenderer.getRootEventValues(context)).toEqual({
      "Triggered At": new Date(EVENT_WITH_ORIGIN.createdAt).toLocaleString(),
      "Alert Type": "load_balancing_health_alert",
      "Event Source": "origin",
      "New Health": "Unhealthy",
      Pool: "Production pool",
    });
  });

  it("shows dash for Triggered At when createdAt is missing", () => {
    const values = onLoadBalancingHealthAlertTriggerRenderer.getRootEventValues({
      event: { ...EVENT, createdAt: "   " },
    });
    expect(values["Triggered At"]).toBe("-");
  });

  it("includes configured pool and health metadata", () => {
    const context: TriggerRendererContext = {
      node: NODE,
      definition: DEFINITION,
      lastEvent: EVENT,
    };

    const props = onLoadBalancingHealthAlertTriggerRenderer.getTriggerProps(context);

    expect(props.metadata).toEqual([
      { icon: "server", label: "Production pool" },
      { icon: "activity", label: "Unhealthy" },
    ]);
    expect(props.lastEventData?.title).toBe("Production pool · origin · Unhealthy");
  });

  it("prefers origin_name for lastEventData title when the payload includes it", () => {
    const context: TriggerRendererContext = {
      node: NODE,
      definition: DEFINITION,
      lastEvent: EVENT_WITH_ORIGIN,
    };

    const props = onLoadBalancingHealthAlertTriggerRenderer.getTriggerProps(context);

    expect(props.metadata).toEqual([
      { icon: "server", label: "Production pool" },
      { icon: "activity", label: "Unhealthy" },
    ]);
    expect(props.lastEventData?.title).toBe("api-primary · origin · Unhealthy");
  });

  it("prefers resolved pool name from node metadata", () => {
    const context: TriggerRendererContext = {
      node: { ...NODE, metadata: { poolName: "Resolved pool" } },
      definition: DEFINITION,
      lastEvent: undefined,
    };

    const props = onLoadBalancingHealthAlertTriggerRenderer.getTriggerProps(context);

    expect(props.metadata).toEqual([
      { icon: "server", label: "Resolved pool" },
      { icon: "activity", label: "Unhealthy" },
    ]);
  });
});
