import { describe, expect, it } from "vitest";
import { tagPackageMapper } from "./tag_package";
import { buildDetailsCtx, buildPackageData, buildPackageOutput } from "./test_helpers";

describe("tagPackageMapper", () => {
  it("extracts package fields from flat payload", () => {
    const pkg = buildPackageData({ status_str: "Completed" });
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildPackageOutput(pkg)] } },
    });

    const details = tagPackageMapper.getExecutionDetails(ctx);

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
    expect(() => tagPackageMapper.getExecutionDetails(ctx)).not.toThrow();
    expect(tagPackageMapper.getExecutionDetails(ctx)["Executed At"]).toBeDefined();
  });

  it("tolerates empty default array", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => tagPackageMapper.getExecutionDetails(ctx)).not.toThrow();
  });
});
