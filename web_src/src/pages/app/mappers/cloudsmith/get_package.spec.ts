import { describe, expect, it } from "vitest";
import { getPackageMapper } from "./get_package";
import { buildDetailsCtx, buildPackageData, buildPackageOutput } from "./test_helpers";

describe("getPackageMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => getPackageMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => getPackageMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("returns Executed At without package fields when output data is missing", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildPackageOutput(undefined)] } },
    });
    const details = getPackageMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Package"]).toBeUndefined();
  });

  it("extracts key package metadata fields", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildPackageOutput(buildPackageData())] } },
    });
    const details = getPackageMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Package"]).toBe("my-package");
    expect(details["Format"]).toBe("python");
    expect(details["Repository"]).toBe("acme/production");
    expect(details["Size"]).toBe("512.0 KB");
    expect(details["URL"]).toBe(
      "https://cloudsmith.io/~acme/repos/production/packages/detail/python/my-package/1.0.0/",
    );
  });

  it("does not include Version, Status, SHA-256 or Uploaded At", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildPackageOutput(buildPackageData())] } },
    });
    const details = getPackageMapper.getExecutionDetails(ctx);
    expect(details["Version"]).toBeUndefined();
    expect(details["Status"]).toBeUndefined();
    expect(details["SHA-256"]).toBeUndefined();
    expect(details["Uploaded At"]).toBeUndefined();
  });

  it("falls back to raw byte size when size_str is missing", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildPackageOutput(buildPackageData({ size_str: undefined, size: 2048 }))] } },
    });
    const details = getPackageMapper.getExecutionDetails(ctx);
    expect(details["Size"]).toBe("2048 bytes");
  });

  it("omits URL when self_html_url is missing", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildPackageOutput(buildPackageData({ self_html_url: undefined }))] } },
    });
    const details = getPackageMapper.getExecutionDetails(ctx);
    expect(details["URL"]).toBeUndefined();
  });

  it("omits Repository when namespace is missing", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: { default: [buildPackageOutput(buildPackageData({ namespace: undefined }))] },
      },
    });
    const details = getPackageMapper.getExecutionDetails(ctx);
    expect(details["Repository"]).toBeUndefined();
  });
});
