import { describe, expect, it } from "vitest";

import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
} from "../types";
import { deleteKVValueMapper } from "./delete_kv_value";
import { eventStateRegistry } from "./index";

// ── Helpers ───────────────────────────────────────────────────────────────────

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Test Node",
    componentName: "cloudflare.deleteKVValue",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildOutput(data: unknown): OutputPayload {
  return {
    type: "cloudflare.kv.value.deleted",
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
  name: "cloudflare.deleteKVValue",
  label: "Delete KV Value",
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

describe("deleteKVValueMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => deleteKVValueMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => deleteKVValueMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when output data is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput({})] } } });
    expect(() => deleteKVValueMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("extracts namespace id, key, and deleted status from output", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [buildOutput({ accountId: "acc123", namespaceId: "ns123", key: "my-key", deleted: true })],
        },
      },
    });
    const details = deleteKVValueMapper.getExecutionDetails(ctx);
    expect(details["Namespace ID"]).toBeUndefined();
    expect(details["Key"]).toBe("my-key");
    expect(details["Deleted"]).toBe("true");
  });

  it("uses dash placeholders when result fields are missing", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({})] } },
    });
    const details = deleteKVValueMapper.getExecutionDetails(ctx);
    expect(details["Namespace ID"]).toBeUndefined();
    expect(details["Key"]).toBe("-");
    expect(details["Deleted"]).toBe("-");
  });

  it("includes executed at timestamp", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: { default: [buildOutput({ namespaceId: "ns123", key: "my-key", deleted: true })] },
      },
    });
    expect(deleteKVValueMapper.getExecutionDetails(ctx)["Executed At"]).toBeDefined();
  });
});

// ── props ─────────────────────────────────────────────────────────────────────

describe("deleteKVValueMapper.props", () => {
  it("shows namespaceName and keyName from node metadata", () => {
    const props = deleteKVValueMapper.props(
      buildPropsContext({
        node: buildNode({
          metadata: { namespaceName: "My Namespace", keyName: "my-key" },
          configuration: { namespace: "ns123", kvKey: "my-key" },
        }),
      }),
    );
    expect(props.metadata).toEqual([
      { icon: "database", label: "My Namespace" },
      { icon: "key", label: "my-key" },
    ]);
  });

  it("falls back to config ids when metadata names are absent", () => {
    const props = deleteKVValueMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { namespace: "ns123", kvKey: "my-key" } }),
      }),
    );
    expect(props.metadata).toEqual([
      { icon: "database", label: "ns123" },
      { icon: "key", label: "my-key" },
    ]);
  });

  it("returns empty metadata when configuration is empty", () => {
    const props = deleteKVValueMapper.props(buildPropsContext({ node: buildNode({ configuration: {} }) }));
    expect(props.metadata).toEqual([]);
  });
});

// ── eventStateRegistry ────────────────────────────────────────────────────────

describe("eventStateRegistry.deleteKVValue", () => {
  it("maps finished success to deleted", () => {
    expect(eventStateRegistry.deleteKVValue.getState(buildExecution())).toBe("deleted");
  });

  it("returns running when execution is in progress", () => {
    const running = buildExecution({
      state: "STATE_STARTED",
      result: "RESULT_UNSPECIFIED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_UNSPECIFIED" as ExecutionInfo["resultReason"],
    });
    expect(eventStateRegistry.deleteKVValue.getState(running)).toBe("running");
  });

  it("returns failed when execution fails", () => {
    const failed = buildExecution({
      state: "STATE_FINISHED",
      result: "RESULT_FAILED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_COMPONENT_FAILED" as ExecutionInfo["resultReason"],
    });
    expect(eventStateRegistry.deleteKVValue.getState(failed)).toBe("failed");
  });
});
