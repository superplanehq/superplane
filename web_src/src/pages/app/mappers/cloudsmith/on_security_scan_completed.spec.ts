import { describe, expect, it } from "vitest";
import type { EventInfo, TriggerEventContext } from "../types";
import { onSecurityScanCompletedTriggerRenderer } from "./on_security_scan_completed";

function event(data: Record<string, unknown>): EventInfo {
  return {
    id: "evt-1",
    createdAt: new Date("2026-06-18T14:00:00Z").toISOString(),
    nodeId: "node-1",
    type: "cloudsmith.package.securityScanned",
    data,
  };
}

const scanData = {
  event: "package.security_scanned",
  namespace: "weskk",
  repository: "superplane-compliance",
  name: "sp-compliance-gpl",
  version: "1.0.0",
  slug_perm: "f3XvJCI9ufJa",
  format: "npm",
  security_scan_status: "2 Vulnerabilities Detected",
  has_vulnerabilities: true,
  max_severity: "High",
  num_vulnerabilities: 2,
};

describe("onSecurityScanCompletedTriggerRenderer", () => {
  it("derives a package title from the event", () => {
    const context: TriggerEventContext = { event: event(scanData) };
    expect(onSecurityScanCompletedTriggerRenderer.getTitleAndSubtitle(context).title).toBe("sp-compliance-gpl 1.0.0");
  });

  it("falls back to a generic title when no package name is present", () => {
    const context: TriggerEventContext = { event: event({}) };
    expect(onSecurityScanCompletedTriggerRenderer.getTitleAndSubtitle(context).title).toBe("Security scan");
  });

  it("maps the scan results to root event values", () => {
    const context: TriggerEventContext = { event: event(scanData) };
    const values = onSecurityScanCompletedTriggerRenderer.getRootEventValues(context);
    expect(values["Received At"]).toBeDefined();
    expect(values["Package"]).toBe("sp-compliance-gpl");
    expect(values["Repository"]).toBe("weskk/superplane-compliance");
    expect(values["Security Scan"]).toBe("2 Vulnerabilities Detected");
    expect(values["Vulnerabilities"]).toBe("2");
    expect(values["Max Severity"]).toBe("High");
  });
});
