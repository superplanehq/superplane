import { describe, expect, it } from "vitest";
import { quarantinePackageMapper, QUARANTINE_PACKAGE_STATE_REGISTRY } from "./quarantine_package";
import { buildDetailsCtx, buildExecution, buildPackageData, buildPackageOutput } from "./test_helpers";

describe("quarantinePackageMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => quarantinePackageMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => quarantinePackageMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("returns Executed At without package fields when data is missing", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildPackageOutput(undefined)] } },
    });
    const details = quarantinePackageMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Name"]).toBeUndefined();
  });

  it("extracts package details without URL", () => {
    const pkg = buildPackageData({ status_str: "Quarantined" });
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildPackageOutput(pkg)] } },
    });
    const details = quarantinePackageMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Name"]).toBe("my-package");
    expect(details["Version"]).toBe("1.0.0");
    expect(details["Format"]).toBe("python");
    expect(details["Status"]).toBe("Quarantined");
    expect(details["URL"]).toBeUndefined();
  });
});

describe("QUARANTINE_PACKAGE_STATE_REGISTRY", () => {
  it("returns quarantined event type when quarantine output is present", () => {
    const execution = buildExecution({
      result: "RESULT_PASSED",
      outputs: {
        default: [buildPackageOutput(buildPackageData(), "cloudsmith.package.quarantined")],
      },
    });
    const state = QUARANTINE_PACKAGE_STATE_REGISTRY.getState(execution);
    expect(state).toBe("cloudsmith.package.quarantined");
  });

  it("returns released event type when release output is present", () => {
    const execution = buildExecution({
      result: "RESULT_PASSED",
      outputs: {
        default: [buildPackageOutput(buildPackageData({ status_str: "Available" }), "cloudsmith.package.released")],
      },
    });
    const state = QUARANTINE_PACKAGE_STATE_REGISTRY.getState(execution);
    expect(state).toBe("cloudsmith.package.released");
  });

  it("returns failed state on failed execution", () => {
    const execution = buildExecution({ result: "RESULT_FAILED" });
    const state = QUARANTINE_PACKAGE_STATE_REGISTRY.getState(execution);
    expect(state).toBe("failed");
  });
});
