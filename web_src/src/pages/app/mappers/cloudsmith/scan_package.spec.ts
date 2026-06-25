import { describe, expect, it } from "vitest";
import { scanPackageMapper } from "./scan_package";
import { buildDetailsCtx, buildOutput } from "./test_helpers";

describe("scanPackageMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => scanPackageMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => scanPackageMapper.getExecutionDetails(ctx)).not.toThrow();
  });

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
