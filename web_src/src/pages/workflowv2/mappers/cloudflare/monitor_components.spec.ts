import { describe, expect, it } from "vitest";
import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
} from "../types";
import { createMonitorMapper } from "./create_monitor";
import { deleteMonitorMapper } from "./delete_monitor";

const DEFINITION: ComponentDefinition = {
  name: "cloudflare.createMonitor",
  label: "Create Monitor",
  description: "",
  icon: "activity",
  color: "orange",
};

function baseContext(
  node: NodeInfo,
  definition: ComponentDefinition = DEFINITION,
  lastExecutions: ExecutionInfo[] = [],
): ComponentBaseContext {
  return {
    nodes: [node],
    node,
    componentDefinition: definition,
    lastExecutions,
    currentUser: undefined,
    actions: {
      invokeNodeExecutionHook: async () => undefined,
    },
  };
}

const DELETE_MONITOR_DEFINITION: ComponentDefinition = {
  ...DEFINITION,
  name: "cloudflare.deleteMonitor",
  label: "Delete Monitor",
  icon: "trash-2",
};

function buildDeleteMonitorOutputPayload(data: unknown): OutputPayload {
  return {
    type: "cloudflare.monitor.deleted",
    timestamp: new Date().toISOString(),
    data,
  };
}

function buildDeleteMonitorDetailsCtx(overrides?: {
  node?: Partial<NodeInfo>;
  execution?: Partial<ExecutionInfo>;
}): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-del",
    name: "Delete Cloudflare monitor",
    componentName: "cloudflare.deleteMonitor",
    isCollapsed: false,
    configuration: { monitor: "monitor123" },
    ...overrides?.node,
  };
  const execution: ExecutionInfo = {
    id: "exec-1",
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    resultReason: "RESULT_REASON_OK",
    resultMessage: "",
    metadata: {},
    configuration: {},
    rootEvent: undefined,
    ...overrides?.execution,
  };
  return { nodes: [node], node, execution };
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
      metadata: {
        poolName: "Production",
      },
    };

    expect(createMonitorMapper.props(baseContext(node)).metadata).toEqual([
      { icon: "activity", label: "Login monitor" },
      { icon: "radio", label: "HTTP" },
      { icon: "link", label: "/health · Port 80" },
      { icon: "server", label: "Pool: Production" },
      { icon: "settings", label: "Advanced health check settings" },
    ]);
  });

  it("reads resolved pool name when metadata uses snake_case keys", () => {
    const node: NodeInfo = {
      id: "node-pool-snake",
      name: "Create Cloudflare monitor",
      componentName: "cloudflare.createMonitor",
      isCollapsed: false,
      configuration: {
        description: "Probe",
        type: "tcp",
        port: 443,
        pool: "501ca97551554e91623f1bcfecb6deee",
      },
      metadata: {
        pool_name: "Production pool",
      },
    };

    expect(createMonitorMapper.props(baseContext(node)).metadata).toContainEqual({
      icon: "server",
      label: "Pool: Production pool",
    });
  });

  it("falls back to pool id in metadata when node metadata has no pool name", () => {
    const node: NodeInfo = {
      id: "node-pool-id-only",
      name: "Create Cloudflare monitor",
      componentName: "cloudflare.createMonitor",
      isCollapsed: false,
      configuration: {
        description: "M",
        type: "tcp",
        port: 443,
        pool: "501ca97551554e91623f1bcfecb6deee",
      },
    };

    expect(createMonitorMapper.props(baseContext(node)).metadata).toContainEqual({
      icon: "server",
      label: "Pool: 501ca97551554e91623f1bcfecb6deee",
    });
  });

  it("uses pool name from latest create monitor execution output when node metadata is absent", () => {
    const node: NodeInfo = {
      id: "node-pool-output",
      name: "Create Cloudflare monitor",
      componentName: "cloudflare.createMonitor",
      isCollapsed: false,
      configuration: {
        description: "M",
        type: "tcp",
        port: 443,
        pool: "501ca97551554e91623f1bcfecb6deee",
      },
    };

    const execution = {
      outputs: {
        default: [
          {
            data: {
              poolId: "501ca97551554e91623f1bcfecb6deee",
              pool: { id: "501ca97551554e91623f1bcfecb6deee", name: "Production pool" },
            },
          },
        ],
      },
    } as unknown as ExecutionInfo;

    expect(createMonitorMapper.props(baseContext(node, DEFINITION, [execution])).metadata).toContainEqual({
      icon: "server",
      label: "Pool: Production pool",
    });
  });

  it("shows advanced badge when advanced.interval is set", () => {
    const node: NodeInfo = {
      id: "node-advanced-interval",
      name: "Create Cloudflare monitor",
      componentName: "cloudflare.createMonitor",
      isCollapsed: false,
      configuration: {
        description: "Probe",
        type: "tcp",
        port: 443,
        advanced: { interval: 120 },
      },
    };

    expect(createMonitorMapper.props(baseContext(node)).metadata).toContainEqual({
      icon: "settings",
      label: "Advanced health check settings",
    });
  });

  it("shows advanced badge for legacy flat timing fields without nested advanced", () => {
    const node: NodeInfo = {
      id: "node-legacy-flat",
      name: "Create Cloudflare monitor",
      componentName: "cloudflare.createMonitor",
      isCollapsed: false,
      configuration: {
        description: "Probe",
        type: "https",
        path: "/",
        port: 443,
        interval: 90,
        retries: 0,
      },
    };

    expect(createMonitorMapper.props(baseContext(node)).metadata).toContainEqual({
      icon: "settings",
      label: "Advanced health check settings",
    });
  });

  it("reads monitor description from snake_case metadata keys", () => {
    const node: NodeInfo = {
      id: "node-monitor-snake",
      name: "Delete Cloudflare monitor",
      componentName: "cloudflare.deleteMonitor",
      isCollapsed: false,
      configuration: {
        monitor: "monitor123",
      },
      metadata: {
        monitor_id: "monitor123",
        monitor_description: "Edge health",
      },
    };

    expect(deleteMonitorMapper.props(baseContext(node, DELETE_MONITOR_DEFINITION)).metadata).toContainEqual({
      icon: "trash-2",
      label: "Edge health",
    });
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

    expect(deleteMonitorMapper.props(baseContext(node, DELETE_MONITOR_DEFINITION)).metadata).toEqual([
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
          ...DELETE_MONITOR_DEFINITION,
        }),
      ).metadata,
    ).toEqual([
      { icon: "trash-2", label: "monitor123" },
      { icon: "shield-alert", label: "Force delete" },
    ]);
  });
});

describe("deleteMonitorMapper.getExecutionDetails", () => {
  it("shows resolved monitor description under Monitor when metadata matches output id", () => {
    const ctx = buildDeleteMonitorDetailsCtx({
      node: {
        metadata: { monitorId: "monitor123", monitorDescription: "Edge health" },
        configuration: { monitor: "monitor123" },
      },
      execution: {
        outputs: {
          default: [
            buildDeleteMonitorOutputPayload({
              accountId: "acc",
              monitorId: "monitor123",
              deleted: true,
              references: [],
            }),
          ],
        },
      },
    });
    const details = deleteMonitorMapper.getExecutionDetails(ctx);
    expect(details["Monitor"]).toBe("Edge health");
    expect(details["Monitor ID"]).toBeUndefined();
    expect(details["Deleted"]).toBe("Yes");
    expect(details["References"]).toBe("0");
  });

  it("falls back to monitor id in execution details when metadata is absent", () => {
    const ctx = buildDeleteMonitorDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildDeleteMonitorOutputPayload({
              monitorId: "monitor-xyz",
              deleted: true,
            }),
          ],
        },
      },
    });
    expect(deleteMonitorMapper.getExecutionDetails(ctx)["Monitor"]).toBe("monitor-xyz");
  });
});
