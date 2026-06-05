import { describe, expect, it } from "vitest";

import { getInstanceMapper } from "./get_instance";
import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../../types";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Get Instance",
    componentName: "ec2.getInstance",
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
      name: "ec2.getInstance",
      label: "EC2 • Get Instance",
      description: "",
      icon: "server",
      color: "gray",
    },
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

describe("getInstanceMapper.props", () => {
  it("shows instance name and region from node metadata", () => {
    const props = getInstanceMapper.props(
      buildComponentCtx({
        configuration: { region: "us-east-1", instance: "i-abc123" },
        metadata: {
          region: "us-east-1",
          instanceId: "i-abc123",
          instanceName: "my-server",
        },
      }),
    );

    expect(props.metadata).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ icon: "server", label: "my-server" }),
        expect.objectContaining({ icon: "hash", label: "i-abc123" }),
        expect.objectContaining({ icon: "globe", label: "us-east-1" }),
      ]),
    );
  });

  it("shows only the instance ID when no distinct name was resolved (name === id)", () => {
    // resolveInstanceName falls back to the raw instance ID, never an empty string,
    // so the realistic "no name" case is instanceName === instanceId.
    const props = getInstanceMapper.props(
      buildComponentCtx({
        configuration: { region: "eu-west-1", instance: "i-xyz999" },
        metadata: {
          region: "eu-west-1",
          instanceId: "i-xyz999",
          instanceName: "i-xyz999",
        },
      }),
    );

    const serverItem = props.metadata?.find((m) => m.icon === "server");
    expect(serverItem?.label).toBe("i-xyz999");
    // hash row must not appear when name and id are the same value
    const hashItem = props.metadata?.find((m) => m.icon === "hash");
    expect(hashItem).toBeUndefined();
  });

  it("falls back to configuration when node metadata is absent", () => {
    const props = getInstanceMapper.props(
      buildComponentCtx({
        configuration: { region: "ap-southeast-1", instance: "i-fallback" },
      }),
    );

    expect(props.metadata).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ icon: "server", label: "i-fallback" }),
        expect.objectContaining({ icon: "globe", label: "ap-southeast-1" }),
      ]),
    );
  });
});

describe("getInstanceMapper.getExecutionDetails", () => {
  it("shows placeholder fields while output is not yet available", () => {
    const ctx = buildDetailsCtx({
      node: {
        configuration: {
          region: "us-east-1",
          instance: "i-abc123",
        },
      },
      execution: { outputs: undefined },
    });

    const details = getInstanceMapper.getExecutionDetails(ctx);
    expect(details["Retrieved At"]).toBeTruthy();
    expect(details["Region"]).toBe("us-east-1");
    expect(details["State"]).toBe("-");
    expect(details["Instance Type"]).toBe("-");
    expect(details["Public IP"]).toBe("-");
  });

  it("maps output fields when instance details are present", () => {
    const ctx = buildDetailsCtx({
      node: {
        configuration: { region: "us-east-1" },
      },
      execution: {
        outputs: {
          default: [
            buildOutput({
              instanceId: "i-abc123",
              state: "running",
              instanceType: "t3.micro",
              publicIpAddress: "54.1.2.3",
              region: "us-east-1",
            }),
          ],
        },
      },
    });

    const details = getInstanceMapper.getExecutionDetails(ctx);
    expect(Object.keys(details)).toEqual(["Retrieved At", "Region", "State", "Instance Type", "Public IP"]);
    expect(details["State"]).toBe("running");
    expect(details["Instance Type"]).toBe("t3.micro");
    expect(details["Public IP"]).toBe("54.1.2.3");
  });

  it("prefers region from output over configuration", () => {
    const ctx = buildDetailsCtx({
      node: {
        configuration: { region: "us-east-1" },
      },
      execution: {
        outputs: {
          default: [
            buildOutput({
              instanceId: "i-abc123",
              state: "running",
              instanceType: "t3.micro",
              publicIpAddress: "",
              region: "eu-central-1",
            }),
          ],
        },
      },
    });

    const details = getInstanceMapper.getExecutionDetails(ctx);
    expect(details["Region"]).toBe("eu-central-1");
  });
});
