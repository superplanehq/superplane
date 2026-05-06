import { describe, expect, it } from "vitest";
import { createGPUDropletMapper } from "./create_gpu_droplet";
import { buildDetailsCtx, buildDropletData, buildOutput } from "./gpu_droplet_test_helpers";

describe("createGPUDropletMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => createGPUDropletMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => createGPUDropletMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("returns Created At without droplet fields when output data is missing", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput({})] } } });
    const details = createGPUDropletMapper.getExecutionDetails(ctx);
    expect(details["Created At"]).toBeDefined();
    expect(details["Droplet ID"]).toBe("-");
    expect(details["Name"]).toBe("-");
  });

  it("extracts all droplet fields from output", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(buildDropletData())] } },
    });
    const details = createGPUDropletMapper.getExecutionDetails(ctx);
    expect(details["Created At"]).toBeDefined();
    expect(details["Droplet ID"]).toBe("123456");
    expect(details["Name"]).toBe("gpu-droplet-1");
    expect(details["Region"]).toBe("New York 3");
    expect(details["GPU Size"]).toBe("gpu-h100x1-80gb");
    expect(details["OS"]).toBe("Ubuntu 22.04 (LTS) x64");
    expect(details["IP Address"]).toBe("1.2.3.4");
  });

  it("omits IP Address when no public IPv4 network", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [buildOutput(buildDropletData({ networks: { v4: [{ type: "private", ip_address: "10.0.0.1" }] } }))],
        },
      },
    });
    const details = createGPUDropletMapper.getExecutionDetails(ctx);
    expect(details["IP Address"]).toBeUndefined();
  });

  it("falls back to slug for region and OS when name is absent", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildOutput(
              buildDropletData({
                region: { slug: "nyc3" },
                image: { slug: "ubuntu-22-04-x64" },
              }),
            ),
          ],
        },
      },
    });
    const details = createGPUDropletMapper.getExecutionDetails(ctx);
    expect(details["Region"]).toBe("nyc3");
    expect(details["OS"]).toBe("ubuntu-22-04-x64");
  });
});
