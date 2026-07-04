import { describe, expect, it } from "vitest";

import { createApplicationMapper } from "./create_application";
import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Create Application",
    componentName: "oci.createApplication",
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
      name: "oci.createApplication",
      label: "Create Application",
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

describe("createApplicationMapper.props", () => {
  it("includes displayName in metadata", () => {
    const props = createApplicationMapper.props(buildComponentCtx({ configuration: { displayName: "my-app" } }));
    expect(props.metadata).toEqual(expect.arrayContaining([expect.objectContaining({ icon: "tag", label: "my-app" })]));
  });

  it("shows subnetName from node metadata over raw subnet", () => {
    const props = createApplicationMapper.props(
      buildComponentCtx({
        configuration: { subnet: "ocid1.subnet.xxx" },
        metadata: { subnetName: "my-subnet" },
      }),
    );
    expect(props.metadata).toEqual(
      expect.arrayContaining([expect.objectContaining({ icon: "network", label: "my-subnet" })]),
    );
  });

  it("falls back to subnet when subnetName is absent", () => {
    const props = createApplicationMapper.props(
      buildComponentCtx({ configuration: { subnet: "ocid1.subnet.xxx" }, metadata: {} }),
    );
    expect(props.metadata).toEqual(
      expect.arrayContaining([expect.objectContaining({ icon: "network", label: "ocid1.subnet.xxx" })]),
    );
  });

  it("shows shape from node metadata", () => {
    const props = createApplicationMapper.props(buildComponentCtx({ metadata: { shape: "GENERIC_X86" } }));
    expect(props.metadata).toEqual(
      expect.arrayContaining([expect.objectContaining({ icon: "cpu", label: "GENERIC_X86" })]),
    );
  });

  it("produces empty metadata when configuration and metadata are empty", () => {
    const props = createApplicationMapper.props(buildComponentCtx());
    expect(props.metadata).toEqual([]);
  });
});

// ── getExecutionDetails ────────────────────────────────────────────────

describe("createApplicationMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => createApplicationMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default output array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => createApplicationMapper.getExecutionDetails(ctx)).not.toThrow();
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
    const details = createApplicationMapper.getExecutionDetails(ctx);
    expect(new Date(details["Executed At"]).getTime()).toBe(new Date(startedAt).getTime());
  });

  it("falls back to execution.createdAt for Executed At when metadata.startedAt is absent", () => {
    const createdAt = new Date("2026-01-01T09:00:00Z").toISOString();
    const ctx = buildDetailsCtx({ execution: { createdAt, metadata: {}, outputs: undefined } });
    const details = createApplicationMapper.getExecutionDetails(ctx);
    expect(new Date(details["Executed At"]).getTime()).toBe(new Date(createdAt).getTime());
  });

  it("maps output fields to display labels", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              displayName: "my-app",
              lifecycleState: "ACTIVE",
              applicationId: "ocid1.fnapp.xxx",
            }),
          ],
        },
      },
    });

    const details = createApplicationMapper.getExecutionDetails(ctx);
    expect(details["Display Name"]).toBe("my-app");
    expect(details["State"]).toBe("ACTIVE");
    expect(details["Application ID"]).toBe("ocid1.fnapp.xxx");
  });

  it("omits optional fields that are absent from output", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [buildOutput({ displayName: "my-app" })],
        },
      },
    });

    const details = createApplicationMapper.getExecutionDetails(ctx);
    expect(details["Display Name"]).toBe("my-app");
    expect(details["State"]).toBeUndefined();
    expect(details["Application ID"]).toBeUndefined();
  });
});
