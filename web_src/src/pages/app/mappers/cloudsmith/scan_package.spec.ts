import { describe, expect, it } from "bun:test";
import { scanPackageMapper } from "./scan_package";
import { buildDetailsCtx, buildOutput } from "./test_helpers";

describe("scanPackageMapper.getExecutionDetails", () => {
  it("returns Executed At", () => {
    const ctx = buildDetailsCtx();
    const details = scanPackageMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
  });

  it("shows Repository and Package when present in payload", () => {
    const result = { repository: "acme/production", package: "perm123" };
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(result)] } },
    });
    const details = scanPackageMapper.getExecutionDetails(ctx);
    expect(details["Repository"]).toBe("acme/production");
    expect(details["Package"]).toBe("perm123");
  });

  it("omits Repository and Package when missing from payload", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput({})] } },
    });
    const details = scanPackageMapper.getExecutionDetails(ctx);
    expect(details["Repository"]).toBeUndefined();
    expect(details["Package"]).toBeUndefined();
  });
});
