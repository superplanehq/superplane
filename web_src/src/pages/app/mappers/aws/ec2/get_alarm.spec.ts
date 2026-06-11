import { describe, expect, it } from "vitest";

import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
} from "../../types";
import { getAlarmMapper } from "./get_alarm";
import { eventStateRegistry } from "../index";

// ── Helpers ───────────────────────────────────────────────────────────────────

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Get Alarm Node",
    componentName: "aws.ec2.getAlarm",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildOutput(data: unknown): OutputPayload {
  return {
    type: "aws.ec2.alarm",
    timestamp: new Date().toISOString(),
    data,
  };
}

function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  return {
    id: "exec-1",
    createdAt: "2026-06-01T10:00:00.000Z",
    updatedAt: "2026-06-01T10:00:05.000Z",
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

const defaultDefinition: ComponentDefinition = {
  name: "aws.ec2.getAlarm",
  label: "EC2 • Get Alarm",
  description: "",
  icon: "aws",
  color: "gray",
};

function buildPropsContext(overrides?: Partial<ComponentBaseContext>): ComponentBaseContext {
  return {
    nodes: [],
    node: buildNode(),
    componentDefinition: defaultDefinition,
    lastExecutions: [],
    currentUser: undefined,
    actions: { invokeNodeExecutionHook: async () => {} },
    ...overrides,
  };
}

const alarmOutputData = {
  alarmName: "HighCPU",
  alarmArn: "arn:aws:cloudwatch:us-east-1:123456789012:alarm:HighCPU",
  alarmDescription: "High CPU utilization alarm",
  namespace: "AWS/EC2",
  metricName: "CPUUtilization",
  statistic: "Average",
  period: 300,
  evaluationPeriods: 1,
  threshold: 80,
  comparisonOperator: "GreaterThanThreshold",
  stateValue: "ALARM",
  stateReason: "Threshold Crossed: 1 datapoint (85.3) was greater than the threshold (80.0).",
  treatMissingData: "missing",
  dimensions: [{ name: "InstanceId", value: "i-abc123" }],
  region: "us-east-1",
};

// ── getExecutionDetails ───────────────────────────────────────────────────────

describe("getAlarmMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => getAlarmMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => getAlarmMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("uses configuration and node metadata when output is absent", () => {
    const ctx = buildDetailsCtx({
      node: {
        configuration: { region: "eu-west-1", alarm: "HighCPU" },
        metadata: { alarmName: "HighCPU", region: "eu-west-1" },
      },
      execution: { outputs: undefined },
    });
    const details = getAlarmMapper.getExecutionDetails(ctx);
    expect(details["Alarm Name"]).toBe("HighCPU");
    expect(details["Region"]).toBe("eu-west-1");
    expect(details["Metric"]).toBe("-");
    expect(details["State"]).toBe("-");
  });

  it("extracts alarm fields from output", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(alarmOutputData)] } },
    });
    const details = getAlarmMapper.getExecutionDetails(ctx);
    expect(details["Alarm Name"]).toBe("HighCPU");
    expect(details["Metric"]).toBe("CPUUtilization");
    expect(details["State"]).toBe("ALARM");
    expect(details["Region"]).toBe("us-east-1");
  });

  it("includes retrieved at timestamp", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(alarmOutputData)] } },
    });
    expect(getAlarmMapper.getExecutionDetails(ctx)["Retrieved At"]).toBeDefined();
  });

  it("prefers updatedAt over createdAt for retrieved timestamp", () => {
    const ctx = buildDetailsCtx({
      execution: {
        createdAt: "2026-06-01T10:00:00.000Z",
        updatedAt: "2026-06-01T10:00:05.000Z",
        outputs: { default: [buildOutput(alarmOutputData)] },
      },
    });
    const retrievedAt = getAlarmMapper.getExecutionDetails(ctx)["Retrieved At"];
    expect(retrievedAt).toBe(new Date("2026-06-01T10:00:05.000Z").toLocaleString());
  });
});

// ── props ─────────────────────────────────────────────────────────────────────

describe("getAlarmMapper.props", () => {
  it("uses node name as title", () => {
    const props = getAlarmMapper.props(buildPropsContext());
    expect(props.title).toBe("Get Alarm Node");
  });

  it("falls back to component label when node name is empty", () => {
    const props = getAlarmMapper.props(buildPropsContext({ node: buildNode({ name: "" }) }));
    expect(props.title).toBe("EC2 • Get Alarm");
  });

  it("includes alarm name from configuration in metadata", () => {
    const props = getAlarmMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { alarm: "HighCPU", region: "us-east-1" } }),
      }),
    );
    const labels = props.metadata?.map((m) => m.label) ?? [];
    expect(labels).toContain("HighCPU");
    expect(labels).toContain("us-east-1");
  });

  it("includes alarm name from node metadata when configuration has none", () => {
    const props = getAlarmMapper.props(
      buildPropsContext({
        node: buildNode({ metadata: { alarmName: "StatusCheckFailed" } }),
      }),
    );
    const labels = props.metadata?.map((m) => m.label) ?? [];
    expect(labels).toContain("StatusCheckFailed");
  });

  it("returns empty metadata when configuration and metadata are empty", () => {
    const props = getAlarmMapper.props(buildPropsContext({ node: buildNode({ configuration: {}, metadata: {} }) }));
    expect(props.metadata).toEqual([]);
  });

  it("sets includeEmptyState when no executions", () => {
    const props = getAlarmMapper.props(buildPropsContext({ lastExecutions: [] }));
    expect(props.includeEmptyState).toBe(true);
  });

  it("clears includeEmptyState when there is an execution", () => {
    const props = getAlarmMapper.props(buildPropsContext({ lastExecutions: [buildExecution()] }));
    expect(props.includeEmptyState).toBe(false);
  });
});

// ── eventStateRegistry ────────────────────────────────────────────────────────

describe("eventStateRegistry['ec2.getAlarm']", () => {
  it("maps finished success to retrieved", () => {
    expect(eventStateRegistry["ec2.getAlarm"].getState(buildExecution())).toBe("retrieved");
  });

  it("returns running when execution is in progress", () => {
    const running = buildExecution({
      state: "STATE_STARTED",
      result: "RESULT_UNSPECIFIED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_UNSPECIFIED" as ExecutionInfo["resultReason"],
    });
    expect(eventStateRegistry["ec2.getAlarm"].getState(running)).toBe("running");
  });

  it("returns failed when execution fails", () => {
    const failed = buildExecution({
      state: "STATE_FINISHED",
      result: "RESULT_FAILED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_COMPONENT_FAILED" as ExecutionInfo["resultReason"],
    });
    expect(eventStateRegistry["ec2.getAlarm"].getState(failed)).toBe("failed");
  });
});
