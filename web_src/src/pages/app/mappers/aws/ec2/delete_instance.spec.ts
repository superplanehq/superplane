import { describe, expect, it } from "vitest";

import { deleteInstanceMapper } from "./delete_instance";
import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../../types";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Delete Instance",
    componentName: "ec2.deleteInstance",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  return {
    id: "exec-1",
    createdAt: new Date("2026-05-21T12:00:00Z").toISOString(),
    updatedAt: new Date("2026-05-21T12:01:00Z").toISOString(),
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
      name: "ec2.deleteInstance",
      label: "EC2 • Delete Instance",
      description: "",
      icon: "server",
      color: "gray",
    },
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

describe("deleteInstanceMapper.props", () => {
  it("includes instance name and region in metadata", () => {
    const props = deleteInstanceMapper.props(
      buildComponentCtx({
        configuration: {
          region: "us-east-1",
          instance: "i-abc123",
        },
        metadata: {
          instanceName: "my-instance",
          region: "us-east-1",
        },
      }),
    );

    expect(props.metadata).toEqual([
      { icon: "server", label: "my-instance" },
      { icon: "globe", label: "us-east-1" },
    ]);
  });

  it("falls back to instance id when name metadata is missing", () => {
    const props = deleteInstanceMapper.props(
      buildComponentCtx({
        configuration: {
          region: "us-east-1",
          instance: "i-abc123",
        },
      }),
    );

    expect(props.metadata).toEqual([
      { icon: "server", label: "i-abc123" },
      { icon: "globe", label: "us-east-1" },
    ]);
  });
});

describe("deleteInstanceMapper.getExecutionDetails", () => {
  it("shows deleted at and region while termination is in progress", () => {
    const ctx = buildDetailsCtx({
      node: {
        configuration: {
          region: "us-east-1",
          instance: "i-abc123",
        },
      },
      execution: {
        outputs: undefined,
      },
    });

    const details = deleteInstanceMapper.getExecutionDetails(ctx);
    expect(details["Deleted At"]).toBe(new Date("2026-05-21T12:01:00Z").toLocaleString());
    expect(details["Region"]).toBe("us-east-1");
    expect(details["State"]).toBe("-");
    expect(details["Instance ID"]).toBeUndefined();
  });

  it("maps terminated output fields without instance id", () => {
    const ctx = buildDetailsCtx({
      node: {
        configuration: {
          region: "us-east-1",
        },
      },
      execution: {
        outputs: {
          default: [
            buildOutput({
              instance: "i-abc123",
              state: "terminated",
            }),
          ],
        },
      },
    });

    const details = deleteInstanceMapper.getExecutionDetails(ctx);
    expect(details["Deleted At"]).toBe(new Date("2026-05-21T12:01:00Z").toLocaleString());
    expect(details["State"]).toBe("terminated");
    expect(details["Instance ID"]).toBeUndefined();
  });
});
