import { describe, expect, it } from "vitest";

import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
} from "../types";
import { getTunnelMapper } from "./get_tunnel";
import { eventStateRegistry } from "./index";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test Node",
    componentName: "cloudflare.getTunnel",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildOutput(data: unknown): OutputPayload {
  return {
    type: "cloudflare.tunnel.fetched",
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
  name: "cloudflare.getTunnel",
  label: "Get Tunnel",
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

const tunnelOutputData = {
  tunnel: {
    id: "tun123",
    name: "edge-tunnel",
    status: "healthy",
    config_src: "cloudflare",
  },
  accountId: "acc123",
};

describe("getTunnelMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => getTunnelMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("includes tunnel fields from output", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: { default: [buildOutput(tunnelOutputData)] },
      },
    });
    const details = getTunnelMapper.getExecutionDetails(ctx);
    expect(details["Tunnel ID"]).toBeUndefined();
    expect(details["Name"]).toBe("edge-tunnel");
    expect(details["Status"]).toBe("healthy");
  });
});

describe("eventStateRegistry.getTunnel", () => {
  it("maps finished passed to fetched", () => {
    expect(eventStateRegistry.getTunnel.getState(buildExecution())).toBe("fetched");
  });
});

describe("getTunnelMapper.props", () => {
  it("returns props without throwing", () => {
    const props = getTunnelMapper.props(buildPropsContext({}));
    expect(props.title).toBeDefined();
  });
});
