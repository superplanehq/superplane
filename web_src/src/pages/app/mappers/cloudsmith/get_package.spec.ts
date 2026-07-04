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
    const pkg = buildPackageData({ self_webapp_url: "https://app.cloudsmith.com/acme/r/production/" });
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildPackageOutput(pkg)] } },
    });
    const details = getPackageMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Package"]).toBe("my-package");
    expect(details["Format"]).toBe("python");
    expect(details["Size"]).toBe("512.0 KB");
    expect(details["URL"]).toBe("https://app.cloudsmith.com/acme/r/production/");
  });

  it("does not include Version, Repository, Sync Completed, Sync Progress, or Quarantined", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildPackageOutput(buildPackageData())] } },
    });
    const details = getPackageMapper.getExecutionDetails(ctx);
    expect(details["Version"]).toBeUndefined();
    expect(details["Repository"]).toBeUndefined();
    expect(details["Sync Completed"]).toBeUndefined();
    expect(details["Sync Progress"]).toBeUndefined();
    expect(details["Quarantined"]).toBeUndefined();
  });

  it("shows status and stage fields", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildPackageOutput(buildPackageData())] } },
    });
    const details = getPackageMapper.getExecutionDetails(ctx);
    expect(details["Status"]).toBe("Available");
    expect(details["Stage"]).toBe("Fully Synchronised");
    expect(details["Security Scan"]).toBe("No Vulnerabilities Found");
  });

  it("uses self_webapp_url for the URL field", () => {
    const ctx = buildDetailsCtx({
      execution: {
        outputs: {
          default: [
            buildPackageOutput(
              buildPackageData({
                self_webapp_url: "https://app.cloudsmith.com/acme/r/production/docker/hello/",
                self_html_url: "https://cloudsmith.io/~acme/repos/production/packages/detail/",
              }),
            ),
          ],
        },
      },
    });
    const details = getPackageMapper.getExecutionDetails(ctx);
    expect(details["URL"]).toBe("https://app.cloudsmith.com/acme/r/production/docker/hello/");
  });

  it("omits URL when self_webapp_url is missing", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildPackageOutput(buildPackageData({ self_webapp_url: undefined }))] } },
    });
    const details = getPackageMapper.getExecutionDetails(ctx);
    expect(details["URL"]).toBeUndefined();
  });

  it("falls back to raw byte size when size_str is missing", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildPackageOutput(buildPackageData({ size_str: undefined, size: 2048 }))] } },
    });
    const details = getPackageMapper.getExecutionDetails(ctx);
    expect(details["Size"]).toBe("2048 bytes");
  });
});
