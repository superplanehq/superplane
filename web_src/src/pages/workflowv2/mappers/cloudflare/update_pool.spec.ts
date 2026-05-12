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
import { updatePoolMapper } from "./update_pool";

// ── Helpers ───────────────────────────────────────────────────────────────────

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test Node",
    componentName: "cloudflare.updatePool",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildOutput(data: unknown): OutputPayload {
  return {
    type: "cloudflare.pool",
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
  name: "cloudflare.updatePool",
  label: "Update Pool",
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

const poolOutputData = {
  pool: {
    id: "pool123",
    name: "primary-pool",
    description: "Primary origin pool",
    enabled: true,
    minimum_origins: 1,
    origins: [
      { name: "origin-1", address: "1.2.3.4", enabled: true, weight: 1 },
      { name: "origin-2", address: "5.6.7.8", enabled: true, weight: 1 },
    ],
  },
  accountId: "acc123",
};

// ── getExecutionDetails ───────────────────────────────────────────────────────

describe("updatePoolMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => updatePoolMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => updatePoolMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when output data has no pool key", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput({})] } } });
    expect(() => updatePoolMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("extracts pool fields from output", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(poolOutputData)] } },
    });
    const details = updatePoolMapper.getExecutionDetails(ctx);
    expect(details["Pool ID"]).toBe("pool123");
    expect(details["Name"]).toBe("primary-pool");
    expect(details["Description"]).toBe("Primary origin pool");
    expect(details["Enabled"]).toBe("true");
    expect(details["Minimum Origins"]).toBe("1");
    expect(details["Number of Origins"]).toBe("2");
  });

  it("includes executed at timestamp", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(poolOutputData)] } },
    });
    expect(updatePoolMapper.getExecutionDetails(ctx)["Executed At"]).toBeDefined();
  });
});

// ── props ─────────────────────────────────────────────────────────────────────

describe("updatePoolMapper.props", () => {
  it("shows pool name from configuration when set", () => {
    const props = updatePoolMapper.props(
      buildPropsContext({ node: buildNode({ configuration: { name: "my-pool", pool: "pool-id" } }) }),
    );
    expect(props.metadata).toEqual([{ icon: "network", label: "my-pool" }]);
  });

  it("falls back to pool id when name is absent", () => {
    const props = updatePoolMapper.props(
      buildPropsContext({ node: buildNode({ configuration: { pool: "pool-id" } }) }),
    );
    expect(props.metadata).toEqual([{ icon: "network", label: "pool-id" }]);
  });

  it("returns empty metadata when configuration is empty", () => {
    const props = updatePoolMapper.props(buildPropsContext({ node: buildNode({ configuration: {} }) }));
    expect(props.metadata).toEqual([]);
  });
});

// ── eventStateRegistry ────────────────────────────────────────────────────────

describe("eventStateRegistry.updatePool", () => {
  it("maps finished success to updated", () => {
    expect(eventStateRegistry.updatePool.getState(buildExecution())).toBe("updated");
  });

  it("returns running when execution is in progress", () => {
    const running = buildExecution({
      state: "STATE_STARTED",
      result: "RESULT_UNSPECIFIED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_UNSPECIFIED" as ExecutionInfo["resultReason"],
    });
    expect(eventStateRegistry.updatePool.getState(running)).toBe("running");
  });

  it("returns failed when execution fails", () => {
    const failed = buildExecution({
      state: "STATE_FINISHED",
      result: "RESULT_FAILED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_COMPONENT_FAILED" as ExecutionInfo["resultReason"],
    });
    expect(eventStateRegistry.updatePool.getState(failed)).toBe("failed");
  });
});
