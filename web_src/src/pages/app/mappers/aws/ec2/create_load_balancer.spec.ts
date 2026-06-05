import { describe, expect, it } from "vitest";

import { createLoadBalancerMapper } from "./create_load_balancer";
import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../../types";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Create Load Balancer",
    componentName: "ec2.createLoadBalancer",
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
      name: "ec2.createLoadBalancer",
      label: "EC2 • Create Load Balancer",
      description: "",
      icon: "server",
      color: "gray",
    },
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

describe("createLoadBalancerMapper.props", () => {
  it("includes name, type, and region in metadata", () => {
    const props = createLoadBalancerMapper.props(
      buildComponentCtx({
        configuration: {
          name: "my-alb",
          region: "us-east-1",
          type: "application",
        },
        metadata: {
          name: "my-alb",
          type: "application",
          region: "us-east-1",
        },
      }),
    );

    expect(props.metadata).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ icon: "server", label: "my-alb" }),
        expect.objectContaining({ icon: "layers", label: "application" }),
        expect.objectContaining({ icon: "globe", label: "us-east-1" }),
      ]),
    );
  });

  it("falls back to configuration values when node metadata is absent", () => {
    const props = createLoadBalancerMapper.props(
      buildComponentCtx({
        configuration: {
          name: "my-nlb",
          region: "eu-west-1",
          type: "network",
        },
      }),
    );

    expect(props.metadata).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ icon: "server", label: "my-nlb" }),
        expect.objectContaining({ icon: "layers", label: "network" }),
        expect.objectContaining({ icon: "globe", label: "eu-west-1" }),
      ]),
    );
  });

  it("returns empty metadata when no configuration is present", () => {
    const props = createLoadBalancerMapper.props(buildComponentCtx());
    expect(props.metadata).toEqual([]);
  });
});

describe("createLoadBalancerMapper.getExecutionDetails", () => {
  it("shows core fields while the load balancer is still provisioning", () => {
    const ctx = buildDetailsCtx({
      node: {
        configuration: {
          name: "my-alb",
          region: "us-east-1",
          type: "application",
          scheme: "internet-facing",
        },
      },
      execution: {
        outputs: undefined,
      },
    });

    const details = createLoadBalancerMapper.getExecutionDetails(ctx);
    expect(details["Created At"]).toBe(new Date("2026-05-21T12:00:00Z").toLocaleString());
    expect(details["Name"]).toBe("my-alb");
    expect(details["Region"]).toBe("us-east-1");
    expect(details["Type"]).toBe("application");
    expect(details["Scheme"]).toBe("internet-facing");
    expect(details["State"]).toBe("-");
    expect(details["DNS Name"]).toBe("-");
  });

  it("maps active load balancer output fields", () => {
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
              loadBalancerArn:
                "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/my-alb/50dc6c495c0c9188",
              name: "my-alb",
              dnsName: "my-alb-123456789.us-east-1.elb.amazonaws.com",
              scheme: "internet-facing",
              type: "application",
              state: "active",
              vpcId: "vpc-0598c7d356eba48d7",
              region: "us-east-1",
            }),
          ],
        },
      },
    });

    const details = createLoadBalancerMapper.getExecutionDetails(ctx);
    expect(details["Name"]).toBe("my-alb");
    expect(details["Region"]).toBe("us-east-1");
    expect(details["Type"]).toBe("application");
    expect(details["Scheme"]).toBe("internet-facing");
    expect(details["State"]).toBe("active");
    expect(details["DNS Name"]).toBe("my-alb-123456789.us-east-1.elb.amazonaws.com");
  });
});
