import { describe, expect, it } from "vitest";

import { manageElasticIPMapper, MANAGE_ELASTIC_IP_STATE_REGISTRY } from "./manage_elastic_ip";
import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../../types";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Manage Elastic IP",
    componentName: "ec2.manageElasticIP",
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
      name: "ec2.manageElasticIP",
      label: "EC2 • Manage Elastic IP",
      description: "",
      icon: "server",
      color: "gray",
    },
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

describe("manageElasticIPMapper.props", () => {
  it("shows instance name, operation label, and region in metadata", () => {
    const props = manageElasticIPMapper.props(
      buildComponentCtx({
        configuration: {
          region: "us-east-1",
          operation: "associate",
          instance: "i-abc123",
        },
        metadata: { region: "us-east-1", operation: "associate", instanceName: "my-server" },
      }),
    );

    expect(props.metadata).toEqual([
      { icon: "server", label: "my-server" },
      { icon: "link", label: "Associate" },
      { icon: "globe", label: "us-east-1" },
    ]);
  });

  it("renders correct label for disassociate operation", () => {
    const props = manageElasticIPMapper.props(
      buildComponentCtx({
        configuration: { region: "eu-west-1", operation: "disassociate" },
      }),
    );

    const operationItem = props.metadata?.find((m) => m.icon === "link");
    expect(operationItem?.label).toBe("Disassociate");
  });

  it("does not show instance metadata for disassociate operation", () => {
    const props = manageElasticIPMapper.props(
      buildComponentCtx({
        configuration: {
          region: "us-east-1",
          operation: "disassociate",
          instance: "i-abc123",
        },
        metadata: { region: "us-east-1", operation: "disassociate", instanceName: "my-server" },
      }),
    );

    expect(props.metadata).toEqual([
      { icon: "link", label: "Disassociate" },
      { icon: "globe", label: "us-east-1" },
    ]);
  });
});

describe("manageElasticIPMapper.getExecutionDetails", () => {
  it("shows operation and region while output is not yet available", () => {
    const ctx = buildDetailsCtx({
      node: {
        configuration: { region: "us-east-1", operation: "associate" },
      },
      execution: { outputs: undefined },
    });

    const details = manageElasticIPMapper.getExecutionDetails(ctx);
    expect(details["Completed At"]).toBeTruthy();
    expect(details["Operation"]).toBe("Associate");
    expect(details["Region"]).toBe("us-east-1");
  });

  it("maps associate output without ID fields", () => {
    const ctx = buildDetailsCtx({
      node: {
        configuration: { region: "us-east-1", operation: "associate" },
      },
      execution: {
        outputs: {
          default: [
            {
              type: "aws.ec2.elastic-ip.associated",
              timestamp: new Date().toISOString(),
              data: {
                associationId: "eipassoc-xyz789",
                allocationId: "eipalloc-abc123",
                instanceId: "i-abc123",
                region: "us-east-1",
              },
            },
          ],
        },
      },
    });

    const details = manageElasticIPMapper.getExecutionDetails(ctx);
    expect(details["Operation"]).toBe("Associate");
    expect(details["Region"]).toBe("us-east-1");
    expect(details["Association ID"]).toBeUndefined();
    expect(details["Allocation ID"]).toBeUndefined();
    expect(details["Instance ID"]).toBeUndefined();
  });

  it("uses output payload type for operation when configuration changed after run", () => {
    const ctx = buildDetailsCtx({
      node: {
        configuration: { region: "us-east-1", operation: "associate" },
      },
      execution: {
        outputs: {
          default: [
            {
              type: "aws.ec2.elastic-ip.disassociated",
              timestamp: new Date().toISOString(),
              data: {
                associationId: "eipassoc-xyz789",
                region: "us-east-1",
              },
            },
          ],
        },
      },
    });

    const details = manageElasticIPMapper.getExecutionDetails(ctx);
    expect(details["Operation"]).toBe("Disassociate");
    expect(details["Region"]).toBe("us-east-1");
  });

  it("maps disassociate output without ID fields", () => {
    const ctx = buildDetailsCtx({
      node: {
        configuration: { region: "us-east-1", operation: "disassociate" },
      },
      execution: {
        outputs: {
          default: [
            {
              type: "aws.ec2.elastic-ip.disassociated",
              timestamp: new Date().toISOString(),
              data: {
                associationId: "eipassoc-xyz789",
                region: "us-east-1",
              },
            },
          ],
        },
      },
    });

    const details = manageElasticIPMapper.getExecutionDetails(ctx);
    expect(details["Operation"]).toBe("Disassociate");
    expect(details["Region"]).toBe("us-east-1");
    expect(details["Association ID"]).toBeUndefined();
    expect(details["Allocation ID"]).toBeUndefined();
    expect(details["Instance ID"]).toBeUndefined();
  });
});

describe("MANAGE_ELASTIC_IP_STATE_REGISTRY", () => {
  const successExecution = buildExecution({
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
  });

  it("returns associated payload type as state", () => {
    const execution = {
      ...successExecution,
      outputs: {
        default: [{ type: "aws.ec2.elastic-ip.associated", timestamp: "", data: {} }],
      },
    };

    expect(MANAGE_ELASTIC_IP_STATE_REGISTRY.getState(execution)).toBe("aws.ec2.elastic-ip.associated");
  });

  it("returns disassociated payload type as state", () => {
    const execution = {
      ...successExecution,
      outputs: {
        default: [{ type: "aws.ec2.elastic-ip.disassociated", timestamp: "", data: {} }],
      },
    };

    expect(MANAGE_ELASTIC_IP_STATE_REGISTRY.getState(execution)).toBe("aws.ec2.elastic-ip.disassociated");
  });

  it("falls back to success when no recognised elastic IP event is present", () => {
    const execution = {
      ...successExecution,
      outputs: { default: [{ type: "aws.ec2.instance", timestamp: "", data: {} }] },
    };

    expect(MANAGE_ELASTIC_IP_STATE_REGISTRY.getState(execution)).toBe("success");
  });

  it("passes non-success states through unchanged", () => {
    const failed = buildExecution({ state: "STATE_FINISHED", result: "RESULT_FAILED", resultMessage: "error" });
    expect(MANAGE_ELASTIC_IP_STATE_REGISTRY.getState(failed)).toBe("error");
  });
});
