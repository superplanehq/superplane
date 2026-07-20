import { describe, expect, it } from "vitest";
import { getPackageVulnerabilitiesMapper } from "./get_package_vulnerabilities";
import { buildDetailsCtx, buildPackageOutput } from "./test_helpers";
import type { VulnerabilityScanResult } from "./types";

function buildScanResult(overrides?: Partial<VulnerabilityScanResult>): VulnerabilityScanResult {
  return {
    identifier: "1ceRAXarsZ93o5b7",
    created_at: "2026-06-18T07:08:34.479287Z",
    package: {
      identifier: "YFf7Vw1SnOnK",
      name: "hello-go-app",
      version: "cd8e0196c8cfe78b87690ec03900b775c7823d32f09ec8f87f760411059de7e2",
      url: "https://api.cloudsmith.io/v1/packages/acme/production/YFf7Vw1SnOnK/",
    },
    scan_id: null,
    has_vulnerabilities: true,
    num_vulnerabilities: 27,
    max_severity: "Critical",
    ...overrides,
  };
}

describe("getPackageVulnerabilitiesMapper.getExecutionDetails", () => {
  it("does not throw when outputs is undefined", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: undefined } });
    expect(() => getPackageVulnerabilitiesMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("does not throw when default array is empty", () => {
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [] } } });
    expect(() => getPackageVulnerabilitiesMapper.getExecutionDetails(ctx)).not.toThrow();
  });

  it("returns only Executed At when scan result has no identifier", () => {
    const ctx = buildDetailsCtx({
      execution: { outputs: { default: [buildPackageOutput({})] } },
    });
    const details = getPackageVulnerabilitiesMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Vulnerabilities Found"]).toBeUndefined();
  });

  it("shows vulnerabilities found, total, max severity and scanned at", () => {
    const result = buildScanResult();
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildPackageOutput(result)] } } });
    const details = getPackageVulnerabilitiesMapper.getExecutionDetails(ctx);
    expect(details["Executed At"]).toBeDefined();
    expect(details["Vulnerabilities Found"]).toBe("Yes");
    expect(details["Total"]).toBe("27");
    expect(details["Max Severity"]).toBe("Critical");
    expect(details["Scanned At"]).toBeDefined();
  });

  it("shows no vulnerabilities when scan is clean", () => {
    const result = buildScanResult({
      has_vulnerabilities: false,
      num_vulnerabilities: 0,
      max_severity: undefined,
    });
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildPackageOutput(result)] } } });
    const details = getPackageVulnerabilitiesMapper.getExecutionDetails(ctx);
    expect(details["Vulnerabilities Found"]).toBe("No");
    expect(details["Total"]).toBe("0");
    expect(details["Max Severity"]).toBeUndefined();
  });

  it("omits Max Severity when not present", () => {
    const result = buildScanResult({ max_severity: undefined });
    const ctx = buildDetailsCtx({ execution: { outputs: { default: [buildPackageOutput(result)] } } });
    const details = getPackageVulnerabilitiesMapper.getExecutionDetails(ctx);
    expect(details["Max Severity"]).toBeUndefined();
  });
});
