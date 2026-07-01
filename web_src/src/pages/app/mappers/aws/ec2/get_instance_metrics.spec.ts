import { describe, expect, it } from "vitest";
import { getInstanceMetricsMapper } from "./get_instance_metrics";
import type { ComponentBaseContext, ExecutionDetailsContext, ExecutionInfo, NodeInfo } from "../../types";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Get Instance Metrics",
    componentName: "ec2.getInstanceMetrics",
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
      name: "ec2.getInstanceMetrics",
      label: "EC2 • Get Instance Metrics",
      description: "",
      icon: "aws",
      color: "gray",
    },
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
  };
}

describe("getInstanceMetricsMapper.props", () => {
  it("shows instance name and lookback period in metadata (no region)", () => {
    const props = getInstanceMetricsMapper.props(
      buildComponentCtx({
        configuration: { region: "us-east-1", instance: "i-abc123", lookbackPeriod: "1h" },
        metadata: { region: "us-east-1", instanceId: "i-abc123", instanceName: "my-server" },
      }),
    );

    expect(props.metadata).toEqual([
      { icon: "server", label: "my-server" },
      { icon: "hash", label: "i-abc123" },
      { icon: "clock", label: "Last 1 hour" },
    ]);
  });

  it("omits hash row when instanceName equals instanceId", () => {
    const props = getInstanceMetricsMapper.props(
      buildComponentCtx({
        configuration: { region: "us-east-1", instance: "i-abc123", lookbackPeriod: "24h" },
        metadata: { region: "us-east-1", instanceId: "i-abc123", instanceName: "i-abc123" },
      }),
    );

    expect(props.metadata).toEqual([
      { icon: "server", label: "i-abc123" },
      { icon: "clock", label: "Last 24 hours" },
    ]);
  });

  it("falls back to configuration instance when no metadata", () => {
    const props = getInstanceMetricsMapper.props(
      buildComponentCtx({
        configuration: { region: "eu-west-1", instance: "i-fallback", lookbackPeriod: "7d" },
      }),
    );

    const instanceItem = props.metadata?.find((m) => m.icon === "server");
    expect(instanceItem?.label).toBe("i-fallback");
  });
});

describe("getInstanceMetricsMapper.getExecutionDetails", () => {
  it("returns placeholder dashes when no output", () => {
    const details = getInstanceMetricsMapper.getExecutionDetails(
      buildDetailsCtx({
        node: { configuration: { region: "us-east-1", lookbackPeriod: "1h" } },
      }),
    );

    expect(details["Region"]).toBe("us-east-1");
    expect(details["Lookback Period"]).toBe("Last 1 hour");
    expect(details["Avg CPU"]).toBe("-");
    expect(details["Avg Net In"]).toBe("-");
    expect(details["Avg Net Out"]).toBe("-");
  });

  it("formats cpu and network metrics from output", () => {
    const details = getInstanceMetricsMapper.getExecutionDetails(
      buildDetailsCtx({
        node: { configuration: { region: "us-east-1", lookbackPeriod: "1h" } },
        execution: {
          outputs: {
            default: [
              buildOutput({
                instanceId: "i-abc123",
                region: "us-east-1",
                lookbackPeriod: "1h",
                start: "2026-05-21T11:00:00Z",
                end: "2026-05-21T12:00:00Z",
                avgCpuUsagePercent: 12.45,
                totalNetworkInBytes: 1_048_576,
                totalNetworkOutBytes: 524_288,
                avgNetworkInBytesPerSec: 291.27,
                avgNetworkOutBytesPerSec: 145.64,
              }),
            ],
          },
        },
      }),
    );

    expect(details["Avg CPU"]).toBe("12.45%");
    expect(details["Avg Net In"]).toContain("/s");
    expect(details["Avg Net Out"]).toContain("/s");
    expect(Object.keys(details)).not.toContain("Net In (total)");
    expect(Object.keys(details)).not.toContain("Net Out (total)");
  });

  it("includes memory row when avgMemoryUsagePercent is present", () => {
    const details = getInstanceMetricsMapper.getExecutionDetails(
      buildDetailsCtx({
        node: { configuration: { region: "us-east-1", lookbackPeriod: "1h" } },
        execution: {
          outputs: {
            default: [
              buildOutput({
                avgCpuUsagePercent: 5.0,
                totalNetworkInBytes: 0,
                totalNetworkOutBytes: 0,
                avgNetworkInBytesPerSec: 0,
                avgNetworkOutBytesPerSec: 0,
                avgMemoryUsagePercent: 67.2,
              }),
            ],
          },
        },
      }),
    );

    expect(details["Avg Memory"]).toBe("67.2%");
  });

  it("omits memory row when avgMemoryUsagePercent is absent", () => {
    const details = getInstanceMetricsMapper.getExecutionDetails(
      buildDetailsCtx({
        node: { configuration: { region: "us-east-1", lookbackPeriod: "1h" } },
        execution: {
          outputs: {
            default: [
              buildOutput({
                avgCpuUsagePercent: 5.0,
                totalNetworkInBytes: 0,
                totalNetworkOutBytes: 0,
                avgNetworkInBytesPerSec: 0,
                avgNetworkOutBytesPerSec: 0,
              }),
            ],
          },
        },
      }),
    );

    expect(Object.keys(details)).not.toContain("Avg Memory");
  });
});
