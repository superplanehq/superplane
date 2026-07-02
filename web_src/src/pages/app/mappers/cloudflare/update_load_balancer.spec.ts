import { describe, expect, it } from "vitest";

import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
} from "../types";
import { eventStateRegistry } from "./index";
import { updateLoadBalancerMapper } from "./update_load_balancer";

// ── Helpers ───────────────────────────────────────────────────────────────────

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test Node",
    componentName: "cloudflare.updateLoadBalancer",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildOutput(data: unknown): OutputPayload {
  return {
    type: "cloudflare.loadBalancer",
    timestamp: new Date().toISOString(),
    data,
  };
}

function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  return {
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
    ...overrides,
  };
}

function buildDetailsCtx(overrides?: {
  node?: Partial<NodeInfo>;
  execution?: Partial<ExecutionInfo>;
}): ExecutionDetailsContext {
  const node = buildNode(overrides?.node);
  return { nodes: [node], node, execution: buildExecution(overrides?.execution) };
}

const defaultDefinition: ComponentDefinition = {
  name: "cloudflare.updateLoadBalancer",
  label: "Update Load Balancer",
  description: "",
  icon: "cloud",
  color: "orange",
};

function buildPropsContext(overrides?: Partial<ComponentBaseContext>): ComponentBaseContext {
  return {
    nodes: [],
    node: buildNode(),
    componentDefinition: defaultDefinition,
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
    ...overrides,
  };
}

const lbOutputData = {
  loadBalancer: {
    id: "lb123",
    name: "my-lb",
    description: "Primary load balancer",
    enabled: true,
    proxied: true,
    steering_policy: "random",
    session_affinity: "cookie",
    default_pools: ["pool1", "pool2"],
  },
};

// ── getExecutionDetails ───────────────────────────────────────────────────────

describe("updateLoadBalancerMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => updateLoadBalancerMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => updateLoadBalancerMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when output data has no loadBalancer key", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput({})] } } });
    expect(() => updateLoadBalancerMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("extracts load balancer fields from output", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(lbOutputData)] } },
    });
    const details = updateLoadBalancerMapper.getExecutionDetails(ctx);
    expect(details["Name"]).toBe("my-lb");
    expect(details["Description"]).toBe("Primary load balancer");
    expect(details["Enabled"]).toBe("true");
    expect(details["Proxied"]).toBe("true");
    expect(details["Default Pools"]).toBe("2");
  });

  it("omits description when it is absent", () => {
    const data = {
      loadBalancer: { id: "lb1", name: "my-lb", enabled: true, steering_policy: "off", default_pools: [] },
    };
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(data)] } },
    });
    expect(updateLoadBalancerMapper.getExecutionDetails(ctx)["Description"]).toBeUndefined();
  });

  it("includes executed at timestamp", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(lbOutputData)] } },
    });
    expect(updateLoadBalancerMapper.getExecutionDetails(ctx)["Executed At"]).toBeDefined();
  });
});

// ── props ─────────────────────────────────────────────────────────────────────

describe("updateLoadBalancerMapper.props", () => {
  it("prefers load balancer name from node metadata", () => {
    const props = updateLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({
          configuration: { loadBalancer: "lb-id", steeringPolicy: "random" },
          metadata: { loadBalancerName: "Resolved Name" },
        }),
      }),
    );
    expect(props.metadata).toContainEqual({ icon: "network", label: "Resolved Name" });
  });

  it("falls back to load balancer id from configuration when metadata is absent", () => {
    const props = updateLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { loadBalancer: "lb-id" }, metadata: {} }),
      }),
    );
    expect(props.metadata).toEqual([{ icon: "network", label: "lb-id" }]);
  });

  it("shows description in metadata when set", () => {
    const props = updateLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { loadBalancer: "lb-id", description: "My LB" }, metadata: {} }),
      }),
    );
    expect(props.metadata).toContainEqual({ icon: "text", label: "My LB" });
  });

  it("omits description when not set", () => {
    const props = updateLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { loadBalancer: "lb-id" }, metadata: {} }),
      }),
    );
    expect(props.metadata?.find((m) => m.icon === "text")).toBeUndefined();
  });

  it("shows steering policy in metadata when set", () => {
    const props = updateLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { loadBalancer: "lb-id", steeringPolicy: "geo" }, metadata: {} }),
      }),
    );
    expect(props.metadata).toContainEqual({ icon: "git-branch", label: "geo" });
  });

  it("omits steering policy when not set", () => {
    const props = updateLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { loadBalancer: "lb-id" }, metadata: {} }),
      }),
    );
    const labels = props.metadata?.map((m) => m.label) ?? [];
    expect(labels).not.toContain("geo");
    expect(props.metadata).toHaveLength(1);
  });

  it("shows default pools count in metadata when set", () => {
    const props = updateLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { loadBalancer: "lb-id", defaultPools: ["pool1", "pool2"] }, metadata: {} }),
      }),
    );
    expect(props.metadata).toContainEqual({ icon: "layers", label: "2 pools" });
  });

  it("shows singular pool label when only one default pool", () => {
    const props = updateLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { loadBalancer: "lb-id", defaultPools: ["pool1"] }, metadata: {} }),
      }),
    );
    expect(props.metadata).toContainEqual({ icon: "layers", label: "1 pool" });
  });

  it("omits pools item when defaultPools is empty", () => {
    const props = updateLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { loadBalancer: "lb-id", defaultPools: [] }, metadata: {} }),
      }),
    );
    expect(props.metadata?.find((m) => m.icon === "layers")).toBeUndefined();
  });

  it("shows enabled state in metadata when set", () => {
    const props = updateLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { loadBalancer: "lb-id", enabled: true }, metadata: {} }),
      }),
    );
    expect(props.metadata).toContainEqual({ icon: "check-circle", label: "Enabled" });
  });

  it("shows disabled state in metadata when enabled is false", () => {
    const props = updateLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { loadBalancer: "lb-id", enabled: false }, metadata: {} }),
      }),
    );
    expect(props.metadata).toContainEqual({ icon: "circle", label: "Disabled" });
  });

  it("omits enabled item when enabled is not set", () => {
    const props = updateLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { loadBalancer: "lb-id" }, metadata: {} }),
      }),
    );
    expect(props.metadata?.find((m) => m.icon === "check-circle" || m.icon === "circle")).toBeUndefined();
  });

  it("returns empty metadata when configuration is empty", () => {
    const props = updateLoadBalancerMapper.props(
      buildPropsContext({ node: buildNode({ configuration: {}, metadata: {} }) }),
    );
    expect(props.metadata).toEqual([]);
  });
});

// ── eventStateRegistry ────────────────────────────────────────────────────────

describe("eventStateRegistry.updateLoadBalancer", () => {
  it("maps finished success to updated", () => {
    expect(eventStateRegistry.updateLoadBalancer.getState(buildExecution())).toBe("updated");
  });

  it("returns running when execution is in progress", () => {
    const running = buildExecution({
      state: "STATE_STARTED",
      result: "RESULT_UNSPECIFIED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_UNSPECIFIED" as ExecutionInfo["resultReason"],
    });
    expect(eventStateRegistry.updateLoadBalancer.getState(running)).toBe("running");
  });

  it("returns failed when execution fails", () => {
    const failed = buildExecution({
      state: "STATE_FINISHED",
      result: "RESULT_FAILED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_COMPONENT_FAILED" as ExecutionInfo["resultReason"],
    });
    expect(eventStateRegistry.updateLoadBalancer.getState(failed)).toBe("failed");
  });
});
