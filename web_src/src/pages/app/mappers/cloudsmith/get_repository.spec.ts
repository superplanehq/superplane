import { describe, expect, it } from "vitest";
import { getRepositoryMapper } from "./get_repository";
import { buildDetailsCtx, buildOutput, buildRepositoryData } from "./test_helpers";

describe("getRepositoryMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => getRepositoryMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => getRepositoryMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("returns Executed At without repository fields when output data is missing", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildOutput(undefined)] } } });
    const details = getRepositoryMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Name"]).toBeUndefined();
  });

  it("extracts the key repository fields", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(buildRepositoryData())] } },
    });
    const details = getRepositoryMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Name"]).toBe("Production");
    expect(details["Namespace"]).toBe("acme");
    expect(details["Size"]).toBe("500.0 MB");
    expect(details["Packages"]).toBe("312");
    expect(details["Downloads"]).toBe("18234");
    expect(details["Quarantined Packages"]).toBe("1");
    expect(details["Policy Violations"]).toBe("2");
  });

  it("shows at most six fields when compliance counts are zero", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [buildOutput(buildRepositoryData({ num_quarantined_packages: 0, num_policy_violated_packages: 0 }))],
        },
      },
    });
    const details = getRepositoryMapper.getExecutionDetails(ctx);
    expect(Object.keys(details)).toHaveLength(6);
  });

  it("falls back to raw byte size when size_str is missing", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildOutput(buildRepositoryData({ size_str: undefined, size: 1024 }))] } },
    });
    const details = getRepositoryMapper.getExecutionDetails(ctx);
    expect(details["Size"]).toBe("1024 bytes");
  });

  it("omits compliance counts when they are zero", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [buildOutput(buildRepositoryData({ num_quarantined_packages: 0, num_policy_violated_packages: 0 }))],
        },
      },
    });
    const details = getRepositoryMapper.getExecutionDetails(ctx);
    expect(details["Quarantined Packages"]).toBeUndefined();
    expect(details["Policy Violations"]).toBeUndefined();
  });
});
