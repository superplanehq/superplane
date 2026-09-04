import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  isOnboardingSetupReturnPath,
  pendingGitHubAccountPicker,
  pendingGitHubBrowserAction,
  pendingGitHubInstallPicker,
  startDirectGitHubConnect,
} from "./startDirectGitHubConnect";

const remember = vi.hoisted(() => vi.fn());
const follow = vi.hoisted(() => vi.fn(() => true));

vi.mock("@/lib/integrationSetupReturn", () => ({
  rememberIntegrationSetupReturn: remember,
  INTEGRATION_SETUP_STAY_PARAM: "setupStay",
}));

vi.mock("@/lib/browserAction", () => ({
  followBrowserAction: follow,
}));

describe("pendingGitHubBrowserAction", () => {
  it("returns the pending GitHub browser action", () => {
    const action = { method: "GET", url: "https://github.com/apps/superplane/installations/new" };
    expect(
      pendingGitHubBrowserAction(
        [
          {
            metadata: { integrationName: "github" },
            status: { state: "ready", browserAction: action, metadata: { startedByUserID: "user-1" } },
          },
          {
            metadata: { integrationName: "github" },
            status: { state: "pending", browserAction: action, metadata: { startedByUserID: "user-1" } },
          },
        ],
        "user-1",
      ),
    ).toEqual(action);
  });

  it("returns undefined when every GitHub connection is ready", () => {
    expect(
      pendingGitHubBrowserAction([{ metadata: { integrationName: "github" }, status: { state: "ready" } }], "user-1"),
    ).toBeUndefined();
  });

  it("returns undefined when the pending row belongs to a teammate", () => {
    expect(
      pendingGitHubBrowserAction(
        [
          {
            metadata: { integrationName: "github" },
            status: {
              state: "pending",
              browserAction: { method: "GET", url: "https://github.com/apps/superplane/installations/new" },
              metadata: { startedByUserID: "user-1" },
            },
          },
        ],
        "user-2",
      ),
    ).toBeUndefined();
  });

  it("reuses a pending action when the current user is not loaded yet", () => {
    const action = { method: "GET", url: "https://github.com/apps/superplane/installations/new" };
    expect(
      pendingGitHubBrowserAction([
        {
          metadata: { integrationName: "github" },
          status: { state: "pending", browserAction: action, metadata: { startedByUserID: "user-1" } },
        },
      ]),
    ).toEqual(action);
  });
});

describe("pendingGitHubInstallPicker", () => {
  it("returns a pending GitHub connection with two or more installs", () => {
    expect(
      pendingGitHubInstallPicker(
        [
          {
            metadata: { id: "int-1", integrationName: "github" },
            status: {
              state: "pending",
              metadata: {
                startedByUserID: "user-1",
                pendingInstallations: [
                  { id: "11", accountLogin: "acme" },
                  { id: "22", accountLogin: "octo" },
                ],
              },
            },
          },
        ],
        "user-1",
      ),
    ).toEqual({ id: "int-1" });
    expect(
      pendingGitHubAccountPicker(
        [
          {
            metadata: { id: "int-1", integrationName: "github" },
            status: {
              state: "pending",
              metadata: {
                startedByUserID: "user-1",
                state: "csrf",
                githubApp: { slug: "superplane" },
                pendingInstallations: [
                  { id: "11", accountLogin: "acme" },
                  { id: "22", accountLogin: "octo" },
                ],
              },
            },
          },
        ],
        "user-1",
      ),
    ).toEqual({
      id: "int-1",
      state: "csrf",
      appSlug: "superplane",
      installations: [
        { id: "11", accountLogin: "acme" },
        { id: "22", accountLogin: "octo" },
      ],
    });
  });

  it("returns undefined when there is no picker", () => {
    expect(
      pendingGitHubInstallPicker(
        [
          {
            metadata: { id: "int-1", integrationName: "github" },
            status: { state: "pending", browserAction: { method: "GET", url: "https://github.com" } },
          },
        ],
        "user-1",
      ),
    ).toBeUndefined();
  });

  it("returns undefined when the current user is not loaded yet", () => {
    expect(
      pendingGitHubAccountPicker([
        {
          metadata: { id: "int-1", integrationName: "github" },
          status: {
            state: "pending",
            metadata: {
              startedByUserID: "user-1",
              pendingInstallations: [
                { id: "11", accountLogin: "acme" },
                { id: "22", accountLogin: "octo" },
              ],
            },
          },
        },
      ]),
    ).toBeUndefined();
  });

  it("returns undefined when the picker belongs to a teammate", () => {
    expect(
      pendingGitHubInstallPicker(
        [
          {
            metadata: { id: "int-1", integrationName: "github" },
            status: {
              state: "pending",
              metadata: {
                startedByUserID: "user-1",
                pendingInstallations: [
                  { id: "11", accountLogin: "acme" },
                  { id: "22", accountLogin: "octo" },
                ],
              },
            },
          },
        ],
        "user-2",
      ),
    ).toBeUndefined();
  });
});

