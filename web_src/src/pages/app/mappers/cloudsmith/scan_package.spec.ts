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

  it("shows Package Name when present in payload", () => {
    const result = { repository: "acme/production", package: "perm123", name: "my-package" };
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(result)] } },
    });
    const details = scanPackageMapper.getExecutionDetails(ctx);
    expect(details["Package Name"]).toBe("my-package");
  });

  it("omits Package Name when name is missing from payload", () => {
    const result = { repository: "acme/production", package: "perm123" };
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(result)] } },
    });
    const details = scanPackageMapper.getExecutionDetails(ctx);
    expect(details["Package Name"]).toBeUndefined();
  });
});
