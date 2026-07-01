import { describe, expect, it } from "vitest";

import { invokeFunctionMapper } from "./invoke_function";
import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Invoke Function",
    componentName: "oci.invokeFunction",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  return {
    id: "exec-1",
    createdAt: new Date("2026-01-01T00:00:00Z").toISOString(),
    updatedAt: new Date("2026-01-01T00:01:00Z").toISOString(),
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

function buildOutput(data: Record<string, unknown>) {
  return { type: "json", timestamp: new Date().toISOString(), data };
}

function buildDetailsCtx(overrides?: {
  node?: Partial<NodeInfo>;
  execution?: Partial<ExecutionInfo>;
}): ExecutionDetailsContext {
  const node = buildNode(overrides?.node);
  return { nodes: [node], node, execution: buildExecution(overrides?.execution) };
}

function buildComponentCtx(nodeOverrides?: Partial<NodeInfo>): ComponentBaseContext {
  const node = buildNode(nodeOverrides);
  return {
    nodes: [node],
    node,
    componentDefinition: {
      name: "oci.invokeFunction",
      label: "Invoke Function",
      description: "",
      icon: "oci",
      color: "red",
    },
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

// ── props / metadata list ──────────────────────────────────────────────

describe("invokeFunctionMapper.props", () => {
  it("shows applicationName from node metadata over raw application", () => {
    const props = invokeFunctionMapper.props(
      buildComponentCtx({
        configuration: { application: "ocid1.fnapp.xxx" },
        metadata: { applicationName: "my-app" },
      }),
    );
    expect(props.metadata).toEqual(
      expect.arrayContaining([expect.objectContaining({ icon: "layout-grid", label: "my-app" })]),
    );
  });

  it("falls back to application when applicationName is absent", () => {
    const props = invokeFunctionMapper.props(
      buildComponentCtx({ configuration: { application: "ocid1.fnapp.xxx" }, metadata: {} }),
    );
    expect(props.metadata).toEqual(
      expect.arrayContaining([expect.objectContaining({ icon: "layout-grid", label: "ocid1.fnapp.xxx" })]),
    );
  });

  it("shows functionName from node metadata over raw function", () => {
    const props = invokeFunctionMapper.props(
      buildComponentCtx({
        configuration: { function: "ocid1.fnfunc.xxx" },
        metadata: { functionName: "my-fn" },
      }),
    );
    expect(props.metadata).toEqual(expect.arrayContaining([expect.objectContaining({ icon: "zap", label: "my-fn" })]));
  });

  it("falls back to function when functionName is absent", () => {
    const props = invokeFunctionMapper.props(
      buildComponentCtx({ configuration: { function: "ocid1.fnfunc.xxx" }, metadata: {} }),
    );
    expect(props.metadata).toEqual(
      expect.arrayContaining([expect.objectContaining({ icon: "zap", label: "ocid1.fnfunc.xxx" })]),
    );
  });

  it("produces empty metadata when configuration and metadata are empty", () => {
    const props = invokeFunctionMapper.props(buildComponentCtx());
    expect(props.metadata).toEqual([]);
  });
});

// ── getExecutionDetails ────────────────────────────────────────────────

describe("invokeFunctionMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => invokeFunctionMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default output array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => invokeFunctionMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("uses metadata.startedAt for Executed At when present", () => {
    const startedAt = new Date("2026-01-01T08:00:00Z").toISOString();
    const ctx = buildDetailsCtx({
      execution: {
        createdAt: new Date("2026-01-01T09:00:00Z").toISOString(),
        metadata: { startedAt },
        outputs: undefined,
      },
    });
    const details = invokeFunctionMapper.getExecutionDetails(ctx);
    expect(new Date(details["Executed At"]).getTime()).toBe(new Date(startedAt).getTime());
  });

  it("falls back to execution.createdAt for Executed At when metadata.startedAt is absent", () => {
    const createdAt = new Date("2026-01-01T09:00:00Z").toISOString();
    const ctx = buildDetailsCtx({ execution: { createdAt, metadata: {}, outputs: undefined } });
    const details = invokeFunctionMapper.getExecutionDetails(ctx);
    expect(new Date(details["Executed At"]).getTime()).toBe(new Date(createdAt).getTime());
  });

  it("maps output fields to display labels", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              functionId: "ocid1.fnfunc.xxx",
              statusCode: 200,
              response: '{"result":"ok"}',
            }),
          ],
        },
      },
    });

    const details = invokeFunctionMapper.getExecutionDetails(ctx);
    expect(details["Status Code"]).toBe("200");
    expect(details["Function ID"]).toBe("ocid1.fnfunc.xxx");
    expect(details["Response"]).toBe('{"result":"ok"}');
  });

  it("omits optional fields that are absent from output", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [buildOutput({ statusCode: 200 })],
        },
      },
    });

    const details = invokeFunctionMapper.getExecutionDetails(ctx);
    expect(details["Status Code"]).toBe("200");
    expect(details["Function ID"]).toBeUndefined();
    expect(details["Response"]).toBeUndefined();
  });
});
