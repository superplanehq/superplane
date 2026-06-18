import { describe, expect, it } from "vitest";
import { tagPackageMapper } from "./tag_package";
import { buildDetailsCtx, buildOutput, buildPackageOperationResult } from "./test_helpers";

describe("tagPackageMapper", () => {
  it("extracts package result fields", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(buildPackageOperationResult())] } },
    });

    const details = tagPackageMapper.getExecutionDetails(ctx);

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
    expect(() => tagPackageMapper.getExecutionDetails(ctx)).not.toThrow();
    expect(tagPackageMapper.getExecutionDetails(ctx)["Executed At"]).toBeDefined();
  });
});
