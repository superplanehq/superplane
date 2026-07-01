import { describe, expect, it } from "vitest";
import { getVMInstanceMetricsMapper, GET_VM_INSTANCE_METRICS_STATE_REGISTRY } from "./get_vm_instance_metrics";
import { buildDetailsCtx, buildExecution, buildOutput } from "./vm_mapper_test_helpers";

const metricsData = (overrides?: Record<string, unknown>) => ({
  instanceId: "123",
  name: "my-vm",
  zone: "us-central1-a",
  lookbackPeriod: "1h",
  avgCpuUsagePercent: 23.47,
  avgNetworkInboundBytesPerSec: 10485.76,
  avgNetworkOutboundBytesPerSec: 8192.33,
  ...overrides,
});

describe("getVMInstanceMetricsMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => getVMInstanceMetricsMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("returns only Executed At when output data is missing", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    const details = getVMInstanceMetricsMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Avg CPU"]).toBeUndefined();
  });

  it("formats the metric values and resolves the lookback label", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput(metricsData())] } } });
    const details = getVMInstanceMetricsMapper.getExecutionDetails(ctx);
    expect(details["Instance Name"]).toBe("my-vm");
    expect(details["Lookback"]).toBe("Last 1 hour");
    expect(details["Avg CPU"]).toBe("23.47%");
    expect(details["Avg Inbound"]).toBe("10485.76 B/s");
    expect(details["Avg Outbound"]).toBe("8192.33 B/s");
  });

  it("renders zero metric values rather than omitting them", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(metricsData({ avgCpuUsagePercent: 0 }))] } },
    });
    const details = getVMInstanceMetricsMapper.getExecutionDetails(ctx);
    expect(details["Avg CPU"]).toBe("0%");
  });
});

describe("GET_VM_INSTANCE_METRICS_STATE_REGISTRY", () => {
  it("maps a successful execution to the FETCHED state", () => {
    const execution = buildExecution({ outputs: { default: [buildOutput(metricsData())] } });
    expect(GET_VM_INSTANCE_METRICS_STATE_REGISTRY.getState(execution)).toBe("fetched");
  });

  it("labels the fetched state as FETCHED", () => {
    expect(GET_VM_INSTANCE_METRICS_STATE_REGISTRY.stateMap["fetched"].label).toBe("FETCHED");
  });

  it("propagates non-success states", () => {
    const execution = buildExecution({
      result: "RESULT_FAILED",
      resultReason: "RESULT_REASON_ERROR",
      resultMessage: "boom",
    });
    expect(GET_VM_INSTANCE_METRICS_STATE_REGISTRY.getState(execution)).not.toBe("fetched");
  });
});
