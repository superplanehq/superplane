import { describe, expect, it } from "vitest";
import type { ComponentBaseContext, ComponentDefinition, NodeInfo } from "../types";
import { createMonitorMapper } from "./create_monitor";
import { deleteMonitorMapper } from "./delete_monitor";

const DEFINITION: ComponentDefinition = {
  name: "cloudflare.createMonitor",
  label: "Create Monitor",
  description: "",
  icon: "activity",
  color: "orange",
};

function baseContext(node: NodeInfo, definition: ComponentDefinition = DEFINITION): ComponentBaseContext {
  return {
    nodes: [node],
    node,
    componentDefinition: definition,
    lastExecutions: [],
    currentUser: undefined,
    actions: {
      invokeNodeExecutionHook: async () => undefined,
    },
  };
}

describe("Cloudflare monitor component mappers", () => {
  it("shows create monitor configuration metadata", () => {
    const node: NodeInfo = {
      id: "node-1",
      name: "Create Cloudflare monitor",
      componentName: "cloudflare.createMonitor",
      isCollapsed: false,
      configuration: {
        description: "Login monitor",
        type: "http",
        path: "/health",
        port: 80,
        pool: "pool123",
        advanced: {
          expectedCodes: "2xx",
        },
      },
    };

    expect(createMonitorMapper.props(baseContext(node)).metadata).toEqual([
      { icon: "activity", label: "Login monitor" },
      { icon: "radio", label: "HTTP" },
      { icon: "link", label: "/health · Port 80" },
      { icon: "server", label: "Pool: pool123" },
      { icon: "settings", label: "Advanced health check settings" },
    ]);
  });

  it("shows delete monitor configuration metadata using resolved description when present", () => {
    const node: NodeInfo = {
      id: "node-2",
      name: "Delete Cloudflare monitor",
      componentName: "cloudflare.deleteMonitor",
      isCollapsed: false,
      configuration: {
        monitor: "monitor123",
        force: true,
      },
      metadata: {
        monitorId: "monitor123",
        monitorDescription: "Edge health",
      },
    };

    expect(
      deleteMonitorMapper.props(
        baseContext(node, {
          ...DEFINITION,
          name: "cloudflare.deleteMonitor",
          label: "Delete Monitor",
          icon: "trash-2",
        }),
      ).metadata,
    ).toEqual([
      { icon: "trash-2", label: "Edge health" },
      { icon: "shield-alert", label: "Force delete" },
    ]);
  });

  it("falls back to configured monitor id when node metadata is absent or stale", () => {
    const node: NodeInfo = {
      id: "node-2",
      name: "Delete Cloudflare monitor",
      componentName: "cloudflare.deleteMonitor",
      isCollapsed: false,
      configuration: {
        monitor: "monitor123",
        force: true,
      },
    };

    expect(
      deleteMonitorMapper.props(
        baseContext(node, {
          ...DEFINITION,
          name: "cloudflare.deleteMonitor",
          label: "Delete Monitor",
          icon: "trash-2",
        }),
      ).metadata,
    ).toEqual([
      { icon: "trash-2", label: "monitor123" },
      { icon: "shield-alert", label: "Force delete" },
    ]);
  });
});
