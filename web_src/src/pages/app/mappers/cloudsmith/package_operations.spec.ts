import { describe, expect, it } from "vitest";
import { deletePackageMapper, resyncPackageMapper, tagPackageMapper } from "./package_operations";
import { buildDetailsCtx, buildOutput, buildPackageOperationResult } from "./test_helpers";

describe("package operation mappers", () => {
  it("resyncPackageMapper extracts package result fields", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(buildPackageOperationResult())] } },
    });

    const details = resyncPackageMapper.getExecutionDetails(ctx);

    expect(details["Executed At"]).toBeDefined();
    expect(details["Repository"]).toBe("acme/production");
    expect(details["Package"]).toBe("pkg_123");
    expect(details["Operation"]).toBe("resynced");
    expect(details["Name"]).toBe("billing-api");
    expect(details["Version"]).toBe("1.2.3");
    expect(details["Format"]).toBe("docker");
    expect(details["Status"]).toBe("Completed");
    expect(details["Tags"]).toBe("latest, production");
  });

  it("tagPackageMapper tolerates missing outputs", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });

    expect(() => tagPackageMapper.getExecutionDetails(ctx)).not.toThrow();
    expect(tagPackageMapper.getExecutionDetails(ctx)["Executed At"]).toBeDefined();
  });

  it("deletePackageMapper shows identifier even when no package object is emitted", () => {
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
});
