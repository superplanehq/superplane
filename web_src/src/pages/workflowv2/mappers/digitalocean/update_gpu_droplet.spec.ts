import { describe, expect, it } from "vitest";
import { updateGPUDropletMapper } from "./update_gpu_droplet";
import { buildDetailsCtx, buildDropletData, buildOutput } from "./gpu_droplet_test_helpers";

describe("updateGPUDropletMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => updateGPUDropletMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => updateGPUDropletMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("returns Executed At without droplet fields when output data is missing", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput({})] } } });
    const details = updateGPUDropletMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Droplet ID"]).toBe("-");
  });

  it("extracts all update fields from droplet output", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(buildDropletData())] } },
    });
    const details = updateGPUDropletMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Droplet ID"]).toBe("123456");
    expect(details["Name"]).toBe("gpu-droplet-1");
    expect(details["Status"]).toBe("active");
    expect(details["Region"]).toBe("New York 3");
    expect(details["GPU Size"]).toBe("gpu-h100x1-80gb");
    expect(details["Memory"]).toBe("245760 MB");
    expect(details["vCPUs"]).toBe("20");
    expect(details["Disk"]).toBe("480 GB");
    expect(details["IP Address"]).toBe("1.2.3.4");
  });

  it("omits IP Address when no public IPv4 network", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [buildOutput(buildDropletData({ networks: { v4: [] } }))],
        },
      },
    });
    const details = updateGPUDropletMapper.getExecutionDetails(ctx);
    expect(details["IP Address"]).toBeUndefined();
  });
});
