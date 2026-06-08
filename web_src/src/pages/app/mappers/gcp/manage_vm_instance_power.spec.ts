import { describe, expect, it } from "vitest";
import {
  manageVMInstancePowerMapper,
  MANAGE_VM_INSTANCE_POWER_STATE_REGISTRY,
  powerStateMap,
} from "./manage_vm_instance_power";
import { buildDetailsCtx, buildExecution, buildOutput } from "./vm_mapper_test_helpers";

const powerOutput = (operation: string) =>
  buildOutput(
    {
      name: "my-vm",
      zone: "us-central1-a",
      status: "TERMINATED",
      instanceId: "123",
      operation,
    },
    `gcp.compute.vmInstance.power.${operation}`,
  );

describe("manageVMInstancePowerMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => manageVMInstancePowerMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("returns Executed At with no instance fields when output data is missing", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    const details = manageVMInstancePowerMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Instance Name"]).toBeUndefined();
  });

  it("extracts instance fields and maps the operation label", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [powerOutput("power_off")] } } });
    const details = manageVMInstancePowerMapper.getExecutionDetails(ctx);
    expect(details["Instance Name"]).toBe("my-vm");
    expect(details["Zone"]).toBe("us-central1-a");
    expect(details["Operation"]).toBe("Stop");
    expect(details["Status"]).toBe("TERMINATED");
  });
});

describe("MANAGE_VM_INSTANCE_POWER_STATE_REGISTRY.getState", () => {
  it("returns the per-operation power state on success", () => {
    const execution = buildExecution({ outputs: { default: [powerOutput("power_on")] } });
    expect(MANAGE_VM_INSTANCE_POWER_STATE_REGISTRY.getState(execution)).toBe("gcp.compute.vmInstance.power.power_on");
  });

  it("falls back to success when there is no power event", () => {
    const execution = buildExecution({ outputs: { default: [] } });
    expect(MANAGE_VM_INSTANCE_POWER_STATE_REGISTRY.getState(execution)).toBe("success");
  });

  it("propagates non-success states (e.g. failure)", () => {
    const execution = buildExecution({
      result: "RESULT_FAILED",
      resultReason: "RESULT_REASON_ERROR",
      resultMessage: "boom",
      outputs: { default: [] },
    });
    expect(MANAGE_VM_INSTANCE_POWER_STATE_REGISTRY.getState(execution)).not.toBe(
      "gcp.compute.vmInstance.power.power_on",
    );
  });

  it("defines labelled state-map entries for every operation", () => {
    expect(powerStateMap["gcp.compute.vmInstance.power.power_on"].label).toBe("STARTED");
    expect(powerStateMap["gcp.compute.vmInstance.power.power_off"].label).toBe("STOPPED");
    expect(powerStateMap["gcp.compute.vmInstance.power.reset"].label).toBe("RESET");
    expect(powerStateMap["gcp.compute.vmInstance.power.suspend"].label).toBe("SUSPENDED");
    expect(powerStateMap["gcp.compute.vmInstance.power.resume"].label).toBe("RESUMED");
  });
});
