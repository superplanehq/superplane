import { describe, expect, it } from "vitest";

import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
} from "../types";
import { deleteLoadBalancerMapper } from "./delete_load_balancer";
import { eventStateRegistry } from "./index";

// ── Helpers ───────────────────────────────────────────────────────────────────

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test Node",
    componentName: "cloudflare.deleteLoadBalancer",
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
  name: "cloudflare.deleteLoadBalancer",
  label: "Delete Load Balancer",
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

// ── getExecutionDetails ───────────────────────────────────────────────────────

describe("deleteLoadBalancerMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => deleteLoadBalancerMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => deleteLoadBalancerMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when output data is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput({})] } } });
    expect(() => deleteLoadBalancerMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("extracts load balancer id and deleted status from output", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [buildOutput({ loadBalancerId: "lb123", deleted: true })],
        },
      },
    });
    const details = deleteLoadBalancerMapper.getExecutionDetails(ctx);
    expect(details["Deleted"]).toBe("true");
  });

  it("uses dash placeholders when result fields are missing", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({})] } },
    });
    const details = deleteLoadBalancerMapper.getExecutionDetails(ctx);
    expect(details["Deleted"]).toBe("-");
  });

  it("includes executed at timestamp", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: { default: [buildOutput({ loadBalancerId: "lb123", deleted: true })] },
      },
    });
    expect(deleteLoadBalancerMapper.getExecutionDetails(ctx)["Executed At"]).toBeDefined();
  });
});

// ── props ─────────────────────────────────────────────────────────────────────

describe("deleteLoadBalancerMapper.props", () => {
  it("prefers load balancer name from node metadata", () => {
    const props = deleteLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({
          configuration: { loadBalancer: "lb-id" },
          metadata: { loadBalancerName: "My LB" },
        }),
      }),
    );
    expect(props.metadata).toEqual([{ icon: "network", label: "My LB" }]);
  });

  it("falls back to load balancer id from configuration when metadata is absent", () => {
    const props = deleteLoadBalancerMapper.props(
      buildPropsContext({
        node: buildNode({
          configuration: { loadBalancer: "lb-id" },
          metadata: {},
        }),
      }),
    );
    expect(props.metadata).toEqual([{ icon: "network", label: "lb-id" }]);
  });

  it("returns empty metadata when both metadata and configuration are empty", () => {
    const props = deleteLoadBalancerMapper.props(
      buildPropsContext({ node: buildNode({ configuration: {}, metadata: {} }) }),
    );
    expect(props.metadata).toEqual([]);
  });
});

// ── eventStateRegistry ────────────────────────────────────────────────────────

describe("eventStateRegistry.deleteLoadBalancer", () => {
  it("maps finished success to deleted", () => {
    expect(eventStateRegistry.deleteLoadBalancer.getState(buildExecution())).toBe("deleted");
  });

  it("returns running when execution is in progress", () => {
    const running = buildExecution({
      state: "STATE_STARTED",
      result: "RESULT_UNSPECIFIED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_UNSPECIFIED" as ExecutionInfo["resultReason"],
    });
    expect(eventStateRegistry.deleteLoadBalancer.getState(running)).toBe("running");
  });

  it("returns failed when execution fails", () => {
    const failed = buildExecution({
      state: "STATE_FINISHED",
      result: "RESULT_FAILED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_COMPONENT_FAILED" as ExecutionInfo["resultReason"],
    });
    expect(eventStateRegistry.deleteLoadBalancer.getState(failed)).toBe("failed");
  });
});
