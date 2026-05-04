import { describe, expect, it } from "vitest";

import { deleteFunctionMapper } from "./delete_function";
import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Delete Function",
    componentName: "oci.deleteFunction",
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
      name: "oci.deleteFunction",
      label: "Delete Function",
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

describe("deleteFunctionMapper.props", () => {
  it("shows applicationName from node metadata", () => {
    const props = deleteFunctionMapper.props(buildComponentCtx({ metadata: { applicationName: "my-app" } }));
    expect(props.metadata).toEqual(
      expect.arrayContaining([expect.objectContaining({ icon: "layout-grid", label: "my-app" })]),
    );
  });

  it("falls back to applicationId when applicationName is absent", () => {
    const props = deleteFunctionMapper.props(buildComponentCtx({ metadata: { applicationId: "ocid1.fnapp.xxx" } }));
    expect(props.metadata).toEqual(
      expect.arrayContaining([expect.objectContaining({ icon: "layout-grid", label: "ocid1.fnapp.xxx" })]),
    );
  });

  it("shows functionName from node metadata", () => {
    const props = deleteFunctionMapper.props(buildComponentCtx({ metadata: { functionName: "my-fn" } }));
    expect(props.metadata).toEqual(expect.arrayContaining([expect.objectContaining({ icon: "zap", label: "my-fn" })]));
  });

  it("falls back to functionId when functionName is absent", () => {
    const props = deleteFunctionMapper.props(buildComponentCtx({ metadata: { functionId: "ocid1.fnfunc.xxx" } }));
    expect(props.metadata).toEqual(
      expect.arrayContaining([expect.objectContaining({ icon: "zap", label: "ocid1.fnfunc.xxx" })]),
    );
  });

  it("produces empty metadata when node metadata is empty", () => {
    const props = deleteFunctionMapper.props(buildComponentCtx());
    expect(props.metadata).toEqual([]);
  });
});

// ── getExecutionDetails ────────────────────────────────────────────────

describe("deleteFunctionMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => deleteFunctionMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default output array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => deleteFunctionMapper.getExecutionDetails(ctx)).not.toThrow();
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
    const details = deleteFunctionMapper.getExecutionDetails(ctx);
    expect(new Date(details["Executed At"]).getTime()).toBe(new Date(startedAt).getTime());
  });

  it("falls back to execution.createdAt for Executed At when metadata.startedAt is absent", () => {
    const createdAt = new Date("2026-01-01T09:00:00Z").toISOString();
    const ctx = buildDetailsCtx({ execution: { createdAt, metadata: {}, outputs: undefined } });
    const details = deleteFunctionMapper.getExecutionDetails(ctx);
    expect(new Date(details["Executed At"]).getTime()).toBe(new Date(createdAt).getTime());
  });

  it("maps output fields to display labels", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              functionId: "ocid1.fnfunc.xxx",
              deleted: true,
            }),
          ],
        },
      },
    });

    const details = deleteFunctionMapper.getExecutionDetails(ctx);
    expect(details["Function ID"]).toBe("ocid1.fnfunc.xxx");
    expect(details["Deleted"]).toBe("true");
  });

  it("omits optional fields that are absent from output", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [buildOutput({ functionId: "ocid1.fnfunc.xxx" })],
        },
      },
    });

    const details = deleteFunctionMapper.getExecutionDetails(ctx);
    expect(details["Function ID"]).toBe("ocid1.fnfunc.xxx");
    expect(details["Deleted"]).toBeUndefined();
  });
});
