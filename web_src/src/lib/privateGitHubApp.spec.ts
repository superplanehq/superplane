import { beforeEach, describe, expect, it, vi } from "bun:test";

import { peekIntegrationSetupReturn } from "./integrationSetupReturn";

vi.mock("@/lib/browserAction", () => ({
  followBrowserAction: vi.fn(() => true),
}));
import {
  connectPrivateGitHubApp,
  githubPrivateAppSetupPath,
  privateGitHubAppCreateConfiguration,
  startPrivateGitHubAppSetup,
} from "./privateGitHubApp";

describe("privateGitHubAppCreateConfiguration", () => {
  it("sends privateApp for GitHub setup create", () => {
    expect(privateGitHubAppCreateConfiguration("github")).toEqual({ privateApp: true });
  });

  it("sends the GitHub Enterprise host and factory onboarding flag", () => {
    expect(
      privateGitHubAppCreateConfiguration("github", {
        host: "acme.ghe.com",
        factoryOnboarding: true,
        returnTo: "/org-1/workspaces/ws/setup?step=repo",
      }),
    ).toEqual({
      privateApp: true,
      host: "acme.ghe.com",
      factoryOnboarding: true,
      setupReturnPath: "/org-1/workspaces/ws/setup?step=repo",
    });
  });

  it("does not send privateApp for other integrations", () => {
    expect(privateGitHubAppCreateConfiguration("slack")).toBeUndefined();
  });
});

describe("githubPrivateAppSetupPath", () => {
  it("opens the GitHub setup wizard", () => {
    expect(githubPrivateAppSetupPath("org-1")).toBe("/org-1/settings/integrations/github/setup");
  });

  it("opens the GitHub setup wizard under a workspace integrations path", () => {
    expect(githubPrivateAppSetupPath("org-1", "/org-1/workspaces/pay/settings/organization/integrations")).toBe(
      "/org-1/workspaces/pay/settings/organization/integrations/github/setup",
    );
  });
});

describe("startPrivateGitHubAppSetup", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("stores the return path and opens the setup wizard", () => {
    const goTo = vi.fn();

    startPrivateGitHubAppSetup({
      organizationId: "org-1",
      returnTo: "/org-1/workspaces/ws/setup?step=vcs&pick=newest",
      goTo,
    });

    expect(goTo).toHaveBeenCalledWith("/org-1/settings/integrations/github/setup");
    expect(peekIntegrationSetupReturn("org-1")).toBe("/org-1/workspaces/ws/setup?step=vcs&pick=newest");
  });
});

describe("connectPrivateGitHubApp", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("opens the wizard when the setup flow feature is on", async () => {
    const goTo = vi.fn();
    const create = vi.fn();

    await connectPrivateGitHubApp({
      useWizard: true,
      organizationId: "org-1",
      goTo,
      existingNames: new Set(),
      connected: [],
      create,
    });

    expect(goTo).toHaveBeenCalledWith("/org-1/settings/integrations/github/setup");
    expect(create).not.toHaveBeenCalled();
  });

  it("creates with privateApp when the setup flow feature is off", async () => {
    const goTo = vi.fn();
    const create = vi.fn().mockResolvedValue({
      integration: { status: { browserAction: { method: "POST", url: "https://github.com/settings/apps/new" } } },
    });

    await connectPrivateGitHubApp({
      useWizard: false,
      organizationId: "org-1",
      goTo,
      existingNames: new Set(),
      connected: [],
      create,
    });

    expect(create).toHaveBeenCalledWith({
      integrationName: "github",
      name: "github",
      configuration: { privateApp: true },
    });
    expect(goTo).not.toHaveBeenCalled();
  });

  it("creates a private app for GitHub Enterprise and opens the wizard", async () => {
    const goTo = vi.fn();
    const onCreated = vi.fn();
    const create = vi.fn().mockResolvedValue({
      integration: { metadata: { id: "int-ghe" }, status: { setupState: { currentStep: "selectOwner" } } },
    });

    await connectPrivateGitHubApp({
      useWizard: true,
      organizationId: "org-1",
      host: "acme.ghe.com",
      factoryOnboarding: true,
      returnTo: "/org-1/workspaces/ws/setup?step=repo",
      goTo,
      existingNames: new Set(),
      connected: [],
      create,
      onCreated,
    });

    expect(create).toHaveBeenCalledWith({
      integrationName: "github",
      name: "github",
      configuration: {
        privateApp: true,
        host: "acme.ghe.com",
        factoryOnboarding: true,
        setupReturnPath: "/org-1/workspaces/ws/setup?step=repo",
      },
    });
    expect(onCreated).toHaveBeenCalledWith("int-ghe");
    expect(goTo).toHaveBeenCalledWith("/org-1/settings/integrations/github/setup?integrationId=int-ghe");
  });

  it("stays on the caller after factory onboarding creates the connection", async () => {
    const goTo = vi.fn();
    const onCreated = vi.fn();
    const create = vi.fn().mockResolvedValue({
      integration: { metadata: { id: "int-ghe" } },
    });

    const opened = await connectPrivateGitHubApp({
      useWizard: true,
      stayAfterCreate: true,
      organizationId: "org-1",
      host: "acme.ghe.com",
      factoryOnboarding: true,
      returnTo: "/org-1/workspaces/ws/setup?step=repo",
      goTo,
      existingNames: new Set(),
      connected: [],
      create,
      onCreated,
    });

    expect(opened).toBe(true);
    expect(onCreated).toHaveBeenCalledWith("int-ghe");
    expect(goTo).not.toHaveBeenCalled();
  });
});
