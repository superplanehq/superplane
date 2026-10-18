import { describe, expect, it } from "bun:test";
import { createCloudWatchFixtures } from "./test-support";

import { updateAlarmMapper } from "./update_alarm";
import { eventStateRegistry } from "../index";

// ── Helpers ───────────────────────────────────────────────────────────────────

const { buildNode, buildOutput, buildExecution, buildDetailsCtx, buildPropsContext } = createCloudWatchFixtures({
  nodeName: "Update Alarm Node",
  componentName: "aws.cloudwatch.updateAlarm",
  label: "CloudWatch • Update Alarm",
  outputType: "aws.cloudwatch.alarm",
  createdAt: "2026-06-04T09:12:40.000Z",
  updatedAt: "2026-06-04T09:12:45.000Z",
});

const alarmOutputData = {
  alarmName: "api-high-cpu",
  namespace: "AWS/EC2",
  metricName: "CPUUtilization",
  statistic: "Average",
  comparisonOperator: "GreaterThanThreshold",
  threshold: 90,
  stateValue: "OK",
  region: "us-east-1",
  consoleUrl: "https://us-east-1.console.aws.amazon.com/cloudwatch/home?region=us-east-1#alarmsV2:alarm/api-high-cpu",
};

const thresholdConfiguration = {
  region: "us-east-1",
  alarm: "api-high-cpu",
  thresholdCondition: { threshold: 90, comparisonOperator: "GreaterThanThreshold" },
};

// ── getExecutionDetails ───────────────────────────────────────────────────────

describe("updateAlarmMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => updateAlarmMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => updateAlarmMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("shows at most six rows, starting with the timestamp", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(alarmOutputData)] } },
    });
    const keys = Object.keys(updateAlarmMapper.getExecutionDetails(ctx));
    expect(keys.length).toBeLessThanOrEqual(6);
    expect(keys[0]).toBe("Updated At");
  });

  it("extracts alarm fields from output", () => {
    const ctx = buildDetailsCtx({
      node: { configuration: thresholdConfiguration },
      execution: { outputs: { default: [buildOutput(alarmOutputData)] } },
    });
    const details = updateAlarmMapper.getExecutionDetails(ctx);
    expect(details["Alarm Name"]).toBe("api-high-cpu");
    expect(details["Metric"]).toBe("AWS/EC2 / CPUUtilization");
    expect(details["Condition"]).toBe("Average > 90");
    expect(details["Updated"]).toBe("Threshold");
  });

  it("lists every enabled field in the updated row", () => {
    const ctx = buildDetailsCtx({
      node: {
        configuration: {
          ...thresholdConfiguration,
          period: 60,
          alarmActions: ["arn:aws:sns:us-east-1:123456789012:ops-alerts"],
        },
      },
      execution: { outputs: { default: [buildOutput(alarmOutputData)] } },
    });
    expect(updateAlarmMapper.getExecutionDetails(ctx)["Updated"]).toBe("Threshold, Period, Alarm Actions");
  });

  it("includes the EC2 action in the updated row", () => {
    const ctx = buildDetailsCtx({
      node: { configuration: { region: "us-east-1", alarm: "api-status-check", ec2Action: "recover" } },
      execution: { outputs: { default: [buildOutput(alarmOutputData)] } },
    });
    expect(updateAlarmMapper.getExecutionDetails(ctx)["Updated"]).toBe("EC2 Action");
  });

  it("shows a dash when nothing is enabled", () => {
    const ctx = buildDetailsCtx({
      node: { configuration: { region: "us-east-1", alarm: "api-high-cpu" } },
      execution: { outputs: { default: [buildOutput(alarmOutputData)] } },
    });
    expect(updateAlarmMapper.getExecutionDetails(ctx)["Updated"]).toBe("-");
  });

  it("uses the console URL from the output", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(alarmOutputData)] } },
    });
    expect(updateAlarmMapper.getExecutionDetails(ctx)["CloudWatch Console"]).toBe(alarmOutputData.consoleUrl);
  });

  it("falls back to configuration and node metadata when output is absent", () => {
    const ctx = buildDetailsCtx({
      node: {
        configuration: thresholdConfiguration,
        metadata: { alarmName: "api-high-cpu", region: "us-east-1" },
      },
      execution: { outputs: undefined },
    });
    const details = updateAlarmMapper.getExecutionDetails(ctx);
    expect(details["Alarm Name"]).toBe("api-high-cpu");
    expect(details["Metric"]).toBe("-");
    expect(details["Condition"]).toBe("-");
    expect(details["CloudWatch Console"]).toBe(
      "https://us-east-1.console.aws.amazon.com/cloudwatch/home?region=us-east-1#alarmsV2:alarm/api-high-cpu",
    );
  });

  it("prefers updatedAt over createdAt for the timestamp", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(alarmOutputData)] } },
    });
    expect(updateAlarmMapper.getExecutionDetails(ctx)["Updated At"]).toBe(
      new Date("2026-06-04T09:12:45.000Z").toLocaleString(),
    );
  });
});

// ── props ─────────────────────────────────────────────────────────────────────

describe("updateAlarmMapper.props", () => {
  it("shows the alarm and the first two enabled fields", () => {
    const props = updateAlarmMapper.props(
      buildPropsContext({
        node: buildNode({
          configuration: { ...thresholdConfiguration, period: 60, statistic: "Maximum" },
        }),
      }),
    );
    const labels = props.metadata?.map((item) => item.label) ?? [];
    expect(labels).toEqual(["api-high-cpu", "Threshold", "Statistic"]);
  });

  it("returns empty metadata when configuration and metadata are empty", () => {
    const props = updateAlarmMapper.props(buildPropsContext({ node: buildNode({ configuration: {}, metadata: {} }) }));
    expect(props.metadata).toEqual([]);
  });
});

// ── eventStateRegistry ────────────────────────────────────────────────────────

describe("eventStateRegistry['cloudwatch.updateAlarm']", () => {
  it("maps finished success to updated", () => {
    expect(eventStateRegistry["cloudwatch.updateAlarm"].getState(buildExecution())).toBe("updated");
  });
});
