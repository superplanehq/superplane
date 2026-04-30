import { describe, expect, it } from "vitest";

import { deleteApplicationMapper } from "./delete_application";
import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Delete Application",
    componentName: "oci.deleteApplication",
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
      name: "oci.deleteApplication",
      label: "Delete Application",
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

describe("deleteApplicationMapper.props", () => {
  it("shows applicationName from node metadata", () => {
    const props = deleteApplicationMapper.props(buildComponentCtx({ metadata: { applicationName: "my-app" } }));
    expect(props.metadata).toEqual(
      expect.arrayContaining([expect.objectContaining({ icon: "trash-2", label: "my-app" })]),
    );
  });

  it("falls back to application from configuration when applicationName is absent", () => {
    const props = deleteApplicationMapper.props(
      buildComponentCtx({ configuration: { application: "ocid1.fnapp.xxx" }, metadata: {} }),
    );
    expect(props.metadata).toEqual(
      expect.arrayContaining([expect.objectContaining({ icon: "trash-2", label: "ocid1.fnapp.xxx" })]),
    );
  });

  it("produces empty metadata when both applicationName and application are absent", () => {
    const props = deleteApplicationMapper.props(buildComponentCtx());
    expect(props.metadata).toEqual([]);
  });
});

// ── getExecutionDetails ────────────────────────────────────────────────

describe("deleteApplicationMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => deleteApplicationMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default output array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => deleteApplicationMapper.getExecutionDetails(ctx)).not.toThrow();
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
    const details = deleteApplicationMapper.getExecutionDetails(ctx);
    expect(new Date(details["Executed At"]).getTime()).toBe(new Date(startedAt).getTime());
  });

  it("falls back to execution.createdAt for Executed At when metadata.startedAt is absent", () => {
    const createdAt = new Date("2026-01-01T09:00:00Z").toISOString();
    const ctx = buildDetailsCtx({ execution: { createdAt, metadata: {}, outputs: undefined } });
    const details = deleteApplicationMapper.getExecutionDetails(ctx);
    expect(new Date(details["Executed At"]).getTime()).toBe(new Date(createdAt).getTime());
  });

  it("maps output fields to display labels", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              applicationId: "ocid1.fnapp.xxx",
              displayName: "my-app",
              deleted: true,
            }),
          ],
        },
      },
    });

    const details = deleteApplicationMapper.getExecutionDetails(ctx);
    expect(details["Display Name"]).toBe("my-app");
    expect(details["Deleted"]).toBe("true");
  });

  it("omits optional fields that are absent from output", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [buildOutput({ applicationId: "ocid1.fnapp.xxx", displayName: "my-app" })],
        },
      },
    });

    const details = deleteApplicationMapper.getExecutionDetails(ctx);
    expect(details["Display Name"]).toBe("my-app");
    expect(details["Deleted"]).toBeUndefined();
  });
});
