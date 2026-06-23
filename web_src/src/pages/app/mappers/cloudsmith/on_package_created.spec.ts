import { describe, expect, it } from "vitest";
import type { EventInfo, TriggerEventContext } from "../types";
import { onPackageCreatedTriggerRenderer } from "./on_package_created";

function event(data: Record<string, unknown>): EventInfo {
  return {
    id: "evt-1",
    createdAt: new Date("2026-06-18T14:00:00Z").toISOString(),
    nodeId: "node-1",
    type: "cloudsmith.package.created",
    data,
  };
}

const packageData = {
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
};

describe("onPackageCreatedTriggerRenderer", () => {
  it("derives a package title from the event", () => {
    const context: TriggerEventContext = { event: event(packageData) };
    expect(onPackageCreatedTriggerRenderer.getTitleAndSubtitle(context).title).toBe("sp-compliance-mit 1.0.0");
  });

  it("falls back to a generic title when no package name is present", () => {
    const context: TriggerEventContext = { event: event({}) };
    expect(onPackageCreatedTriggerRenderer.getTitleAndSubtitle(context).title).toBe("Package created");
  });

  it("maps the package fields to root event values", () => {
    const context: TriggerEventContext = { event: event(packageData) };
    const values = onPackageCreatedTriggerRenderer.getRootEventValues(context);
    expect(values["Received At"]).toBeDefined();
    expect(values["Package"]).toBe("sp-compliance-mit");
    expect(values["Repository"]).toBe("weskk/superplane-compliance");
    expect(values["Format"]).toBe("npm");
    expect(values["License"]).toBe("MIT");
    expect(values["Uploader"]).toBe("superplane-dnig");
    expect(values["Status"]).toBe("Completed");
  });
});
