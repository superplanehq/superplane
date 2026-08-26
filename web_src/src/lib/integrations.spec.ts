import { describe, expect, it } from "vitest";

import { isCapabilityBasedIntegrationDefinition, usesHostedGitHubAppInstall } from "./integrations";

describe("usesHostedGitHubAppInstall", () => {
  it("is true only for GitHub with hostedAppInstall", () => {
    expect(usesHostedGitHubAppInstall({ name: "github", hostedAppInstall: true })).toBe(true);
    expect(usesHostedGitHubAppInstall({ name: "github", hostedAppInstall: false })).toBe(false);
    expect(usesHostedGitHubAppInstall({ name: "slack", hostedAppInstall: true })).toBe(false);
    expect(usesHostedGitHubAppInstall(undefined)).toBe(false);
  });
});

describe("isCapabilityBasedIntegrationDefinition", () => {
  it("follows legacySetupOnly", () => {
    expect(isCapabilityBasedIntegrationDefinition({ legacySetupOnly: false })).toBe(true);
    expect(isCapabilityBasedIntegrationDefinition({ legacySetupOnly: true })).toBe(false);
  });
});
