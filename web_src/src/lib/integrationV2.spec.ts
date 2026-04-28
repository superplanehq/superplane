import type { OrganizationsIntegration } from "@/api-client";
import { describe, expect, it } from "vitest";
import { getIntegrationV2SetupPath, integrationUsesNewSetupFlow, isIntegrationV2SetupEnabled } from "./integrationV2";

describe("isIntegrationV2SetupEnabled", () => {
  it("returns true for integrations using the new setup flow", () => {
    expect(isIntegrationV2SetupEnabled("github")).toBe(true);
    expect(isIntegrationV2SetupEnabled("semaphore")).toBe(true);
  });

  it("returns false for integrations outside the v2 whitelist", () => {
    expect(isIntegrationV2SetupEnabled("slack")).toBe(false);
    expect(isIntegrationV2SetupEnabled(undefined)).toBe(false);
    expect(isIntegrationV2SetupEnabled(null)).toBe(false);
  });
});

describe("getIntegrationV2SetupPath", () => {
  it("builds the expected route", () => {
    expect(getIntegrationV2SetupPath("org-1", "github")).toBe("/org-1/settings/integrations/github/setup");
  });
});

describe("integrationUsesNewSetupFlow", () => {
  it("returns true when the installation exposes capability state", () => {
    const integration: OrganizationsIntegration = {
      status: { capabilities: [{ name: "foo", state: "STATE_ENABLED" }] },
    };
    expect(integrationUsesNewSetupFlow(integration)).toBe(true);
  });

  it("returns false when capabilities are absent or empty", () => {
    expect(integrationUsesNewSetupFlow(undefined)).toBe(false);
    expect(integrationUsesNewSetupFlow(null)).toBe(false);
    expect(integrationUsesNewSetupFlow({})).toBe(false);
    expect(integrationUsesNewSetupFlow({ status: {} })).toBe(false);
    expect(integrationUsesNewSetupFlow({ status: { capabilities: [] } })).toBe(false);
  });
});
