import { describe, expect, it } from "vitest";
import { listPackagesMapper } from "./list_packages";
import { buildDetailsCtx, buildPackageData, buildPackageOutput } from "./test_helpers";

describe("listPackagesMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => listPackagesMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => listPackagesMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("returns Executed At when output data is missing", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildPackageOutput(undefined)] } },
    });
    const details = listPackagesMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Packages Found"]).toBeUndefined();
  });

  it("shows count of packages found", () => {
    const pkg = buildPackageData();
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [buildListPackageOutput(pkg), buildListPackageOutput(buildPackageData({ name: "other-pkg" }))],
        },
      },
    });
    const details = listPackagesMapper.getExecutionDetails(ctx);
    expect(details["Packages Found"]).toBe("2");
  });

  it("shows format and status from first package", () => {
    const pkg = buildPackageData({ format: "docker", status_str: "Available" });
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildListPackageOutput(pkg)] } },
    });
    const details = listPackagesMapper.getExecutionDetails(ctx);
    expect(details["Format"]).toBe("docker");
    expect(details["Status"]).toBe("Available");
  });

  it("shows security scan status", () => {
    const pkg = buildPackageData({ security_scan_status: "No Vulnerabilities Found" });
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildListPackageOutput(pkg)] } },
    });
    const details = listPackagesMapper.getExecutionDetails(ctx);
    expect(details["Security Scan"]).toBe("No Vulnerabilities Found");
  });

  it("does not include Package name or Version in details", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildListPackageOutput(buildPackageData())] } },
    });
    const details = listPackagesMapper.getExecutionDetails(ctx);
    expect(details["Package"]).toBeUndefined();
    expect(details["Version"]).toBeUndefined();
  });
});

function buildListPackageOutput(data: unknown) {
  return buildPackageOutput(data, "cloudsmith.packages.listed");
}
