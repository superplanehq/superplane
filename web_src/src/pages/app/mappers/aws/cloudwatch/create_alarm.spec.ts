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
    componentName: "aws.cloudwatch.createAlarm",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildOutput(data: unknown): OutputPayload {
  return {
    type: "aws.cloudwatch.alarm",
    timestamp: new Date().toISOString(),
    data,
  };
}

function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  return {
    id: "exec-1",
    createdAt: "2026-06-03T16:36:13.000Z",
    updatedAt: "2026-06-03T16:36:15.000Z",
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
  name: "aws.cloudwatch.createAlarm",
  label: "CloudWatch • Create Alarm",
  description: "",
  icon: "aws.cloudwatch",
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
  alarmName: "api-high-cpu",
  alarmArn: "arn:aws:cloudwatch:us-east-1:123456789012:alarm:api-high-cpu",
  namespace: "AWS/EC2",
  metricName: "CPUUtilization",
  statistic: "Average",
  comparisonOperator: "GreaterThanThreshold",
  threshold: 80,
  period: 300,
  evaluationPeriods: 3,
  stateValue: "INSUFFICIENT_DATA",
  dimensions: [{ name: "InstanceId", value: "i-1234567890abcdef0" }],
  region: "us-east-1",
  consoleUrl: "https://us-east-1.console.aws.amazon.com/cloudwatch/home?region=us-east-1#alarmsV2:alarm/api-high-cpu",
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

  it("shows at most six rows, starting with the timestamp", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(alarmOutputData)] } },
    });
    const keys = Object.keys(createAlarmMapper.getExecutionDetails(ctx));
    expect(keys.length).toBeLessThanOrEqual(6);
    expect(keys[0]).toBe("Created At");
  });

  it("extracts alarm fields from output", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(alarmOutputData)] } },
    });
    const details = createAlarmMapper.getExecutionDetails(ctx);
    expect(details["Alarm Name"]).toBe("api-high-cpu");
    expect(details["Metric"]).toBe("AWS/EC2 / CPUUtilization");
    expect(details["Condition"]).toBe("Average > 80");
    expect(details["State"]).toBe("INSUFFICIENT_DATA");
  });

  it("uses the console URL from the output", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(alarmOutputData)] } },
    });
    expect(createAlarmMapper.getExecutionDetails(ctx)["CloudWatch Console"]).toBe(alarmOutputData.consoleUrl);
  });

  it("builds the console URL from configuration when output has none", () => {
    const ctx = buildDetailsCtx({
      node: { configuration: { region: "eu-west-1", alarmName: "queue depth" } },
      execution: { outputs: undefined },
    });
    expect(createAlarmMapper.getExecutionDetails(ctx)["CloudWatch Console"]).toBe(
      "https://eu-west-1.console.aws.amazon.com/cloudwatch/home?region=eu-west-1#alarmsV2:alarm/queue%20depth",
    );
  });

  it("falls back to configuration and node metadata when output is absent", () => {
    const ctx = buildDetailsCtx({
      node: {
        configuration: {
          region: "eu-west-1",
          alarmName: "api-high-cpu",
          namespace: "AWS/SQS",
          metricName: "ApproximateNumberOfMessagesVisible",
          statistic: "Maximum",
          comparisonOperator: "GreaterThanOrEqualToThreshold",
          threshold: 100,
        },
        metadata: { alarmName: "api-high-cpu", region: "eu-west-1" },
      },
      execution: { outputs: undefined },
    });
    const details = createAlarmMapper.getExecutionDetails(ctx);
    expect(details["Alarm Name"]).toBe("api-high-cpu");
    expect(details["Metric"]).toBe("AWS/SQS / ApproximateNumberOfMessagesVisible");
    expect(details["Condition"]).toBe("Maximum >= 100");
    expect(details["State"]).toBe("-");
  });

  it("includes the created at timestamp", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(alarmOutputData)] } },
    });
    expect(createAlarmMapper.getExecutionDetails(ctx)["Created At"]).toBe(
      new Date("2026-06-03T16:36:13.000Z").toLocaleString(),
    );
  });
});

// ── props ─────────────────────────────────────────────────────────────────────

describe("createAlarmMapper.props", () => {
  it("uses node name as title", () => {
    expect(createAlarmMapper.props(buildPropsContext()).title).toBe("Create Alarm Node");
  });

  it("falls back to component label when node name is empty", () => {
    const props = createAlarmMapper.props(buildPropsContext({ node: buildNode({ name: "" }) }));
    expect(props.title).toBe("CloudWatch • Create Alarm");
  });

  it("includes alarm, metric and dimensions in metadata", () => {
    const props = createAlarmMapper.props(
      buildPropsContext({
        node: buildNode({
          configuration: { alarmName: "api-high-cpu", metricName: "CPUUtilization" },
          metadata: { dimensions: "InstanceId=i-1234567890abcdef0" },
        }),
      }),
    );
    const labels = props.metadata?.map((item) => item.label) ?? [];
    expect(labels).toEqual(["api-high-cpu", "CPUUtilization", "InstanceId=i-1234567890abcdef0"]);
  });

  it("returns empty metadata when configuration and metadata are empty", () => {
    const props = createAlarmMapper.props(buildPropsContext({ node: buildNode({ configuration: {}, metadata: {} }) }));
    expect(props.metadata).toEqual([]);
  });

  it("sets includeEmptyState when no executions", () => {
    expect(createAlarmMapper.props(buildPropsContext({ lastExecutions: [] })).includeEmptyState).toBe(true);
  });

  it("clears includeEmptyState when there is an execution", () => {
    const props = createAlarmMapper.props(buildPropsContext({ lastExecutions: [buildExecution()] }));
    expect(props.includeEmptyState).toBe(false);
  });
});

// ── eventStateRegistry ────────────────────────────────────────────────────────

describe("eventStateRegistry['cloudwatch.createAlarm']", () => {
  it("maps finished success to created", () => {
    expect(eventStateRegistry["cloudwatch.createAlarm"].getState(buildExecution())).toBe("created");
  });

  it("returns running when execution is in progress", () => {
    const running = buildExecution({
      state: "STATE_STARTED",
      result: "RESULT_UNSPECIFIED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_UNSPECIFIED" as ExecutionInfo["resultReason"],
    });
    expect(eventStateRegistry["cloudwatch.createAlarm"].getState(running)).toBe("running");
  });

  it("returns failed when execution fails", () => {
    const failed = buildExecution({
      state: "STATE_FINISHED",
      result: "RESULT_FAILED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_COMPONENT_FAILED" as ExecutionInfo["resultReason"],
    });
    expect(eventStateRegistry["cloudwatch.createAlarm"].getState(failed)).toBe("failed");
  });
});
