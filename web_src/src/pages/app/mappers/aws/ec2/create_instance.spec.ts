import { describe, expect, it } from "vitest";

import { createInstanceMapper, CREATE_INSTANCE_STATE_REGISTRY } from "./create_instance";
import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../../types";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Create Instance",
    componentName: "ec2.createInstance",
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
      name: "ec2.createInstance",
      label: "EC2 • Create Instance",
      description: "",
      icon: "server",
      color: "gray",
    },
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

describe("createInstanceMapper.props", () => {
  it("includes name, region, and operating system in metadata", () => {
    const props = createInstanceMapper.props(
      buildComponentCtx({
        configuration: {
          name: "builder",
          region: "us-east-1",
          imageOs: "ubuntu",
        },
        metadata: {
          imageOsLabel: "Ubuntu",
        },
      }),
    );

    expect(props.metadata).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ icon: "tag", label: "builder" }),
        expect.objectContaining({ icon: "globe", label: "us-east-1" }),
        expect.objectContaining({ icon: "disc", label: "Ubuntu" }),
      ]),
    );
  });

  it("limits node metadata to three items", () => {
    const props = createInstanceMapper.props(
      buildComponentCtx({
        configuration: {
          name: "builder",
          region: "us-east-1",
          imageOs: "ubuntu",
          instanceType: "t3.micro",
          configureRootVolume: true,
          volumeSizeGiB: 30,
          volumeType: "gp3",
        },
        metadata: {
          imageOsLabel: "Ubuntu",
          imageName: "Ubuntu 22.04",
          subnetName: "public-a",
        },
      }),
    );

    expect(props.metadata).toHaveLength(3);
  });
});

describe("createInstanceMapper.getExecutionDetails", () => {
  it("shows core fields while the instance is still launching", () => {
    const ctx = buildDetailsCtx({
      node: {
        configuration: {
          name: "builder",
          region: "us-east-1",
          instanceType: "t3.micro",
        },
      },
      execution: {
        metadata: { instanceId: "i-abc123" },
        outputs: undefined,
      },
    });

    const details = createInstanceMapper.getExecutionDetails(ctx);
    expect(details["Created At"]).toBeTruthy();
    expect(details["Name"]).toBe("builder");
    expect(details["Region"]).toBe("us-east-1");
    expect(details["State"]).toBe("-");
    expect(details["Instance Type"]).toBe("t3.micro");
    expect(details["Public IP"]).toBe("-");
    expect(details["Instance ID"]).toBeUndefined();
  });

  it("maps only the requested output fields", () => {
    const ctx = buildDetailsCtx({
      node: {
        configuration: {
          name: "builder",
          region: "us-east-1",
        },
      },
      execution: {
        createdAt: new Date("2026-05-21T12:00:00Z").toISOString(),
        outputs: {
          default: [
            buildOutput({
              instanceId: "i-abc123",
              name: "builder",
              state: "running",
              instanceType: "t3.micro",
              imageId: "ami-123",
              publicIpAddress: "54.1.2.3",
              privateIpAddress: "10.0.0.10",
            }),
          ],
        },
      },
    });

    const details = createInstanceMapper.getExecutionDetails(ctx);
    expect(Object.keys(details)).toEqual(["Created At", "Name", "Region", "State", "Instance Type", "Public IP"]);
    expect(details["Name"]).toBe("builder");
    expect(details["State"]).toBe("running");
    expect(details["Public IP"]).toBe("54.1.2.3");
    expect(details["Private IP"]).toBeUndefined();
  });

  it("surfaces the failure output when the run emitted to the failed channel", () => {
    const ctx = buildDetailsCtx({
      node: {
        configuration: {
          name: "builder",
          region: "us-east-1",
          instanceType: "t3.micro",
        },
      },
      execution: {
        outputs: {
          failed: [
            buildOutput({
              error: "not enough capacity",
              awsErrorCode: "InsufficientInstanceCapacity",
              lastObservedState: "",
            }),
          ],
        },
      },
    });

    const details = createInstanceMapper.getExecutionDetails(ctx);
    expect(details["State"]).toBe("Failed");
    expect(details["Error"]).toBe("InsufficientInstanceCapacity: not enough capacity");
    expect(details["Public IP"]).toBeUndefined();
  });
});

describe("CREATE_INSTANCE_STATE_REGISTRY", () => {
  it("returns created for a passed execution that emitted to the created channel", () => {
    const execution = buildExecution({
      outputs: { created: [buildOutput({ instanceId: "i-abc123", state: "running" })] },
    });

    expect(CREATE_INSTANCE_STATE_REGISTRY.getState(execution)).toBe("created");
  });

  it("returns failed when the run emitted to the failed channel even though the execution passed", () => {
    const execution = buildExecution({
      outputs: { failed: [buildOutput({ error: "boom", awsErrorCode: "InsufficientInstanceCapacity" })] },
    });

    expect(CREATE_INSTANCE_STATE_REGISTRY.getState(execution)).toBe("failed");
  });

  it("returns running while the execution is still in progress", () => {
    const execution = buildExecution({
      state: "STATE_STARTED",
      result: "RESULT_UNSPECIFIED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_UNSPECIFIED" as ExecutionInfo["resultReason"],
      outputs: undefined,
    });

    expect(CREATE_INSTANCE_STATE_REGISTRY.getState(execution)).toBe("running");
  });
});
