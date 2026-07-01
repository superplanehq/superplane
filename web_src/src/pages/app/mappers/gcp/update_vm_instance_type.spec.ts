import { describe, expect, it } from "vitest";
import { updateVMInstanceTypeMapper } from "./update_vm_instance_type";
import { buildDetailsCtx, buildOutput } from "./vm_mapper_test_helpers";

describe("updateVMInstanceTypeMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => updateVMInstanceTypeMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("returns only Executed At when output data is missing", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    const details = updateVMInstanceTypeMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Machine Type"]).toBeUndefined();
  });

  it("extracts the updated instance fields", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput({
              name: "my-vm",
              zone: "us-central1-a",
              status: "RUNNING",
              machineType: "n2-standard-4",
            }),
          ],
        },
      },
    });
    const details = updateVMInstanceTypeMapper.getExecutionDetails(ctx);
    expect(details["Instance Name"]).toBe("my-vm");
    expect(details["Zone"]).toBe("us-central1-a");
    expect(details["Machine Type"]).toBe("n2-standard-4");
    expect(details["Status"]).toBe("RUNNING");
  });
});
