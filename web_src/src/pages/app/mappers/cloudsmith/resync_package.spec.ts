import { describe, expect, it } from "vitest";
import { resyncPackageMapper } from "./resync_package";
import { buildDetailsCtx, buildPackageData, buildPackageOutput } from "./test_helpers";

describe("resyncPackageMapper", () => {
  it("extracts package fields from flat payload", () => {
    const pkg = buildPackageData({ status_str: "Completed" });
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildPackageOutput(pkg)] } },
    });

    const details = resyncPackageMapper.getExecutionDetails(ctx);

    expect(details["Executed At"]).toBeDefined();
    expect(details["Repository"]).toBe("production");
    expect(details["Name"]).toBe("my-package");
    expect(details["Format"]).toBe("python");
    expect(details["Status"]).toBe("Completed");
    expect(details["Package"]).toBeUndefined();
    expect(details["Operation"]).toBeUndefined();
    expect(details["Version"]).toBeUndefined();
    expect(details["Tags"]).toBeUndefined();
  });

  it("tolerates missing outputs", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => resyncPackageMapper.getExecutionDetails(ctx)).not.toThrow();
    expect(resyncPackageMapper.getExecutionDetails(ctx)["Executed At"]).toBeDefined();
  });

  it("tolerates empty default array", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => resyncPackageMapper.getExecutionDetails(ctx)).not.toThrow();
  });
});
