import { describe, expect, it } from "bun:test";
import { createCloudWatchFixtures } from "./test-support";

import { deleteAlarmMapper } from "./delete_alarm";
import { eventStateRegistry } from "../index";

// ── Helpers ───────────────────────────────────────────────────────────────────

const { buildNode, buildOutput, buildExecution, buildDetailsCtx, buildPropsContext } = createCloudWatchFixtures({
  nodeName: "Delete Alarm Node",
  componentName: "aws.cloudwatch.deleteAlarm",
  label: "CloudWatch • Delete Alarm",
  outputType: "aws.cloudwatch.alarm.deleted",
  createdAt: "2026-06-04T09:14:05.000Z",
  updatedAt: "2026-06-04T09:14:11.000Z",
});

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
  it("shows the alarm and the region", () => {
    const props = deleteAlarmMapper.props(buildPropsContext({ node: buildNode({ configuration }) }));
    expect(props.metadata?.map((item) => item.label)).toEqual(["api-high-cpu", "us-east-1"]);
  });

  it("returns empty metadata when configuration and metadata are empty", () => {
    const props = deleteAlarmMapper.props(buildPropsContext({ node: buildNode({ configuration: {}, metadata: {} }) }));
    expect(props.metadata).toEqual([]);
  });
});

// ── eventStateRegistry ────────────────────────────────────────────────────────

describe("eventStateRegistry['cloudwatch.deleteAlarm']", () => {
  it("maps finished success to deleted", () => {
    expect(eventStateRegistry["cloudwatch.deleteAlarm"].getState(buildExecution())).toBe("deleted");
  });
});
