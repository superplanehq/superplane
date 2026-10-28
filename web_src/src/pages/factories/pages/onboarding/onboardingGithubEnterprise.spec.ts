import { beforeEach, describe, expect, it, vi } from "bun:test";

import {
  bindGitHubEnterpriseOnboarding,
  githubEnterpriseOnboardingActions,
  githubOnboardingIntegrationId,
  persistGitHubEnterpriseRepository,
  readOnboardingGithubEnterprise,
  resolvedOnboardingGithubEnterprise,
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

describe("resolvedOnboardingGithubEnterprise", () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  it("restores Enterprise from the saved integration when the tab has no session", () => {
    expect(
      resolvedOnboardingGithubEnterprise({
        factoryId: "factory-1",
        vcsIntegrationId: "int-ghe",
        hostFromIntegration: "acme.ghe.com",
      }),
    ).toEqual({ enabled: true, host: "acme.ghe.com", integrationId: "int-ghe" });
  });

  it("keeps an explicit github.com choice for this session", () => {
    writeOnboardingGithubEnterprise("factory-1", false);
    expect(
      resolvedOnboardingGithubEnterprise({
        factoryId: "factory-1",
        vcsIntegrationId: "int-ghe",
        hostFromIntegration: "acme.ghe.com",
      }),
    ).toEqual({ enabled: false, host: "", integrationId: "" });
  });
});

describe("githubOnboardingIntegrationId", () => {
  it("prefers the session Enterprise integration over a leftover public selection", () => {
    expect(
      githubOnboardingIntegrationId({
        selectionId: "int-selected",
        vcsIntegrationId: "int-onboarding",
        rememberedIntegrationId: "int-session",
      }),
    ).toBe("int-session");
    expect(
      githubOnboardingIntegrationId({
        vcsIntegrationId: "int-onboarding",
        rememberedIntegrationId: "int-session",
      }),
    ).toBe("int-session");
    expect(
      githubOnboardingIntegrationId({
        selectionId: "int-selected",
        vcsIntegrationId: "int-onboarding",
      }),
    ).toBe("int-selected");
    expect(githubOnboardingIntegrationId({ vcsIntegrationId: "int-onboarding" })).toBe("int-onboarding");
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

  it("remembers the integration without locking the workspace provider", async () => {
    const rememberPreferredInstance = vi.fn();
    const refetchConnections = vi.fn().mockResolvedValue(undefined);
    const setSelections = vi.fn((update) => update({}));
    const selectVcsHost = vi.fn();

    await bindGitHubEnterpriseOnboarding({
      organizationId: "org-1",
      factoryId: "factory-1",
      integrationId: "int-ghe",
      host: "acme.ghe.com",
      rememberPreferredInstance,
      refetchConnections,
      setSelections,
      selectVcsHost,
    });

    expect(rememberPreferredInstance).toHaveBeenCalledWith("github", "int-ghe");
    expect(selectVcsHost).toHaveBeenCalledWith("github");
    expect(readOnboardingGithubEnterprise("factory-1")).toEqual({
      enabled: true,
      host: "acme.ghe.com",
      integrationId: "int-ghe",
    });
  });
});

describe("githubEnterpriseOnboardingActions", () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  it("saves the repository against the session Enterprise integration", async () => {
    writeOnboardingGithubEnterprise("factory-1", true, "acme.ghe.com", "int-ghe");
    const updateOnboarding = vi.fn().mockResolvedValue(undefined);
    const selectVcsHost = vi.fn();
    const selectRepo = vi.fn();
    const runSave = vi.fn(async (_setSaving: (saving: boolean) => void, action: () => Promise<unknown>) => {
      await action();
      return true;
    });

    const actions = githubEnterpriseOnboardingActions({
      organizationId: "org-1",
      factoryId: "factory-1",
      githubIntegrationId: "int-github-com",
      setSaving: vi.fn(),
      updateOnboarding,
      rememberPreferredInstance: vi.fn(),
      refetchConnections: vi.fn().mockResolvedValue(undefined),
      setSelections: vi.fn(),
      selectVcsHost,
      selectRepo,
      runSave,
    });

    await actions.selectGitHubEnterpriseRepository("acme/app");

    expect(updateOnboarding).toHaveBeenCalledWith({
      vcsIntegrationId: "int-ghe",
      appRepository: "acme/app",
      backlogRepository: "acme/app",
    });
    expect(selectRepo).toHaveBeenCalledWith("acme/app");
  });
});

describe("persistGitHubEnterpriseRepository", () => {
  it("saves the workspace GitHub binding when a repository is chosen", async () => {
    const updateOnboarding = vi.fn().mockResolvedValue(undefined);
    const selectVcsHost = vi.fn();
    const selectRepo = vi.fn();

    await persistGitHubEnterpriseRepository({
      fullName: "acme/app",
      githubIntegrationId: "int-ghe",
      updateOnboarding,
      selectVcsHost,
      selectRepo,
    });

    expect(updateOnboarding).toHaveBeenCalledWith({
      vcsIntegrationId: "int-ghe",
      appRepository: "acme/app",
      backlogRepository: "acme/app",
    });
    expect(selectVcsHost).toHaveBeenCalledWith("github");
    expect(selectRepo).toHaveBeenCalledWith("acme/app");
  });
});