describe("isOnboardingSetupReturnPath", () => {
  it("accepts onboarding and workspace setup paths", () => {
    expect(isOnboardingSetupReturnPath("/onboarding?attempt=1&step=vcs")).toBe(true);
    expect(isOnboardingSetupReturnPath("/org-1/workspaces/ws/setup?step=vcs")).toBe(true);
    expect(isOnboardingSetupReturnPath("/org-1/settings/integrations")).toBe(false);
    expect(isOnboardingSetupReturnPath(undefined)).toBe(false);
  });
});

describe("startDirectGitHubConnect", () => {
  beforeEach(() => {
    remember.mockClear();
    follow.mockClear();
  });

  it("stays on onboarding when the account picker is already pending", async () => {
    const create = vi.fn();
    const goTo = vi.fn();

    const started = await startDirectGitHubConnect({
      organizationId: "org-1",
      returnTo: "/onboarding?attempt=1&step=vcs",
      existingNames: new Set(),
      connected: [
        {
          metadata: { id: "int-1", integrationName: "github" },
          status: {
            state: "pending",
            browserAction: { method: "GET", url: "https://github.com/login/oauth/authorize" },
            metadata: {
              startedByUserID: "user-1",
              pendingInstallations: [
                { id: "11", accountLogin: "acme" },
                { id: "22", accountLogin: "octo" },
              ],
            },
          },
        },
      ],
      currentUserId: "user-1",
      create,
      goTo,
    });

    expect(started).toBe(true);
    expect(create).not.toHaveBeenCalled();
    expect(follow).not.toHaveBeenCalled();
    expect(goTo).not.toHaveBeenCalled();
    expect(remember).toHaveBeenCalledWith("org-1", "/onboarding?attempt=1&step=vcs");
  });

  it("opens the picker page when pending installs exist", async () => {
    const create = vi.fn();
    const goTo = vi.fn();

    await startDirectGitHubConnect({
      organizationId: "org-1",
      returnTo: "/org-1/settings/integrations",
      existingNames: new Set(),
      connected: [
        {
          metadata: { id: "int-1", integrationName: "github" },
          status: {
            state: "pending",
            browserAction: { method: "GET", url: "https://github.com/login/oauth/authorize" },
            metadata: {
              startedByUserID: "user-1",
              pendingInstallations: [
                { id: "11", accountLogin: "acme" },
                { id: "22", accountLogin: "octo" },
              ],
            },
          },
        },
      ],
      currentUserId: "user-1",
      create,
      goTo,
    });

    expect(create).not.toHaveBeenCalled();
    expect(follow).not.toHaveBeenCalled();
    expect(remember).toHaveBeenCalledWith("org-1", "/org-1/settings/integrations");
    expect(goTo).toHaveBeenCalledWith("/org-1/settings/integrations/int-1?setupStay=1");
  });

  it("opens the picker under a factory integrations path", async () => {
    const create = vi.fn();
    const goTo = vi.fn();

    await startDirectGitHubConnect({
      organizationId: "org-1",
      returnTo: "/org-1/organization/integrations",
      integrationsBasePath: "/org-1/organization/integrations",
      existingNames: new Set(),
      connected: [
        {
          metadata: { id: "int-1", integrationName: "github" },
          status: {
            state: "pending",
            browserAction: { method: "GET", url: "https://github.com/login/oauth/authorize" },
            metadata: {
              startedByUserID: "user-1",
              pendingInstallations: [
                { id: "11", accountLogin: "acme" },
                { id: "22", accountLogin: "octo" },
              ],
            },
          },
        },
      ],
      currentUserId: "user-1",
      create,
      goTo,
    });

    expect(goTo).toHaveBeenCalledWith("/org-1/organization/integrations/int-1?setupStay=1");
  });

  it("reuses a pending browser action", async () => {
    const action = { method: "GET", url: "https://github.com/apps/superplane/installations/new" };
    const create = vi.fn();
    const update = vi.fn().mockResolvedValue(undefined);

    await startDirectGitHubConnect({
      organizationId: "org-1",
      returnTo: "/org-1/workspaces/ws/setup?step=vcs&pick=newest",
      existingNames: new Set(),
      connected: [
        {
          metadata: { id: "int-1", integrationName: "github" },
          status: { state: "pending", browserAction: action, metadata: { startedByUserID: "user-1" } },
        },
      ],
      currentUserId: "user-1",
      create,
      update,
    });

    expect(create).not.toHaveBeenCalled();
    expect(update).toHaveBeenCalledWith({
      id: "int-1",
      configuration: { setupReturnPath: "/org-1/workspaces/ws/setup?step=vcs&pick=newest" },
    });
    expect(remember).toHaveBeenCalledWith("org-1", "/org-1/workspaces/ws/setup?step=vcs&pick=newest");
    expect(follow).toHaveBeenCalledWith(action);
  });

  it("reuses a pending action when the current user is not loaded yet", async () => {
    const action = { method: "GET", url: "https://github.com/apps/superplane/installations/new" };
    const create = vi.fn();

    await startDirectGitHubConnect({
      organizationId: "org-1",
      returnTo: "/onboarding?attempt=1&step=vcs",
      existingNames: new Set(),
      connected: [
        {
          metadata: { id: "int-1", integrationName: "github" },
          status: { state: "pending", browserAction: action, metadata: { startedByUserID: "user-1" } },
        },
      ],
      create,
    });

    expect(create).not.toHaveBeenCalled();
    expect(follow).toHaveBeenCalledWith(action);
  });

  it("creates a new connection when forceNew is set", async () => {
    const action = { method: "GET", url: "https://github.com/apps/superplane/installations/new?state=3" };
    const create = vi.fn().mockResolvedValue({
      integration: { status: { browserAction: action } },
    });

    await startDirectGitHubConnect({
      organizationId: "org-1",
      existingNames: new Set(["github"]),
      connected: [
        {
          metadata: { id: "int-1", integrationName: "github" },
          status: {
            state: "pending",
            browserAction: { method: "GET", url: "https://github.com/login/oauth/authorize" },
            metadata: { startedByUserID: "user-1" },
          },
        },
      ],
      currentUserId: "user-1",
      forceNew: true,
      create,
    });

    expect(create).toHaveBeenCalledWith({ integrationName: "github", name: "github-2" });
    expect(follow).toHaveBeenCalledWith(action);
  });

  it("creates a new connection when the pending row belongs to a teammate", async () => {
    const action = { method: "GET", url: "https://github.com/apps/superplane/installations/new?state=2" };
    const create = vi.fn().mockResolvedValue({
      integration: { status: { browserAction: action } },
    });

    await startDirectGitHubConnect({
      organizationId: "org-1",
      existingNames: new Set(["github"]),
      connected: [
        {
          metadata: { id: "int-1", integrationName: "github" },
          status: {
            state: "pending",
            browserAction: { method: "GET", url: "https://github.com/login/oauth/authorize" },
            metadata: { startedByUserID: "user-1" },
          },
        },
      ],
      currentUserId: "user-2",
      create,
    });

    expect(create).toHaveBeenCalledWith({ integrationName: "github", name: "github-2" });
    expect(follow).toHaveBeenCalledWith(action);
  });

  it("creates a connection and follows the new browser action", async () => {
    const action = { method: "GET", url: "https://github.com/apps/superplane/installations/new?state=1" };
    const create = vi.fn().mockResolvedValue({
      integration: { status: { browserAction: action } },
    });

    await startDirectGitHubConnect({
      organizationId: "org-1",
      returnTo: "/org-1/settings/integrations",
      existingNames: new Set(["github"]),
      connected: [],
      create,
    });

    expect(create).toHaveBeenCalledWith({
      integrationName: "github",
      name: "github-2",
      configuration: { setupReturnPath: "/org-1/settings/integrations" },
    });
    expect(remember).toHaveBeenCalledWith("org-1", "/org-1/settings/integrations");
    expect(follow).toHaveBeenCalledWith(action);
  });

  it("throws when create does not return a browser action", async () => {
    const create = vi.fn().mockResolvedValue({
      integration: { status: { setupState: { currentStep: { name: "selectOwner" } } } },
    });

    await expect(
      startDirectGitHubConnect({
        organizationId: "org-1",
        existingNames: new Set(),
        connected: [],
        create,
      }),
    ).rejects.toThrow("The GitHub App install page did not open.");
  });
});
