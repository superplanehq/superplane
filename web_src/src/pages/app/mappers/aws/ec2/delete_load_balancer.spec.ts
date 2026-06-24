import { describe, expect, it } from "vitest";

import { deleteLoadBalancerMapper } from "./delete_load_balancer";
import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../../types";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Delete Load Balancer",
    componentName: "ec2.deleteLoadBalancer",
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
      name: "ec2.deleteLoadBalancer",
      label: "EC2 • Delete Load Balancer",
      description: "",
      icon: "server",
      color: "gray",
    },
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

describe("deleteLoadBalancerMapper.props", () => {
  it("includes load balancer name and region in metadata", () => {
    const props = deleteLoadBalancerMapper.props(
      buildComponentCtx({
        configuration: {
          region: "us-east-1",
          loadBalancer: "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-alb/50dc6c495c0c9188",
        },
        metadata: {
          loadBalancerName: "my-alb",
          region: "us-east-1",
        },
      }),
    );

    expect(props.metadata).toEqual([
      { icon: "server", label: "my-alb" },
      { icon: "globe", label: "us-east-1" },
    ]);
  });

  it("falls back to load balancer ARN from configuration when name metadata is missing", () => {
    const arn = "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-alb/50dc6c495c0c9188";
    const props = deleteLoadBalancerMapper.props(
      buildComponentCtx({
        configuration: {
          region: "us-east-1",
          loadBalancer: arn,
        },
      }),
    );

    expect(props.metadata).toEqual([
      { icon: "server", label: arn },
      { icon: "globe", label: "us-east-1" },
    ]);
  });

  it("returns empty metadata when no configuration is present", () => {
    const props = deleteLoadBalancerMapper.props(buildComponentCtx());
    expect(props.metadata).toEqual([]);
  });
});

describe("deleteLoadBalancerMapper.getExecutionDetails", () => {
  it("shows deleted at and region while deletion is in progress", () => {
    const ctx = buildDetailsCtx({
      node: {
        configuration: {
          region: "us-east-1",
          loadBalancer: "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-alb/50dc6c495c0c9188",
        },
      },
      execution: {
        outputs: undefined,
      },
    });

    const details = deleteLoadBalancerMapper.getExecutionDetails(ctx);
    expect(details["Deleted At"]).toBe(new Date("2026-05-21T12:01:00Z").toLocaleString());
    expect(details["Region"]).toBe("us-east-1");
    expect(details["State"]).toBe("-");
  });

  it("maps deleted output state field", () => {
    const ctx = buildDetailsCtx({
      node: {
        configuration: {
          region: "eu-west-1",
        },
      },
      execution: {
        outputs: {
          default: [
            buildOutput({
              loadBalancerArn:
                "arn:aws:elasticloadbalancing:eu-west-1:123456789012:loadbalancer/app/my-alb/50dc6c495c0c9188",
              state: "deleting",
            }),
          ],
        },
      },
    });

    const details = deleteLoadBalancerMapper.getExecutionDetails(ctx);
    expect(details["Region"]).toBe("eu-west-1");
    expect(details["State"]).toBe("deleting");
  });
});
