import { describe, expect, it } from "vitest";
import { resyncPackageMapper } from "./resync_package";
import { buildDetailsCtx, buildOutput, buildPackageOperationResult } from "./test_helpers";

describe("resyncPackageMapper", () => {
  it("extracts package result fields", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(buildPackageOperationResult())] } },
    });

    const details = resyncPackageMapper.getExecutionDetails(ctx);

    expect(details["Executed At"]).toBeDefined();
    expect(details["Repository"]).toBe("acme/production");
    expect(details["Name"]).toBe("billing-api");
    expect(details["Format"]).toBe("docker");
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
});
