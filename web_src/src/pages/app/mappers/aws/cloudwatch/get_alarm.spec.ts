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
    componentName: "aws.cloudwatch.getAlarm",
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
    createdAt: "2026-06-04T09:12:40.000Z",
    updatedAt: "2026-06-04T09:12:45.000Z",
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
  name: "aws.cloudwatch.getAlarm",
  label: "CloudWatch • Get Alarm",
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
  namespace: "AWS/EC2",
  metricName: "CPUUtilization",
  statistic: "Average",
  comparisonOperator: "GreaterThanThreshold",
  threshold: 90,
  stateValue: "ALARM",
  region: "us-east-1",
  consoleUrl: "https://us-east-1.console.aws.amazon.com/cloudwatch/home?region=us-east-1#alarmsV2:alarm/api-high-cpu",
};

const configuration = { region: "us-east-1", alarm: "api-high-cpu" };

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

  it("shows at most six rows, starting with the timestamp", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(alarmOutputData)] } },
    });
    const keys = Object.keys(getAlarmMapper.getExecutionDetails(ctx));
    expect(keys.length).toBeLessThanOrEqual(6);
    expect(keys[0]).toBe("Retrieved At");
  });

  it("extracts alarm fields from output", () => {
    const ctx = buildDetailsCtx({
      node: { configuration },
      execution: { outputs: { default: [buildOutput(alarmOutputData)] } },
    });
    const details = getAlarmMapper.getExecutionDetails(ctx);
    expect(details["Alarm Name"]).toBe("api-high-cpu");
    expect(details["Metric"]).toBe("AWS/EC2 / CPUUtilization");
    expect(details["Condition"]).toBe("Average > 90");
    expect(details["State"]).toBe("ALARM");
  });

  it("uses the console URL from the output", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(alarmOutputData)] } },
    });
    expect(getAlarmMapper.getExecutionDetails(ctx)["CloudWatch Console"]).toBe(alarmOutputData.consoleUrl);
  });

  it("builds the console URL from configuration when the output has none", () => {
    const ctx = buildDetailsCtx({
      node: { configuration },
      execution: { outputs: { default: [buildOutput({ ...alarmOutputData, consoleUrl: undefined })] } },
    });
    expect(getAlarmMapper.getExecutionDetails(ctx)["CloudWatch Console"]).toBe(
      "https://us-east-1.console.aws.amazon.com/cloudwatch/home?region=us-east-1#alarmsV2:alarm/api-high-cpu",
    );
  });

  it("builds a China partition console URL", () => {
    const ctx = buildDetailsCtx({
      node: { configuration: { region: "cn-north-1", alarm: "api-high-cpu" } },
      execution: { outputs: undefined },
    });
    expect(getAlarmMapper.getExecutionDetails(ctx)["CloudWatch Console"]).toBe(
      "https://cn-north-1.console.amazonaws.cn/cloudwatch/home?region=cn-north-1#alarmsV2:alarm/api-high-cpu",
    );
  });

  it("falls back to configuration and node metadata when output is absent", () => {
    const ctx = buildDetailsCtx({
      node: { configuration, metadata: { alarmName: "api-high-cpu", region: "us-east-1" } },
      execution: { outputs: undefined },
    });
    const details = getAlarmMapper.getExecutionDetails(ctx);
    expect(details["Alarm Name"]).toBe("api-high-cpu");
    expect(details["Metric"]).toBe("-");
    expect(details["Condition"]).toBe("-");
    expect(details["State"]).toBe("-");
  });

  it("prefers updatedAt over createdAt for the timestamp", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(alarmOutputData)] } },
    });
    expect(getAlarmMapper.getExecutionDetails(ctx)["Retrieved At"]).toBe(
      new Date("2026-06-04T09:12:45.000Z").toLocaleString(),
    );
  });
});

// ── props ─────────────────────────────────────────────────────────────────────

describe("getAlarmMapper.props", () => {
  it("uses node name as title", () => {
    expect(getAlarmMapper.props(buildPropsContext()).title).toBe("Get Alarm Node");
  });

  it("falls back to component label when node name is empty", () => {
    const props = getAlarmMapper.props(buildPropsContext({ node: buildNode({ name: "" }) }));
    expect(props.title).toBe("CloudWatch • Get Alarm");
  });

  it("shows the alarm and the region", () => {
    const props = getAlarmMapper.props(buildPropsContext({ node: buildNode({ configuration }) }));
    expect(props.metadata?.map((item) => item.label)).toEqual(["api-high-cpu", "us-east-1"]);
  });

  it("falls back to node metadata for the alarm name", () => {
    const props = getAlarmMapper.props(
      buildPropsContext({ node: buildNode({ configuration: {}, metadata: { alarmName: "api-high-cpu" } }) }),
    );
    expect(props.metadata?.map((item) => item.label)).toEqual(["api-high-cpu"]);
  });

  it("returns empty metadata when configuration and metadata are empty", () => {
    const props = getAlarmMapper.props(buildPropsContext({ node: buildNode({ configuration: {}, metadata: {} }) }));
    expect(props.metadata).toEqual([]);
  });

  it("sets includeEmptyState when no executions", () => {
    expect(getAlarmMapper.props(buildPropsContext({ lastExecutions: [] })).includeEmptyState).toBe(true);
  });

  it("clears includeEmptyState when there is an execution", () => {
    const props = getAlarmMapper.props(buildPropsContext({ lastExecutions: [buildExecution()] }));
    expect(props.includeEmptyState).toBe(false);
  });
});

// ── eventStateRegistry ────────────────────────────────────────────────────────

describe("eventStateRegistry['cloudwatch.getAlarm']", () => {
  it("maps finished success to retrieved", () => {
    expect(eventStateRegistry["cloudwatch.getAlarm"].getState(buildExecution())).toBe("retrieved");
  });

  it("returns running when execution is in progress", () => {
    const running = buildExecution({
      state: "STATE_STARTED",
      result: "RESULT_UNSPECIFIED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_UNSPECIFIED" as ExecutionInfo["resultReason"],
    });
    expect(eventStateRegistry["cloudwatch.getAlarm"].getState(running)).toBe("running");
  });

  it("returns failed when execution fails", () => {
    const failed = buildExecution({
      state: "STATE_FINISHED",
      result: "RESULT_FAILED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_COMPONENT_FAILED" as ExecutionInfo["resultReason"],
    });
    expect(eventStateRegistry["cloudwatch.getAlarm"].getState(failed)).toBe("failed");
  });
});
