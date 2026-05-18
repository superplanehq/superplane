import { describe, expect, it } from "vitest";

import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
} from "../types";
import { createLoadBalancerMapper } from "./create_load_balancer";
import { eventStateRegistry } from "./index";

// ── Helpers ───────────────────────────────────────────────────────────────────

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test Node",
    componentName: "cloudflare.createLoadBalancer",
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
  name: "cloudflare.createLoadBalancer",
  label: "Create Load Balancer",
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

describe("createLoadBalancerMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => createLoadBalancerMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => createLoadBalancerMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when output data has no loadBalancer key", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput({})] } } });
    expect(() => createLoadBalancerMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("extracts load balancer fields from output", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(lbOutputData)] } },
    });
    const details = createLoadBalancerMapper.getExecutionDetails(ctx);
    expect(details["Name"]).toBe("my-lb");
    expect(details["Description"]).toBe("Primary load balancer");
    expect(details["Enabled"]).toBe("true");
    expect(details["Proxied"]).toBe("true");
    expect(details["Default Pools"]).toBe("2");
  });

  it("omits description when it is absent", () => {
    const data = { loadBalancer: { id: "lb1", name: "my-lb", enabled: true, proxied: false, default_pools: [] } };
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(data)] } },
    });
    expect(createLoadBalancerMapper.getExecutionDetails(ctx)["Description"]).toBeUndefined();
  });

  it("includes executed at timestamp", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(lbOutputData)] } },
    });
    expect(createLoadBalancerMapper.getExecutionDetails(ctx)["Executed At"]).toBeDefined();
  });
});

// ── props ─────────────────────────────────────────────────────────────────────

describe("createLoadBalancerMapper.props", () => {
  it("includes lb name from configuration in metadata", () => {
    const props = createLoadBalancerMapper.props(
      buildPropsContext({ node: buildNode({ configuration: { name: "my-lb", enabled: true } }) }),
    );
    expect(props.metadata).toEqual([
      { icon: "network", label: "my-lb" },
      { icon: "check-circle", label: "Enabled" },
    ]);
  });

  it("includes default pools count in metadata", () => {
    const props = createLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { name: "my-lb", defaultPools: ["pool1", "pool2", "pool3"] } }),
      }),
    );
    expect(props.metadata).toContainEqual({ icon: "layers", label: "3 pools" });
  });

  it("shows singular pool label when only one default pool", () => {
    const props = createLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { name: "my-lb", defaultPools: ["pool1"] } }),
      }),
    );
    expect(props.metadata).toContainEqual({ icon: "layers", label: "1 pool" });
  });

  it("omits pools item when defaultPools is empty", () => {
    const props = createLoadBalancerMapper.props(
      buildPropsContext({ node: buildNode({ configuration: { name: "my-lb", defaultPools: [] } }) }),
    );
    expect(props.metadata?.find((m) => m.icon === "layers")).toBeUndefined();
  });

  it("shows disabled icon when enabled is false", () => {
    const props = createLoadBalancerMapper.props(
      buildPropsContext({ node: buildNode({ configuration: { name: "my-lb", enabled: false } }) }),
    );
    expect(props.metadata).toContainEqual({ icon: "circle", label: "Disabled" });
  });

  it("includes both pools and enabled state in metadata", () => {
    const props = createLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { name: "my-lb", defaultPools: ["pool1", "pool2"], enabled: true } }),
      }),
    );
    expect(props.metadata).toEqual([
      { icon: "network", label: "my-lb" },
      { icon: "layers", label: "2 pools" },
      { icon: "check-circle", label: "Enabled" },
    ]);
  });

  it("omits enabled item when enabled is not set", () => {
    const props = createLoadBalancerMapper.props(
      buildPropsContext({ node: buildNode({ configuration: { name: "my-lb" } }) }),
    );
    expect(props.metadata).toEqual([{ icon: "network", label: "my-lb" }]);
  });

  it("returns empty metadata when configuration is empty", () => {
    const props = createLoadBalancerMapper.props(buildPropsContext({ node: buildNode({ configuration: {} }) }));
    expect(props.metadata).toEqual([]);
  });
});

// ── eventStateRegistry ────────────────────────────────────────────────────────

describe("eventStateRegistry.createLoadBalancer", () => {
  it("maps finished success to created", () => {
    expect(eventStateRegistry.createLoadBalancer.getState(buildExecution())).toBe("created");
  });

  it("returns running when execution is in progress", () => {
    const running = buildExecution({
      state: "STATE_STARTED",
      result: "RESULT_UNSPECIFIED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_UNSPECIFIED" as ExecutionInfo["resultReason"],
    });
    expect(eventStateRegistry.createLoadBalancer.getState(running)).toBe("running");
  });

  it("returns failed when execution fails", () => {
    const failed = buildExecution({
      state: "STATE_FINISHED",
      result: "RESULT_FAILED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_COMPONENT_FAILED" as ExecutionInfo["resultReason"],
    });
    expect(eventStateRegistry.createLoadBalancer.getState(failed)).toBe("failed");
  });
});
