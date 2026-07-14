import { describe, expect, it } from "vitest";

import type { EventInfo, TriggerRendererContext } from "../types";
import { onBroadcastTriggerRenderer } from "./on_broadcast";

const definition = {
  name: "onBroadcast",
  label: "On Broadcast",
  description: "",
  icon: "rss",
  color: "gray",
};

describe("onBroadcastTriggerRenderer", () => {
  it("builds title and subtitle from broadcast event data", () => {
    const event = broadcastEvent({
      app: { id: "app-1", name: "Orders App" },
      node: { id: "broadcast-message", name: "Broadcast Message" },
      payload: { message: "Order shipped" },
    });

    const { title, subtitle } = onBroadcastTriggerRenderer.getTitleAndSubtitle({ event });

    expect(title).toBe("Broadcast from Orders App");
    expect(subtitle).toBeTruthy();
  });

  it("returns broadcast details for the sidebar", () => {
    const event = broadcastEvent({
      app: { id: "app-1", name: "Orders App" },
      node: { id: "broadcast-message", name: "Broadcast Message" },
      payload: { message: "Order shipped" },
    });

    const values = onBroadcastTriggerRenderer.getRootEventValues({ event });

    expect(values.App).toBe("Orders App");
    expect(values["Source node"]).toBe("Broadcast Message");
    expect(values.Payload).toBe("Order shipped");
    expect(values["Received at"]).toBeTruthy();
  });

  it("renders subscribed app as node metadata", () => {
    const context: TriggerRendererContext = {
      node: {
        id: "node-1",
        name: "On Broadcast",
        componentName: "onBroadcast",
        isCollapsed: false,
        metadata: {
          app: {
            id: "app-1",
            name: "Orders App",
          },
        },
        configuration: {
          app: "app-1",
        },
      },
      definition,
      lastEvent: undefined,
    };

    const props = onBroadcastTriggerRenderer.getTriggerProps(context);

    expect(props.metadata?.map((item) => item.label)).toEqual(["Orders App"]);
    expect(props.iconSlug).toBe("rss");
  });
});

function broadcastEvent(data: Record<string, unknown>): EventInfo {
  return {
    id: "evt-1",
    createdAt: new Date().toISOString(),
    nodeId: "node-1",
    type: "app.broadcast",
    data,
  };
}
