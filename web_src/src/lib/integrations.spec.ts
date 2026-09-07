import { describe, expect, it } from "vitest";

import {
  isCapabilityBasedIntegrationDefinition,
  offersPrivateGitHubAppSetup,
  usesHostedGitHubAppInstall,
  usesHostedSentryInstall,
  usesPrivateGitHubAppWizard,
} from "./integrations";

describe("usesHostedGitHubAppInstall", () => {
  it("is true only for GitHub with hostedAppInstall", () => {
    expect(usesHostedGitHubAppInstall({ name: "github", hostedAppInstall: true })).toBe(true);
    expect(usesHostedGitHubAppInstall({ name: "github", hostedAppInstall: false })).toBe(false);
    expect(usesHostedGitHubAppInstall({ name: "slack", hostedAppInstall: true })).toBe(false);
    expect(usesHostedGitHubAppInstall(undefined)).toBe(false);
  });
});

describe("usesHostedSentryInstall", () => {
  it("is true only for Sentry with hostedAppInstall", () => {
    expect(usesHostedSentryInstall({ name: "sentry", hostedAppInstall: true })).toBe(true);
    expect(usesHostedSentryInstall({ name: "sentry", hostedAppInstall: false })).toBe(false);
    expect(usesHostedSentryInstall({ name: "github", hostedAppInstall: true })).toBe(false);
    expect(usesHostedSentryInstall(undefined)).toBe(false);
  });
});

describe("isCapabilityBasedIntegrationDefinition", () => {
  it("follows legacySetupOnly", () => {
    expect(isCapabilityBasedIntegrationDefinition({ legacySetupOnly: false })).toBe(true);
    expect(isCapabilityBasedIntegrationDefinition({ legacySetupOnly: true })).toBe(false);
  });
});

describe("offersPrivateGitHubAppSetup", () => {
  it("is true when hosted GitHub install is on", () => {
    expect(offersPrivateGitHubAppSetup({ name: "github", hostedAppInstall: true, legacySetupOnly: true })).toBe(true);
    expect(offersPrivateGitHubAppSetup({ name: "github", hostedAppInstall: false, legacySetupOnly: false })).toBe(
      false,
    );
    expect(offersPrivateGitHubAppSetup({ name: "slack", hostedAppInstall: true })).toBe(false);
  });
});

describe("usesPrivateGitHubAppWizard", () => {
  it("is true only when hosted install and the setup flow feature are on", () => {
    expect(usesPrivateGitHubAppWizard({ name: "github", hostedAppInstall: true, legacySetupOnly: false })).toBe(true);
    expect(usesPrivateGitHubAppWizard({ name: "github", hostedAppInstall: true, legacySetupOnly: true })).toBe(false);
  });
});
