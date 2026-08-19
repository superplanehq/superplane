import { describe, expect, it } from "vitest";

import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
} from "../../types";
import { deleteAlarmMapper } from "./delete_alarm";
import { eventStateRegistry } from "../index";

// ── Helpers ───────────────────────────────────────────────────────────────────

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Delete Alarm Node",
    componentName: "aws.cloudwatch.deleteAlarm",
    isCollapsed: false,
    configuration: {},
    metadata: {},
    ...overrides,
  };
}

function buildOutput(data: unknown): OutputPayload {
  return {
    type: "aws.cloudwatch.alarm.deleted",
    timestamp: new Date().toISOString(),
    data,
  };
}

function buildExecution(overrides?: Partial<ExecutionInfo>): ExecutionInfo {
  return {
    id: "exec-1",
    createdAt: "2026-06-04T09:14:05.000Z",
    updatedAt: "2026-06-04T09:14:11.000Z",
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
  name: "aws.cloudwatch.deleteAlarm",
  label: "CloudWatch • Delete Alarm",
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

const deletedAlarmOutputData = {
  alarmName: "api-high-cpu",
  namespace: "AWS/EC2",
  metricName: "CPUUtilization",
  statistic: "Average",
  comparisonOperator: "GreaterThanThreshold",
  threshold: 90,
  stateValue: "OK",
  region: "us-east-1",
  deleted: true,
};

const configuration = { region: "us-east-1", alarm: "api-high-cpu" };

// ── getExecutionDetails ───────────────────────────────────────────────────────

describe("deleteAlarmMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => deleteAlarmMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => deleteAlarmMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("shows at most six rows, starting with the timestamp", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(deletedAlarmOutputData)] } },
    });
    const keys = Object.keys(deleteAlarmMapper.getExecutionDetails(ctx));
    expect(keys.length).toBeLessThanOrEqual(6);
    expect(keys[0]).toBe("Deleted At");
  });

  it("describes the alarm that was deleted", () => {
    const ctx = buildDetailsCtx({
      node: { configuration },
      execution: { outputs: { default: [buildOutput(deletedAlarmOutputData)] } },
    });
    const details = deleteAlarmMapper.getExecutionDetails(ctx);
    expect(details["Alarm Name"]).toBe("api-high-cpu");
    expect(details["Metric"]).toBe("AWS/EC2 / CPUUtilization");
    expect(details["Condition"]).toBe("Average > 90");
    expect(details["Region"]).toBe("us-east-1");
    expect(details["Deleted"]).toBe("Yes");
  });

  it("never offers a console link, since the alarm is gone", () => {
    const ctx = buildDetailsCtx({
      node: { configuration },
      execution: { outputs: { default: [buildOutput(deletedAlarmOutputData)] } },
    });
    expect(Object.keys(deleteAlarmMapper.getExecutionDetails(ctx))).not.toContain("CloudWatch Console");
  });

  it("falls back to configuration and node metadata when output is absent", () => {
    const ctx = buildDetailsCtx({
      node: { configuration: {}, metadata: { alarmName: "api-high-cpu", region: "us-east-1" } },
      execution: { outputs: undefined },
    });
    const details = deleteAlarmMapper.getExecutionDetails(ctx);
    expect(details["Alarm Name"]).toBe("api-high-cpu");
    expect(details["Region"]).toBe("us-east-1");
    expect(details["Metric"]).toBe("-");
    expect(details["Condition"]).toBe("-");
    expect(details["Deleted"]).toBe("-");
  });

  it("prefers updatedAt over createdAt for the timestamp", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(deletedAlarmOutputData)] } },
    });
    expect(deleteAlarmMapper.getExecutionDetails(ctx)["Deleted At"]).toBe(
      new Date("2026-06-04T09:14:11.000Z").toLocaleString(),
    );
  });
});

// ── props ─────────────────────────────────────────────────────────────────────

describe("deleteAlarmMapper.props", () => {
  it("uses node name as title", () => {
    expect(deleteAlarmMapper.props(buildPropsContext()).title).toBe("Delete Alarm Node");
  });

  it("falls back to component label when node name is empty", () => {
    const props = deleteAlarmMapper.props(buildPropsContext({ node: buildNode({ name: "" }) }));
    expect(props.title).toBe("CloudWatch • Delete Alarm");
  });

  it("shows the alarm and the region", () => {
    const props = deleteAlarmMapper.props(buildPropsContext({ node: buildNode({ configuration }) }));
    expect(props.metadata?.map((item) => item.label)).toEqual(["api-high-cpu", "us-east-1"]);
  });

  it("returns empty metadata when configuration and metadata are empty", () => {
    const props = deleteAlarmMapper.props(buildPropsContext({ node: buildNode({ configuration: {}, metadata: {} }) }));
    expect(props.metadata).toEqual([]);
  });

  it("sets includeEmptyState when no executions", () => {
    expect(deleteAlarmMapper.props(buildPropsContext({ lastExecutions: [] })).includeEmptyState).toBe(true);
  });

  it("clears includeEmptyState when there is an execution", () => {
    const props = deleteAlarmMapper.props(buildPropsContext({ lastExecutions: [buildExecution()] }));
    expect(props.includeEmptyState).toBe(false);
  });
});

// ── eventStateRegistry ────────────────────────────────────────────────────────

describe("eventStateRegistry['cloudwatch.deleteAlarm']", () => {
  it("maps finished success to deleted", () => {
    expect(eventStateRegistry["cloudwatch.deleteAlarm"].getState(buildExecution())).toBe("deleted");
  });

  it("returns running when execution is in progress", () => {
    const running = buildExecution({
      state: "STATE_STARTED",
      result: "RESULT_UNSPECIFIED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_UNSPECIFIED" as ExecutionInfo["resultReason"],
    });
    expect(eventStateRegistry["cloudwatch.deleteAlarm"].getState(running)).toBe("running");
  });

  it("returns failed when execution fails", () => {
    const failed = buildExecution({
      state: "STATE_FINISHED",
      result: "RESULT_FAILED" as ExecutionInfo["result"],
      resultReason: "RESULT_REASON_COMPONENT_FAILED" as ExecutionInfo["resultReason"],
    });
    expect(eventStateRegistry["cloudwatch.deleteAlarm"].getState(failed)).toBe("failed");
  });
});
