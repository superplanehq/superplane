import { describe, expect, it } from "vitest";

import { manageInstancePowerMapper, MANAGE_INSTANCE_POWER_STATE_REGISTRY } from "./manage_instance_power";
import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../../types";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Manage Instance Power",
    componentName: "ec2.manageInstancePower",
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
      name: "ec2.manageInstancePower",
      label: "EC2 • Manage Instance Power",
      description: "",
      icon: "server",
      color: "gray",
    },
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

describe("manageInstancePowerMapper.props", () => {
  it("shows instance name, operation label, and region in metadata", () => {
    const props = manageInstancePowerMapper.props(
      buildComponentCtx({
        configuration: { region: "us-east-1", instance: "i-abc123", operation: "start" },
        metadata: { region: "us-east-1", instanceName: "my-server" },
      }),
    );

    expect(props.metadata).toEqual([
      { icon: "server", label: "my-server" },
      { icon: "power", label: "Start" },
      { icon: "globe", label: "us-east-1" },
    ]);
  });

  it("falls back to raw instance ID when node metadata has no name", () => {
    const props = manageInstancePowerMapper.props(
      buildComponentCtx({
        configuration: { region: "us-east-1", instance: "i-abc123", operation: "stop" },
      }),
    );

    const serverItem = props.metadata?.find((m) => m.icon === "server");
    expect(serverItem?.label).toBe("i-abc123");
  });

  it("renders correct label for each operation", () => {
    const operations: Record<string, string> = {
      start: "Start",
      stop: "Stop",
      reboot: "Reboot",
      hibernate: "Hibernate",
    };

    for (const [value, label] of Object.entries(operations)) {
      const props = manageInstancePowerMapper.props(
        buildComponentCtx({
          configuration: { region: "us-east-1", instance: "i-abc123", operation: value },
          metadata: { instanceName: "my-server" },
        }),
      );

      const powerItem = props.metadata?.find((m) => m.icon === "power");
      expect(powerItem?.label).toBe(label);
    }
  });
});

