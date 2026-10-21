import { beforeEach, describe, expect, it, vi } from "bun:test";

import {
  bindGitHubEnterpriseOnboarding,
  githubOnboardingIntegrationId,
  readOnboardingGithubEnterprise,
  writeOnboardingGithubEnterprise,
} from "./onboardingGithubEnterprise";

vi.mock("./githubIntegrationSelection", () => ({
  describeInstallationName: vi.fn(async () => "github"),
  selectionsWithSavedVcsInstallation: (
    onboarding: { vcsIntegrationId?: string; vcsProvider?: string },
    selections: Record<string, unknown>,
    installationName: string,
  ) => ({
    ...selections,
    github: { id: onboarding.vcsIntegrationId, name: installationName, ready: true },
  }),
}));

describe("onboarding GitHub Enterprise choice", () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  it("stores the enterprise choice, hostname, and integration", () => {
    writeOnboardingGithubEnterprise("factory-1", true, "acme.ghe.com", "int-ghe");
    expect(readOnboardingGithubEnterprise("factory-1")).toEqual({
      enabled: true,
      host: "acme.ghe.com",
      integrationId: "int-ghe",
    });
  });

  it("keeps the integration id when only the hostname is updated", () => {
    writeOnboardingGithubEnterprise("factory-1", true, "acme.ghe.com", "int-ghe");
    writeOnboardingGithubEnterprise("factory-1", true, "acme.ghe.com");
    expect(readOnboardingGithubEnterprise("factory-1").integrationId).toBe("int-ghe");
  });

  it("clears the choice for github.com", () => {
    writeOnboardingGithubEnterprise("factory-1", true, "acme.ghe.com", "int-ghe");
    writeOnboardingGithubEnterprise("factory-1", false);
    expect(readOnboardingGithubEnterprise("factory-1")).toEqual({ enabled: false, host: "", integrationId: "" });
  });
});

describe("githubOnboardingIntegrationId", () => {
  it("prefers the ready selection, then onboarding, then the remembered id", () => {
    expect(
      githubOnboardingIntegrationId({
        selectionId: "int-selected",
        vcsIntegrationId: "int-onboarding",
        rememberedIntegrationId: "int-session",
      }),
    ).toBe("int-selected");
    expect(
      githubOnboardingIntegrationId({
        vcsIntegrationId: "int-onboarding",
        rememberedIntegrationId: "int-session",
      }),
    ).toBe("int-onboarding");
    expect(githubOnboardingIntegrationId({ rememberedIntegrationId: "int-session" })).toBe("int-session");
  });

  it("ignores a Bitbucket onboarding integration", () => {
    expect(
      githubOnboardingIntegrationId({
        vcsProvider: "bitbucket",
        vcsIntegrationId: "int-bitbucket",
        rememberedIntegrationId: "int-session",
      }),
    ).toBe("");
  });
});

describe("bindGitHubEnterpriseOnboarding", () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  it("stores the integration and marks GitHub as the workspace host", async () => {
    const rememberPreferredInstance = vi.fn();
    const refetchConnections = vi.fn().mockResolvedValue(undefined);
    const setSelections = vi.fn((update) => update({}));
    const updateOnboarding = vi.fn().mockResolvedValue(undefined);
    const selectVcsHost = vi.fn();

    await bindGitHubEnterpriseOnboarding({
      organizationId: "org-1",
      factoryId: "factory-1",
      integrationId: "int-ghe",
      host: "acme.ghe.com",
      rememberPreferredInstance,
      refetchConnections,
      setSelections,
      updateOnboarding,
      selectVcsHost,
    });

    expect(rememberPreferredInstance).toHaveBeenCalledWith("github", "int-ghe");
    expect(updateOnboarding).toHaveBeenCalledWith({ vcsIntegrationId: "int-ghe" });
    expect(selectVcsHost).toHaveBeenCalledWith("github");
    expect(readOnboardingGithubEnterprise("factory-1")).toEqual({
      enabled: true,
      host: "acme.ghe.com",
      integrationId: "int-ghe",
    });
  });
});
