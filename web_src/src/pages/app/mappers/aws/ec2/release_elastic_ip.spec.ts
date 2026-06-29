import { describe, expect, it } from "vitest";

import { releaseElasticIPMapper } from "./release_elastic_ip";
import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../../types";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Release Elastic IP",
    componentName: "ec2.releaseElasticIP",
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
      name: "ec2.releaseElasticIP",
      label: "EC2 • Release Elastic IP",
      description: "",
      icon: "server",
      color: "gray",
    },
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

describe("releaseElasticIPMapper.props", () => {
  it("shows allocation ID and region in metadata", () => {
    const props = releaseElasticIPMapper.props(
      buildComponentCtx({
        configuration: { region: "us-east-1", allocationId: "eipalloc-abc123" },
        metadata: { region: "us-east-1", allocationId: "eipalloc-abc123" },
      }),
    );

    expect(props.metadata).toEqual([
      { icon: "hash", label: "eipalloc-abc123" },
      { icon: "globe", label: "us-east-1" },
    ]);
  });
});

describe("releaseElasticIPMapper.getExecutionDetails", () => {
  it("shows region while output is pending", () => {
    const ctx = buildDetailsCtx({
      node: {
        configuration: { region: "us-east-1", allocationId: "eipalloc-abc123" },
      },
      execution: { outputs: undefined },
    });

    const details = releaseElasticIPMapper.getExecutionDetails(ctx);
    expect(details["Completed At"]).toBeTruthy();
    expect(details["Region"]).toBe("us-east-1");
  });

  it("maps region from output", () => {
    const ctx = buildDetailsCtx({
      node: { configuration: { region: "us-east-1", allocationId: "eipalloc-abc123" } },
      execution: {
        outputs: {
          default: [
            {
              type: "aws.ec2.elastic-ip.released",
              timestamp: new Date().toISOString(),
              data: {
                allocationId: "eipalloc-abc123",
                region: "us-east-1",
              },
            },
          ],
        },
      },
    });

    const details = releaseElasticIPMapper.getExecutionDetails(ctx);
    expect(details["Region"]).toBe("us-east-1");
    expect(details["Allocation ID"]).toBeUndefined();
  });
});
