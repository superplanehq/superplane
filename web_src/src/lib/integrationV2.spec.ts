import type { IntegrationsIntegrationDefinition, OrganizationsIntegration } from "@/api-client";
import { describe, expect, it } from "vitest";
import {
  getIntegrationV2SetupPath,
  integrationSupportsGuidedSetup,
  integrationUsesNewSetupFlow,
} from "./integrationV2";

describe("integrationSupportsGuidedSetup", () => {
  it("returns true when the catalog marks legacySetupOnly false", () => {
    const def: IntegrationsIntegrationDefinition = { name: "github", legacySetupOnly: false };
    expect(integrationSupportsGuidedSetup(def)).toBe(true);
  });

  it("returns false when legacy-only, undefined, or absent", () => {
    expect(integrationSupportsGuidedSetup({ name: "slack", legacySetupOnly: true })).toBe(false);
    expect(integrationSupportsGuidedSetup({ name: "slack" })).toBe(false);
    expect(integrationSupportsGuidedSetup(undefined)).toBe(false);
    expect(integrationSupportsGuidedSetup(null)).toBe(false);
  });
});

describe("getIntegrationV2SetupPath", () => {
  it("builds the expected route", () => {
    expect(getIntegrationV2SetupPath("org-1", "github")).toBe("/org-1/settings/integrations/github/setup");
  });
});

describe("integrationUsesNewSetupFlow", () => {
  it("returns true when legacySetup is false (new setup flow)", () => {
    const integration: OrganizationsIntegration = {
      status: { legacySetup: false, capabilities: [] },
    };
    expect(integrationUsesNewSetupFlow(integration)).toBe(true);
  });

  it("returns false when legacySetup is true even if capabilities are present", () => {
    const integration: OrganizationsIntegration = {
      status: { legacySetup: true, capabilities: [{ name: "foo", state: "STATE_ENABLED" }] },
    };
    expect(integrationUsesNewSetupFlow(integration)).toBe(false);
  });

  it("falls back to capability state when legacySetup is absent (older API)", () => {
    const integration: OrganizationsIntegration = {
      status: { capabilities: [{ name: "foo", state: "STATE_ENABLED" }] },
    };
    expect(integrationUsesNewSetupFlow(integration)).toBe(true);
  });

  it("returns false when capabilities are absent or empty and legacySetup is absent", () => {
    expect(integrationUsesNewSetupFlow(undefined)).toBe(false);
    expect(integrationUsesNewSetupFlow(null)).toBe(false);
    expect(integrationUsesNewSetupFlow({})).toBe(false);
    expect(integrationUsesNewSetupFlow({ status: {} })).toBe(false);
    expect(integrationUsesNewSetupFlow({ status: { capabilities: [] } })).toBe(false);
  });
});
