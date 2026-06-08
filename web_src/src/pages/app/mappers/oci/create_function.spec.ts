import { describe, expect, it } from "vitest";

import { createFunctionMapper } from "./create_function";
import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Create Function",
    componentName: "oci.createFunction",
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
      name: "oci.createFunction",
      label: "Create Function",
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

describe("createFunctionMapper.props", () => {
  it("includes displayName in metadata", () => {
    const props = createFunctionMapper.props(buildComponentCtx({ configuration: { displayName: "my-fn" } }));
    expect(props.metadata).toEqual(expect.arrayContaining([expect.objectContaining({ icon: "tag", label: "my-fn" })]));
  });

  it("shows applicationName from node metadata over raw application", () => {
    const props = createFunctionMapper.props(
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
    const props = createFunctionMapper.props(
      buildComponentCtx({ configuration: { application: "ocid1.fnapp.xxx" }, metadata: {} }),
    );
    expect(props.metadata).toEqual(
      expect.arrayContaining([expect.objectContaining({ icon: "layout-grid", label: "ocid1.fnapp.xxx" })]),
    );
  });

  it("includes image in metadata", () => {
    const props = createFunctionMapper.props(
      buildComponentCtx({ configuration: { image: "fra.ocir.io/ns/repo:latest" } }),
    );
    expect(props.metadata).toEqual(
      expect.arrayContaining([expect.objectContaining({ icon: "box", label: "fra.ocir.io/ns/repo:latest" })]),
    );
  });

  it("produces empty metadata when configuration and metadata are empty", () => {
    const props = createFunctionMapper.props(buildComponentCtx());
    expect(props.metadata).toEqual([]);
  });
});

// ── getExecutionDetails ────────────────────────────────────────────────

describe("createFunctionMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => createFunctionMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default output array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => createFunctionMapper.getExecutionDetails(ctx)).not.toThrow();
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
    const details = createFunctionMapper.getExecutionDetails(ctx);
    expect(new Date(details["Executed At"]).getTime()).toBe(new Date(startedAt).getTime());
  });

  it("falls back to execution.createdAt for Executed At when metadata.startedAt is absent", () => {
    const createdAt = new Date("2026-01-01T09:00:00Z").toISOString();
    const ctx = buildDetailsCtx({ execution: { createdAt, metadata: {}, outputs: undefined } });
    const details = createFunctionMapper.getExecutionDetails(ctx);
    expect(new Date(details["Executed At"]).getTime()).toBe(new Date(createdAt).getTime());
  });

  it("maps output fields to display labels", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              displayName: "my-fn",
              image: "fra.ocir.io/ns/repo:latest",
              memoryInMBs: 256,
              lifecycleState: "ACTIVE",
              invokeEndpoint: "https://aabbcc.call.eu-frankfurt-1.oci.oraclecloud.com",
            }),
          ],
        },
      },
    });

    const details = createFunctionMapper.getExecutionDetails(ctx);
    expect(details["Function Name"]).toBe("my-fn");
    expect(details["Image"]).toBe("fra.ocir.io/ns/repo:latest");
    expect(details["Memory (MB)"]).toBe("256");
    expect(details["State"]).toBe("ACTIVE");
    expect(details["Invoke Endpoint"]).toBe("https://aabbcc.call.eu-frankfurt-1.oci.oraclecloud.com");
  });

  it("omits optional fields that are absent from output", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [buildOutput({ displayName: "my-fn" })],
        },
      },
    });

    const details = createFunctionMapper.getExecutionDetails(ctx);
    expect(details["Function Name"]).toBe("my-fn");
    expect(details["Image"]).toBeUndefined();
    expect(details["Memory (MB)"]).toBeUndefined();
    expect(details["Invoke Endpoint"]).toBeUndefined();
  });
});
