import { describe, expect, it } from "vitest";

import type { TriggerEventContext, TriggerRendererContext, EventInfo } from "../types";
import { onTunnelHealthTriggerRenderer } from "./on_tunnel_health";

const definition = {
  name: "cloudflare.onTunnelHealth",
  label: "On Tunnel Health",
  description: "",
  icon: "activity",
  color: "orange",
};

describe("onTunnelHealthTriggerRenderer", () => {
  it("builds title from event data", () => {
    const event: EventInfo = {
      id: "evt-1",
      createdAt: new Date().toISOString(),
      nodeId: "node-1",
      type: "cloudflare.tunnel.healthEvent",
      data: {
        tunnel_name: "api",
        new_status: "Down",
      },
    };
    const context: TriggerEventContext = { event };
    expect(onTunnelHealthTriggerRenderer.getTitleAndSubtitle(context).title).toContain("api");
    expect(onTunnelHealthTriggerRenderer.getTitleAndSubtitle(context).title).toContain("Down");
  });

  it("getRootEventValues includes status", () => {
    const event: EventInfo = {
      id: "evt-1",
      createdAt: new Date().toISOString(),
      nodeId: "node-1",
      type: "cloudflare.tunnel.healthEvent",
      data: { new_status: "Degraded", tunnel_id: "t1" },
    };
    const context: TriggerEventContext = { event };
    const values = onTunnelHealthTriggerRenderer.getRootEventValues(context);
    expect(values["New Status"]).toBe("Degraded");
    expect(values.Tunnel).toBe("t1");
  });

  it("getRootEventValues reads alert_type from merged-style payload", () => {
    const event: EventInfo = {
      id: "evt-1",
      createdAt: new Date().toISOString(),
      nodeId: "node-1",
      type: "cloudflare.tunnel.healthEvent",
      data: {
        alert_type: "tunnel_health_event",
        new_status: "TUNNEL_STATUS_TYPE_DEGRADED",
        tunnel_name: "tunnel-name",
        tunnel_id: "abc",
      },
    };
    const values = onTunnelHealthTriggerRenderer.getRootEventValues({ event });
    expect(values["Alert Type"]).toBe("tunnel_health_event");
    expect(values["New Status"]).toBe("Degraded");
  });

  it("getTriggerProps returns metadata from configuration", () => {
    const context: TriggerRendererContext = {
      node: {
        id: "n1",
        name: "Tunnel trigger",
        componentName: "cloudflare.onTunnelHealth",
        isCollapsed: false,
        configuration: {
          newStatus: ["TUNNEL_STATUS_TYPE_DOWN", "TUNNEL_STATUS_TYPE_HEALTHY"],
        },
        metadata: {},
      },
      definition,
      lastEvent: undefined,
    };
    const props = onTunnelHealthTriggerRenderer.getTriggerProps(context);
    expect(props.metadata?.some((m) => m.label === "Down, Healthy")).toBe(true);
  });
});
