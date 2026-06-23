import { describe, expect, it } from "vitest";
import { promotePackageMapper } from "./promote_package";
import { buildDetailsCtx, buildPackageData, buildPackageOutput } from "./test_helpers";

describe("promotePackageMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => promotePackageMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => promotePackageMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("returns Executed At without package fields when output data is missing", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildPromoteOutput(undefined)] } },
    });
    const details = promotePackageMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Package"]).toBeUndefined();
  });

  it("extracts key promoted package fields", () => {
    const pkg = buildPackageData({
      name: "my-package",
      version: "1.2.0",
      repository: "production",
      status_str: "Available",
      self_webapp_url: "https://app.cloudsmith.com/acme/r/production/docker/my-package/1.2.0/",
    });
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildPromoteOutput(pkg)] } },
    });
    const details = promotePackageMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Package"]).toBe("my-package");
    expect(details["Version"]).toBe("1.2.0");
    expect(details["Destination"]).toBe("production");
    expect(details["Status"]).toBe("Available");
    expect(details["URL"]).toBe("https://app.cloudsmith.com/acme/r/production/docker/my-package/1.2.0/");
  });

  it("includes URL from self_webapp_url", () => {
    const pkg = buildPackageData({
      self_webapp_url: "https://app.cloudsmith.com/acme/r/prod/docker/pkg/1.0.0/abc",
    });
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildPromoteOutput(pkg)] } },
    });
    const details = promotePackageMapper.getExecutionDetails(ctx);
    expect(details["URL"]).toBe("https://app.cloudsmith.com/acme/r/prod/docker/pkg/1.0.0/abc");
  });

  it("omits URL when self_webapp_url is missing", () => {
    const pkg = buildPackageData({ self_webapp_url: undefined });
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildPromoteOutput(pkg)] } },
    });
    const details = promotePackageMapper.getExecutionDetails(ctx);
    expect(details["URL"]).toBeUndefined();
  });

  it("does not include Size or Security Scan in details", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildPromoteOutput(buildPackageData())] } },
    });
    const details = promotePackageMapper.getExecutionDetails(ctx);
    expect(details["Size"]).toBeUndefined();
    expect(details["Security Scan"]).toBeUndefined();
  });
});

function buildPromoteOutput(data: unknown) {
  return buildPackageOutput(data, "cloudsmith.package.promoted");
}
