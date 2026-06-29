import { describe, expect, it } from "vitest";
import { updateInstanceMapper } from "./update_instance";
import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../../types";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Update Instance",
    componentName: "ec2.updateInstance",
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
    updatedAt: new Date("2026-05-21T12:05:00Z").toISOString(),
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
      name: "ec2.updateInstance",
      label: "EC2 • Update Instance",
      description: "",
      icon: "aws",
      color: "gray",
    },
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

describe("updateInstanceMapper.props", () => {
  it("shows instance name and instance type in metadata (no region)", () => {
    const props = updateInstanceMapper.props(
      buildComponentCtx({
        configuration: { region: "us-east-1", instance: "i-abc123", instanceType: "t3.medium" },
        metadata: { region: "us-east-1", instanceId: "i-abc123", instanceName: "my-server" },
      }),
    );

    expect(props.metadata).toEqual([
      { icon: "server", label: "my-server" },
      { icon: "hash", label: "i-abc123" },
      { icon: "cpu", label: "t3.medium" },
    ]);
  });

  it("omits hash row when instanceName equals instanceId", () => {
    const props = updateInstanceMapper.props(
      buildComponentCtx({
        configuration: { region: "us-east-1", instance: "i-abc123", instanceType: "t3.large" },
        metadata: { region: "us-east-1", instanceId: "i-abc123", instanceName: "i-abc123" },
      }),
    );

    const hashItem = props.metadata?.find((m) => m.icon === "hash");
    expect(hashItem).toBeUndefined();
  });

  it("shows security group when configured", () => {
    const props = updateInstanceMapper.props(
      buildComponentCtx({
        configuration: { region: "us-east-1", instance: "i-abc123", securityGroups: "sg-111aaa" },
        metadata: { instanceId: "i-abc123", instanceName: "my-server" },
      }),
    );

    expect(props.metadata).toContainEqual({ icon: "shield", label: "sg-111aaa" });
  });

  it("shows both instance type and security group when both are configured", () => {
    const props = updateInstanceMapper.props(
      buildComponentCtx({
        configuration: {
          region: "us-east-1",
          instance: "i-abc123",
          instanceType: "t3.medium",
          securityGroups: "sg-111aaa",
        },
        metadata: { instanceId: "i-abc123", instanceName: "my-server" },
      }),
    );

    expect(props.metadata).toContainEqual({ icon: "cpu", label: "t3.medium" });
    expect(props.metadata).toContainEqual({ icon: "shield", label: "sg-111aaa" });
  });

  it("omits instance type and security group rows when neither is configured", () => {
    const props = updateInstanceMapper.props(
      buildComponentCtx({
        configuration: { region: "us-east-1", instance: "i-abc123" },
        metadata: { instanceId: "i-abc123", instanceName: "my-server" },
      }),
    );

    const cpuItem = props.metadata?.find((m) => m.icon === "cpu");
    const shieldItem = props.metadata?.find((m) => m.icon === "shield");
    expect(cpuItem).toBeUndefined();
    expect(shieldItem).toBeUndefined();
  });

  it("falls back to configuration instance when no metadata", () => {
    const props = updateInstanceMapper.props(
      buildComponentCtx({
        configuration: { region: "eu-west-1", instance: "i-fallback", instanceType: "t3.small" },
      }),
    );

    const instanceItem = props.metadata?.find((m) => m.icon === "server");
    expect(instanceItem?.label).toBe("i-fallback");
  });
});

describe("updateInstanceMapper.getExecutionDetails", () => {
  it("returns placeholder dashes when no output", () => {
    const details = updateInstanceMapper.getExecutionDetails(
      buildDetailsCtx({
        node: {
          configuration: { region: "us-east-1", instanceType: "t3.medium" },
        },
      }),
    );

    expect(details["Region"]).toBe("us-east-1");
    expect(details["State"]).toBe("-");
    expect(details["Instance Type"]).toBe("t3.medium");
    expect(details["Public IP"]).toBe("-");
  });

  it("shows output fields when execution succeeded", () => {
    const details = updateInstanceMapper.getExecutionDetails(
      buildDetailsCtx({
        node: { configuration: { region: "us-east-1", instanceType: "t3.large" } },
        execution: {
          outputs: {
            default: [
              buildOutput({
                instanceId: "i-abc123",
                instanceType: "t3.large",
                state: "running",
                region: "us-east-1",
                publicIpAddress: "54.210.167.204",
                privateIpAddress: "10.0.1.23",
              }),
            ],
          },
        },
      }),
    );

    expect(details["State"]).toBe("running");
    expect(details["Instance Type"]).toBe("t3.large");
    expect(details["Public IP"]).toBe("54.210.167.204");
  });

  it("omits Public IP when publicIpAddress is empty", () => {
    const details = updateInstanceMapper.getExecutionDetails(
      buildDetailsCtx({
        node: { configuration: { region: "us-east-1" } },
        execution: {
          outputs: {
            default: [
              buildOutput({
                instanceId: "i-abc123",
                instanceType: "t3.large",
                state: "stopped",
                region: "us-east-1",
                publicIpAddress: "",
              }),
            ],
          },
        },
      }),
    );

    expect(Object.keys(details)).not.toContain("Public IP");
  });
});
