import { describe, expect, it } from "bun:test";

import {
  hiddenFieldsForHostedJira,
  isCapabilityBasedIntegrationDefinition,
  offersPrivateGitHubAppSetup,
  usesHostedGitHubAppInstall,
  usesHostedJiraOAuth,
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

describe("usesHostedJiraOAuth", () => {
  it("is true only for Jira with hostedAppInstall", () => {
    expect(usesHostedJiraOAuth({ name: "jira", hostedAppInstall: true })).toBe(true);
    expect(usesHostedJiraOAuth({ name: "jira", hostedAppInstall: false })).toBe(false);
    expect(usesHostedJiraOAuth({ name: "github", hostedAppInstall: true })).toBe(false);
    expect(usesHostedJiraOAuth(undefined)).toBe(false);
  });
});

describe("hiddenFieldsForHostedJira", () => {
  it("hides Client ID and Client Secret for hosted Jira", () => {
    expect(hiddenFieldsForHostedJira({ name: "jira", hostedAppInstall: true }, [])).toEqual([
      "clientId",
      "clientSecret",
    ]);
  });

  it("keeps caller hidden fields and adds the credential fields", () => {
    expect(hiddenFieldsForHostedJira({ name: "jira", hostedAppInstall: true }, ["enableOpsFeatures"])).toEqual([
      "enableOpsFeatures",
      "clientId",
      "clientSecret",
    ]);
  });

  it("leaves hidden fields unchanged when hosted Jira is off", () => {
    expect(hiddenFieldsForHostedJira({ name: "jira", hostedAppInstall: false }, ["adminKey"])).toEqual(["adminKey"]);
    expect(hiddenFieldsForHostedJira({ name: "github", hostedAppInstall: true }, ["adminKey"])).toEqual(["adminKey"]);
    expect(hiddenFieldsForHostedJira(undefined, ["adminKey"])).toEqual(["adminKey"]);
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
