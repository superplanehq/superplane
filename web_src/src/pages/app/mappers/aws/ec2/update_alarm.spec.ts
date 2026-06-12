import { describe, expect, it } from "vitest";

import type {
  ComponentBaseContext,
  ComponentDefinition,
  ExecutionDetailsContext,
  ExecutionInfo,
  NodeInfo,
  OutputPayload,
} from "../../types";
import { updateAlarmMapper } from "./update_alarm";
import { eventStateRegistry } from "../index";

function buildNode(overrides?: Partial<NodeInfo>): NodeInfo {
  return {
    id: "node-1",
    name: "Update Alarm Node",
    componentName: "aws.ec2.updateAlarm",
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
  name: "aws.ec2.updateAlarm",
  label: "EC2 • Update Alarm",
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
  metricName: "CPUUtilization",
  threshold: 90,
  stateValue: "OK",
  region: "us-east-1",
};

describe("updateAlarmMapper.getExecutionDetails", () => {
  it("uses configuration when output is absent", () => {
    const ctx = buildDetailsCtx({
      node: {
        configuration: {
          region: "us-east-1",
          alarm: "HighCPU",
          thresholdCondition: { threshold: 90, comparisonOperator: "GreaterThanThreshold" },
        },
      },
      execution: { outputs: undefined },
    });
    const details = updateAlarmMapper.getExecutionDetails(ctx);
    expect(details["Alarm Name"]).toBe("HighCPU");
    expect(details["Threshold"]).toBe("90");
    expect(details["Region"]).toBe("us-east-1");
  });

  it("extracts alarm fields from output", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(alarmOutputData)] } },
    });
    const details = updateAlarmMapper.getExecutionDetails(ctx);
    expect(details["Alarm Name"]).toBe("HighCPU");
    expect(details["Threshold"]).toBe("90");
    expect(details["Metric"]).toBe("CPUUtilization");
    expect(details["State"]).toBe("OK");
  });
});

describe("updateAlarmMapper.props", () => {
  it("shows alarm name and toggled field labels from configuration", () => {
    const props = updateAlarmMapper.props(
      buildPropsContext({
        node: buildNode({
          configuration: {
            alarm: "HighCPU",
            alarmDescription: "my desc",
          },
        }),
      }),
    );
    const labels = props.metadata?.map((m) => m.label) ?? [];
    expect(labels).toContain("HighCPU");
    expect(labels).toContain("Description");
  });

  it("reflects the current configuration, not stale stored metadata", () => {
    // Old publish stored ["Threshold", "Period"] but current config only has description toggled.
    const props = updateAlarmMapper.props(
      buildPropsContext({
        node: buildNode({
          configuration: { alarm: "HighCPU", alarmDescription: "new desc" },
          metadata: { updatedFields: ["Threshold", "Period"] },
        }),
      }),
    );
    const labels = props.metadata?.map((m) => m.label) ?? [];
    expect(labels).toContain("Description");
    expect(labels).not.toContain("Period");
  });

  it("removes field label immediately when toggle is turned off", () => {
    const props = updateAlarmMapper.props(
      buildPropsContext({
        node: buildNode({
          // period was previously toggled but is now absent (toggled off)
          configuration: { alarm: "HighCPU", statistic: "Average" },
          metadata: { updatedFields: ["Threshold", "Period"] },
        }),
      }),
    );
    const labels = props.metadata?.map((m) => m.label) ?? [];
    expect(labels).toContain("Statistic");
    expect(labels).not.toContain("Period");
  });

  it("caps metadata at 3 items", () => {
    const props = updateAlarmMapper.props(
      buildPropsContext({
        node: buildNode({
          configuration: {
            alarm: "HighCPU",
            thresholdCondition: { threshold: 90, comparisonOperator: "GreaterThanThreshold" },
            statistic: "Average",
            period: 300,
          },
        }),
      }),
    );
    expect((props.metadata ?? []).length).toBeLessThanOrEqual(3);
  });
});

describe("eventStateRegistry['ec2.updateAlarm']", () => {
  it("maps finished success to updated", () => {
    expect(eventStateRegistry["ec2.updateAlarm"].getState(buildExecution())).toBe("updated");
  });
});
