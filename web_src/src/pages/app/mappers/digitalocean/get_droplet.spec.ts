import { describe, expect, it } from "bun:test";
import { getDropletMapper } from "./get_droplet";
import { buildDetailsCtx, buildDropletData, buildOutput } from "./gpu_droplet_test_helpers";

describe("getDropletMapper.getExecutionDetails", () => {
  it("joins tags when the payload stores an array", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(buildDropletData({ tags: ["web", "prod"] }))] } },
    });
    const details = getDropletMapper.getExecutionDetails(ctx);
    expect(details["Tags"]).toBe("web, prod");
  });

  it("omits Tags when tags is a truthy non-array value", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(buildDropletData({ tags: "web" }))] } },
    });
    expect(() => getDropletMapper.getExecutionDetails(ctx)).not.toThrow();
    expect(getDropletMapper.getExecutionDetails(ctx)["Tags"]).toBeUndefined();
  });
});
