import { describe, expect, it } from "vitest";
import { onPackageCreatedTriggerRenderer } from "./on_package_created";

const event = {
  createdAt: new Date().toISOString(),
  data: {
    event: "package.created",
    namespace: "weskk",
    repository: "superplane-compliance",
    name: "sp-compliance-mit",
    version: "1.0.0",
    slug_perm: "wxu9RDqPfCj0",
    format: "npm",
    license: "MIT",
    uploader: "superplane-dnig",
    uploaded_at: "2026-06-17T14:50:00Z",
    status: "Completed",
  },
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
} as any;

describe("onPackageCreatedTriggerRenderer", () => {
  it("derives a package title from the event", () => {
    expect(onPackageCreatedTriggerRenderer.getTitleAndSubtitle({ event }).title).toBe("sp-compliance-mit 1.0.0");
  });

  it("falls back to a generic title when no package name is present", () => {
    expect(
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      onPackageCreatedTriggerRenderer.getTitleAndSubtitle({ event: { createdAt: event.createdAt, data: {} } as any })
        .title,
    ).toBe("Package created");
  });

  it("maps the package fields to root event values", () => {
    const values = onPackageCreatedTriggerRenderer.getRootEventValues({ event });
    expect(values["Received At"]).toBeDefined();
    expect(values["Package"]).toBe("sp-compliance-mit");
    expect(values["Repository"]).toBe("weskk/superplane-compliance");
    expect(values["Format"]).toBe("npm");
    expect(values["License"]).toBe("MIT");
    expect(values["Uploader"]).toBe("superplane-dnig");
    expect(values["Status"]).toBe("Completed");
  });
});
