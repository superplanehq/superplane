import { describe, expect, it } from "vitest";
import { onSecurityScanCompletedTriggerRenderer } from "./on_security_scan_completed";

const event = {
  createdAt: new Date().toISOString(),
  data: {
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
  },
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
} as any;

describe("onSecurityScanCompletedTriggerRenderer", () => {
  it("derives a package title from the event", () => {
    expect(onSecurityScanCompletedTriggerRenderer.getTitleAndSubtitle({ event }).title).toBe("sp-compliance-gpl 1.0.0");
  });

  it("falls back to a generic title when no package name is present", () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const emptyEvent = { createdAt: event.createdAt, data: {} } as any;
    expect(onSecurityScanCompletedTriggerRenderer.getTitleAndSubtitle({ event: emptyEvent }).title).toBe(
      "Security scan",
    );
  });

  it("maps the scan results to root event values", () => {
    const values = onSecurityScanCompletedTriggerRenderer.getRootEventValues({ event });
    expect(values["Received At"]).toBeDefined();
    expect(values["Package"]).toBe("sp-compliance-gpl");
    expect(values["Repository"]).toBe("weskk/superplane-compliance");
    expect(values["Security Scan"]).toBe("2 Vulnerabilities Detected");
    expect(values["Vulnerabilities"]).toBe("2");
    expect(values["Max Severity"]).toBe("High");
  });
});
