import { describe, expect, it } from "vitest";
import { getIntegrationV2SetupPath, isIntegrationV2SetupEnabled } from "./integrationV2";

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
