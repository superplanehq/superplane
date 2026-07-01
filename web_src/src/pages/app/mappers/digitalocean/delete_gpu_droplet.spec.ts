import { describe, expect, it } from "vitest";
import { deleteGPUDropletMapper } from "./delete_gpu_droplet";
import { buildDetailsCtx, buildOutput } from "./gpu_droplet_test_helpers";

describe("deleteGPUDropletMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => deleteGPUDropletMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => deleteGPUDropletMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("returns Deleted At without droplet ID when output data is missing", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput({})] } } });
    const details = deleteGPUDropletMapper.getExecutionDetails(ctx);
    expect(details["Deleted At"]).toBeDefined();
    expect(details["Droplet ID"]).toBe("-");
  });

  it("extracts Deleted At and Droplet ID from output", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: { default: [buildOutput({ dropletId: 99999 })] },
      },
    });
    const details = deleteGPUDropletMapper.getExecutionDetails(ctx);
    expect(details["Deleted At"]).toBeDefined();
    expect(details["Droplet ID"]).toBe("99999");
  });

  it("returns dash for Droplet ID when value is missing", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({ dropletId: undefined })] } },
    });
    const details = deleteGPUDropletMapper.getExecutionDetails(ctx);
    expect(details["Droplet ID"]).toBe("-");
  });
});
