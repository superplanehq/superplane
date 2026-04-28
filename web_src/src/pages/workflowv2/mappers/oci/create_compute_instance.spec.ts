import { describe, expect, it } from "vitest";

import { createComputeInstanceMapper } from "./create_compute_instance";
import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../types";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Create Compute Instance",
    componentName: "oci.createComputeInstance",
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
      name: "oci.createComputeInstance",
      label: "Create Compute Instance",
      description: "",
      icon: "server",
      color: "red",
    },
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

// ── props / metadata list ──────────────────────────────────────────────

describe("createComputeInstanceMapper.props", () => {
  it("includes displayName, shape, and availabilityDomain in metadata", () => {
    const props = createComputeInstanceMapper.props(
      buildComponentCtx({
        configuration: {
          displayName: "my-instance",
          shape: "VM.Standard.E4.Flex",
          availabilityDomain: "EXAMPLE:EU-FRANKFURT-1-AD-1",
        },
      }),
    );

    expect(props.metadata).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ icon: "tag", label: "my-instance" }),
        expect.objectContaining({ icon: "cpu", label: "VM.Standard.E4.Flex" }),
        expect.objectContaining({ icon: "map-pin", label: "EXAMPLE:EU-FRANKFURT-1-AD-1" }),
      ]),
    );
  });

  it("produces empty metadata when configuration is empty", () => {
    const props = createComputeInstanceMapper.props(buildComponentCtx({ configuration: {} }));
    expect(props.metadata).toEqual([]);
  });

  it("omits metadata items whose values are missing", () => {
    const props = createComputeInstanceMapper.props(
      buildComponentCtx({ configuration: { displayName: "my-instance" } }),
    );
    expect(props.metadata).toHaveLength(1);
    expect(props.metadata![0]).toMatchObject({ label: "my-instance" });
  });
});

// ── getExecutionDetails ────────────────────────────────────────────────

describe("createComputeInstanceMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => createComputeInstanceMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default output array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => createComputeInstanceMapper.getExecutionDetails(ctx)).not.toThrow();
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
    const details = createComputeInstanceMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBe(new Date(startedAt).toLocaleString());
  });

  it("falls back to execution.createdAt for Executed At when metadata.startedAt is absent", () => {
    const createdAt = new Date("2026-01-01T09:00:00Z").toISOString();
    const ctx = buildDetailsCtx({
      execution: { createdAt, metadata: {}, outputs: undefined },
    });
    const details = createComputeInstanceMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBe(new Date(createdAt).toLocaleString());
  });

  it("maps output fields to display labels", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              displayName: "my-instance",
              lifecycleState: "RUNNING",
              shape: "VM.Standard.E4.Flex",
              availabilityDomain: "EXAMPLE:EU-FRANKFURT-1-AD-1",
              region: "eu-frankfurt-1",
              publicIp: "1.2.3.4",
            }),
          ],
        },
      },
    });

    const details = createComputeInstanceMapper.getExecutionDetails(ctx);
    expect(details["Display Name"]).toBe("my-instance");
    expect(details["State"]).toBe("RUNNING");
    expect(details["Shape"]).toBe("VM.Standard.E4.Flex");
    expect(details["Availability Domain"]).toBe("EXAMPLE:EU-FRANKFURT-1-AD-1");
    expect(details["Region"]).toBe("eu-frankfurt-1");
    expect(details["Public IP"]).toBe("1.2.3.4");
  });

  it("omits optional fields that are absent from output", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [buildOutput({ displayName: "my-instance" })],
        },
      },
    });

    const details = createComputeInstanceMapper.getExecutionDetails(ctx);
    expect(details["Display Name"]).toBe("my-instance");
    expect(details["Public IP"]).toBeUndefined();
    expect(details["Region"]).toBeUndefined();
  });
});
