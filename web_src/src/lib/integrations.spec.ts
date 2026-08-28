import { describe, expect, it } from "vitest";

import {
  isCapabilityBasedIntegrationDefinition,
  offersPrivateGitHubAppSetup,
  usesHostedGitHubAppInstall,
} from "./integrations";

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

describe("offersPrivateGitHubAppSetup", () => {
  it("is true only when hosted GitHub install and the setup flow feature are on", () => {
    expect(offersPrivateGitHubAppSetup({ name: "github", hostedAppInstall: true, legacySetupOnly: false })).toBe(true);
    expect(offersPrivateGitHubAppSetup({ name: "github", hostedAppInstall: true, legacySetupOnly: true })).toBe(false);
    expect(offersPrivateGitHubAppSetup({ name: "github", hostedAppInstall: false, legacySetupOnly: false })).toBe(
      false,
    );
    expect(offersPrivateGitHubAppSetup({ name: "slack", hostedAppInstall: true, legacySetupOnly: false })).toBe(false);
  });
});