describe("manageInstancePowerMapper.getExecutionDetails", () => {
  it("shows placeholder state while output is not yet available", () => {
    const ctx = buildDetailsCtx({
      node: {
        configuration: { region: "us-east-1", instance: "i-abc123", operation: "stop" },
      },
      execution: { outputs: undefined },
    });

    const details = manageInstancePowerMapper.getExecutionDetails(ctx);
    expect(details["Completed At"]).toBeTruthy();
    expect(details["Operation"]).toBe("Stop");
    expect(details["Region"]).toBe("us-east-1");
    expect(details["State"]).toBe("-");
  });

  it("maps output fields when instance details are present", () => {
    const ctx = buildDetailsCtx({
      node: {
        configuration: { region: "us-east-1", operation: "start" },
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

    const details = manageInstancePowerMapper.getExecutionDetails(ctx);
    expect(Object.keys(details)).toEqual(["Completed At", "Operation", "Region", "State", "Public IP"]);
    expect(details["Operation"]).toBe("Start");
    expect(details["State"]).toBe("running");
    expect(details["Public IP"]).toBe("54.1.2.3");
  });

  it("omits Public IP when the instance has no public address", () => {
    const ctx = buildDetailsCtx({
      node: { configuration: { region: "us-east-1", operation: "stop" } },
      execution: {
        outputs: {
          default: [
            buildOutput({
              instanceId: "i-abc123",
              state: "stopped",
              publicIpAddress: "",
              region: "us-east-1",
            }),
          ],
        },
      },
    });

    const details = manageInstancePowerMapper.getExecutionDetails(ctx);
    expect(details["Public IP"]).toBeUndefined();
  });

  it("shows stopped state for stop operation", () => {
    const ctx = buildDetailsCtx({
      node: {
        configuration: { region: "eu-west-1", operation: "stop" },
      },
      execution: {
        outputs: {
          default: [
            buildOutput({
              instanceId: "i-abc123",
              state: "stopped",
              region: "eu-west-1",
            }),
          ],
        },
      },
    });

    const details = manageInstancePowerMapper.getExecutionDetails(ctx);
    expect(details["Operation"]).toBe("Stop");
    expect(details["State"]).toBe("stopped");
  });

  it("shows stopped state for hibernate operation", () => {
    const ctx = buildDetailsCtx({
      node: {
        configuration: { region: "us-west-2", operation: "hibernate" },
      },
      execution: {
        outputs: {
          default: [
            buildOutput({
              instanceId: "i-abc123",
              state: "stopped",
              region: "us-west-2",
            }),
          ],
        },
      },
    });

    const details = manageInstancePowerMapper.getExecutionDetails(ctx);
    expect(details["Operation"]).toBe("Hibernate");
    expect(details["State"]).toBe("stopped");
  });

  it("shows running state for reboot operation", () => {
    const ctx = buildDetailsCtx({
      node: {
        configuration: { region: "us-east-1", operation: "reboot" },
      },
      execution: {
        outputs: {
          default: [
            buildOutput({
              instanceId: "i-abc123",
              state: "running",
              region: "us-east-1",
            }),
          ],
        },
      },
    });

    const details = manageInstancePowerMapper.getExecutionDetails(ctx);
    expect(details["Operation"]).toBe("Reboot");
    expect(details["State"]).toBe("running");
  });

  it("prefers region from output over configuration", () => {
    const ctx = buildDetailsCtx({
      node: { configuration: { region: "us-east-1", operation: "start" } },
      execution: {
        outputs: {
          default: [buildOutput({ instanceId: "i-1", state: "running", region: "ap-southeast-1" })],
        },
      },
    });

    const details = manageInstancePowerMapper.getExecutionDetails(ctx);
    expect(details["Region"]).toBe("ap-southeast-1");
  });
});

describe("MANAGE_INSTANCE_POWER_STATE_REGISTRY", () => {
  const successExecution = buildExecution({
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
  });

  it("returns the payload type as state when a power event is present", () => {
    const execution = {
      ...successExecution,
      outputs: {
        default: [{ type: "aws.ec2.instance.power.started", timestamp: "", data: {} }],
      },
    };

    expect(MANAGE_INSTANCE_POWER_STATE_REGISTRY.getState(execution)).toBe("aws.ec2.instance.power.started");
  });

  it("returns 'stopped' badge state for stop output", () => {
    const execution = {
      ...successExecution,
      outputs: {
        default: [{ type: "aws.ec2.instance.power.stopped", timestamp: "", data: {} }],
      },
    };

    expect(MANAGE_INSTANCE_POWER_STATE_REGISTRY.getState(execution)).toBe("aws.ec2.instance.power.stopped");
  });

  it("returns 'rebooted' badge state for reboot output", () => {
    const execution = {
      ...successExecution,
      outputs: {
        default: [{ type: "aws.ec2.instance.power.rebooted", timestamp: "", data: {} }],
      },
    };

    expect(MANAGE_INSTANCE_POWER_STATE_REGISTRY.getState(execution)).toBe("aws.ec2.instance.power.rebooted");
  });

  it("returns 'hibernated' badge state for hibernate output", () => {
    const execution = {
      ...successExecution,
      outputs: {
        default: [{ type: "aws.ec2.instance.power.hibernated", timestamp: "", data: {} }],
      },
    };

    expect(MANAGE_INSTANCE_POWER_STATE_REGISTRY.getState(execution)).toBe("aws.ec2.instance.power.hibernated");
  });

  it("falls back to 'success' when no recognised power event is present", () => {
    const execution = {
      ...successExecution,
      outputs: { default: [{ type: "aws.ec2.instance", timestamp: "", data: {} }] },
    };

    expect(MANAGE_INSTANCE_POWER_STATE_REGISTRY.getState(execution)).toBe("success");
  });

  it("passes non-success states through unchanged", () => {
    const failed = buildExecution({ state: "STATE_FINISHED", result: "RESULT_FAILED", resultMessage: "error" });
    expect(MANAGE_INSTANCE_POWER_STATE_REGISTRY.getState(failed)).toBe("error");
  });
});
