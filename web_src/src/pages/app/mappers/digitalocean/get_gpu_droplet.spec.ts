import { describe, expect, it } from "vitest";
import { getGPUDropletMapper } from "./get_gpu_droplet";
import { buildDetailsCtx, buildDropletData, buildOutput } from "./gpu_droplet_test_helpers";

describe("getGPUDropletMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => getGPUDropletMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => getGPUDropletMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("returns Executed At without droplet fields when output data is missing", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput({})] } } });
    const details = getGPUDropletMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Droplet ID"]).toBe("-");
  });

  it("extracts all droplet fields including status, memory, vCPUs, disk, tags", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(buildDropletData())] } },
    });
    const details = getGPUDropletMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Droplet ID"]).toBe("123456");
    expect(details["Name"]).toBe("gpu-droplet-1");
    expect(details["Status"]).toBe("active");
    expect(details["Region"]).toBe("New York 3");
    expect(details["GPU Size"]).toBe("gpu-h100x1-80gb");
    expect(details["OS"]).toBe("Ubuntu 22.04 (LTS) x64");
    expect(details["Memory"]).toBe("245760 MB");
    expect(details["vCPUs"]).toBe("20");
    expect(details["Disk"]).toBe("480 GB");
    expect(details["IP Address"]).toBe("1.2.3.4");
    expect(details["Tags"]).toBe("gpu, ml");
  });

  it("omits Tags when tags array is empty", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(buildDropletData({ tags: [] }))] } },
    });
    const details = getGPUDropletMapper.getExecutionDetails(ctx);
    expect(details["Tags"]).toBeUndefined();
  });

  it("omits IP Address when no public IPv4 network", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [buildOutput(buildDropletData({ networks: { v4: [] } }))],
        },
      },
    });
    const details = getGPUDropletMapper.getExecutionDetails(ctx);
    expect(details["IP Address"]).toBeUndefined();
  });
});
