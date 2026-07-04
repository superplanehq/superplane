import { describe, expect, it } from "vitest";

import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
} from "../../types";
import { createAlarmMapper } from "./create_alarm";
import { eventStateRegistry } from "../index";

// ── Helpers ───────────────────────────────────────────────────────────────────

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Create Alarm Node",
    componentName: "aws.ec2.createAlarm",
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
  name: "aws.ec2.createAlarm",
  label: "EC2 • Create Alarm",
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
  stateValue: "OK",
  stateReason: "Threshold Crossed: no datapoints",
  treatMissingData: "missing",
  dimensions: [{ name: "InstanceId", value: "i-abc123" }],
  region: "us-east-1",
};

// ── getExecutionDetails ───────────────────────────────────────────────────────

describe("createAlarmMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => createAlarmMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => createAlarmMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("uses configuration values when output is absent", () => {
    const ctx = buildDetailsCtx({
      node: {
        configuration: {
          region: "eu-west-1",
          alarmName: "HighCPU",
          metricName: "CPUUtilization",
        },
      },
      execution: { outputs: undefined },
    });
    const details = createAlarmMapper.getExecutionDetails(ctx);
    expect(details["Region"]).toBe("eu-west-1");
    expect(details["Alarm Name"]).toBe("HighCPU");
    expect(details["Metric"]).toBe("CPUUtilization");
    expect(details["Threshold"]).toBe("-");
    expect(details["State"]).toBe("-");
  });

  it("extracts alarm fields from output", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(alarmOutputData)] } },
    });
    const details = createAlarmMapper.getExecutionDetails(ctx);
    expect(details["Alarm Name"]).toBe("HighCPU");
    expect(details["Metric"]).toBe("CPUUtilization");
    expect(details["Threshold"]).toBe("80");
    expect(details["State"]).toBe("OK");
    expect(details["Region"]).toBe("us-east-1");
  });

  it("includes created at timestamp", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(alarmOutputData)] } },
    });
    expect(createAlarmMapper.getExecutionDetails(ctx)["Created At"]).toBeDefined();
  });
});

// ── props ─────────────────────────────────────────────────────────────────────

describe("createAlarmMapper.props", () => {
  it("uses node name as title", () => {
    const props = createAlarmMapper.props(buildPropsContext());
    expect(props.title).toBe("Create Alarm Node");
  });

  it("falls back to component label when node name is empty", () => {
    const props = createAlarmMapper.props(buildPropsContext({ node: buildNode({ name: "" }) }));
    expect(props.title).toBe("EC2 • Create Alarm");
  });

  it("includes alarm name and metric in metadata from configuration", () => {
    const props = createAlarmMapper.props(
      buildPropsContext({
        node: buildNode({ configuration: { alarmName: "HighCPU", region: "us-east-1", metricName: "CPUUtilization" } }),
      }),
    );
    const labels = props.metadata?.map((m) => m.label) ?? [];
    expect(labels).toContain("HighCPU");
    expect(labels).toContain("CPUUtilization");
    expect(labels).not.toContain("us-east-1");
  });

  it("includes alarm name from node metadata when configuration has none", () => {
    const props = createAlarmMapper.props(
      buildPropsContext({
        node: buildNode({ metadata: { alarmName: "StatusCheckFailed" } }),
      }),
    );
    const labels = props.metadata?.map((m) => m.label) ?? [];
    expect(labels).toContain("StatusCheckFailed");
  });

  it("limits metadata to 3 items", () => {
    const props = createAlarmMapper.props(
      buildPropsContext({
        node: buildNode({
          configuration: { alarmName: "A", region: "us-east-1" },
          metadata: { instanceName: "my-server", instanceId: "i-abc" },
        }),
      }),
    );
    expect((props.metadata ?? []).length).toBeLessThanOrEqual(3);
  });

  it("sets includeEmptyState when no executions", () => {
    const props = createAlarmMapper.props(buildPropsContext({ lastExecutions: [] }));
    expect(props.includeEmptyState).toBe(true);
  });

  it("clears includeEmptyState when there is an execution", () => {
    const props = createAlarmMapper.props(buildPropsContext({ lastExecutions: [buildExecution()] }));
    expect(props.includeEmptyState).toBe(false);
  });
});

// ── eventStateRegistry ────────────────────────────────────────────────────────

describe("eventStateRegistry['ec2.createAlarm']", () => {
  it("maps finished success to created", () => {
    expect(eventStateRegistry["ec2.createAlarm"].getState(buildExecution())).toBe("created");
  });

  it("returns running when execution is in progress", () => {
    const running = buildExecution({
      state: "STATE_STARTED",
      result: "RESULT_UNSPECIFIED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_UNSPECIFIED" as ExecutionInfo["resultReason"],
    });
    expect(eventStateRegistry["ec2.createAlarm"].getState(running)).toBe("running");
  });

  it("returns failed when execution fails", () => {
    const failed = buildExecution({
      state: "STATE_FINISHED",
      result: "RESULT_FAILED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_COMPONENT_FAILED" as ExecutionInfo["resultReason"],
    });
    expect(eventStateRegistry["ec2.createAlarm"].getState(failed)).toBe("failed");
  });
});
