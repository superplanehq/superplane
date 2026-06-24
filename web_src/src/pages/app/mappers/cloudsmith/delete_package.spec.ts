import { describe, expect, it } from "vitest";
import { deletePackageMapper } from "./delete_package";
import { buildDetailsCtx, buildOutput, buildPackageOperationResult } from "./test_helpers";

describe("deletePackageMapper", () => {
  it("extracts package result fields", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(buildPackageOperationResult())] } },
    });

    const details = deletePackageMapper.getExecutionDetails(ctx);

    expect(details["Executed At"]).toBeDefined();
    expect(details["Repository"]).toBe("acme/production");
    expect(details["Package"]).toBe("pkg_123");
    expect(details["Operation"]).toBe("deleted");
    expect(details["Name"]).toBe("billing-api");
    expect(details["Format"]).toBe("docker");
    expect(details["Status"]).toBe("Completed");
    expect(details["Tags"]).toBe("latest, production");
  });

  it("shows identifier even when no package object is emitted", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [buildOutput(buildPackageOperationResult({ data: undefined }))],
        },
      },
    });

    const details = deletePackageMapper.getExecutionDetails(ctx);

    expect(details["Repository"]).toBe("acme/production");
    expect(details["Package"]).toBe("pkg_123");
    expect(details["Operation"]).toBe("deleted");
    expect(details["Name"]).toBeUndefined();
  });

  it("tolerates missing outputs", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => deletePackageMapper.getExecutionDetails(ctx)).not.toThrow();
    expect(deletePackageMapper.getExecutionDetails(ctx)["Executed At"]).toBeDefined();
  });
});
