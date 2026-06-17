import { describe, expect, it } from "vitest";
import { onComplianceCheckCompletedTriggerRenderer } from "./on_compliance_check_completed";

const event = {
  createdAt: new Date().toISOString(),
  data: {
    event: "package.synced",
    namespace: "weskk",
    repository: "superplane-compliance",
    name: "sp-compliance-gpl",
    version: "1.0.0",
    slug_perm: "f3XvJCI9ufJa",
    license: "GPL-3.0-only",
    osi_approved: true,
    policy_violated: false,
    is_quarantined: true,
    status: "Quarantined",
  },
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
} as any;

describe("onComplianceCheckCompletedTriggerRenderer", () => {
  it("derives a package title from the event", () => {
    const { title } = onComplianceCheckCompletedTriggerRenderer.getTitleAndSubtitle({ event });
    expect(title).toBe("sp-compliance-gpl 1.0.0");
  });

  it("falls back to a generic title when no package name is present", () => {
    const { title } = onComplianceCheckCompletedTriggerRenderer.getTitleAndSubtitle({
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      event: { createdAt: event.createdAt, data: {} } as any,
    });
    expect(title).toBe("Compliance check");
  });

  it("maps the compliance fields to root event values", () => {
    const values = onComplianceCheckCompletedTriggerRenderer.getRootEventValues({ event });
    expect(values["Package"]).toBe("sp-compliance-gpl");
    expect(values["Repository"]).toBe("weskk/superplane-compliance");
    expect(values["License"]).toBe("GPL-3.0-only");
    expect(values["OSI Approved"]).toBe("Yes");
    expect(values["Quarantined"]).toBe("Yes");
    expect(values["Policy Violated"]).toBe("No");
    expect(values["Status"]).toBe("Quarantined");
  });
});
